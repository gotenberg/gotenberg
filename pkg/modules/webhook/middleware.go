package webhook

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func webhookMiddleware(w *Webhook) api.Middleware {
	return api.Middleware{
		Stack: api.MultipartStack,
		Handler: func() echo.MiddlewareFunc {
			return func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					webhookUrl := c.Request().Header.Get("Gotenberg-Webhook-Url")
					if webhookUrl == "" {
						return next(c)
					}

					ctx := c.Get("context").(*api.Context)
					cancel := c.Get("cancel").(context.CancelFunc)

					deadline, ok := ctx.Deadline()
					if !ok {
						return errors.New("context has no deadline")
					}

					// Parse all configurations.
					cfg, err := parseWebhookConfig(c, w, deadline, webhookUrl)
					if err != nil {
						return err
					}

					// Set up Headers and Tracing.
					startTime := c.Get("startTime").(time.Time)
					correlationIdHeader := c.Get("correlationIdHeader").(string)
					correlationId := c.Get("correlationId").(string)

					headers := make(http.Header)
					headers.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
					headers.Set(correlationIdHeader, correlationId)
					otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))

					// Initialize the Webhook Client.
					webhookClient := &client{
						url:              cfg.URL,
						method:           cfg.Method,
						errorUrl:         cfg.ErrorURL,
						errorMethod:      cfg.ErrorMethod,
						extraHttpHeaders: cfg.ExtraHTTPHeaders,
						startTime:        startTime,
						client: &retryablehttp.Client{
							HTTPClient:   &http.Client{Timeout: w.clientTimeout},
							RetryMax:     w.maxRetry,
							RetryWaitMin: w.retryMinWait,
							RetryWaitMax: w.retryMaxWait,
							Logger:       gotenberg.NewLeveledLogger(ctx.Log()),
							CheckRetry:   retryablehttp.DefaultRetryPolicy,
							Backoff:      retryablehttp.DefaultBackoff,
						},
						logger: ctx.Log(),
					}

					handleErrFunc := func(err error) {
						sendWebhookError(ctx, webhookClient, headers, err)
					}

					// Execute Sync Flow.
					if w.enableSyncMode {
						return handleSyncWebhook(c, next, ctx, webhookClient, headers, cfg.ExtraHTTPHeaders, handleErrFunc)
					}

					// Execute Async Flow.
					return handleAsyncWebhook(c, next, w, ctx, cancel, deadline, webhookClient, headers, cfg.ExtraHTTPHeaders, handleErrFunc)
				}
			}
		}(),
	}
}

func handleSyncWebhook(c echo.Context, next echo.HandlerFunc, ctx *api.Context, client *client, headers http.Header, extraHeaders map[string]string, handleErr func(error)) error {
	if err := next(c); err != nil {
		if errors.Is(err, api.ErrNoOutputFile) {
			handleErr(api.WrapError(
				fmt.Errorf("%w - the webhook middleware cannot handle the result of this route", err),
				api.NewSentinelHttpError(http.StatusBadRequest, "The webhook middleware can only work with multipart/form-data routes that results in output files"),
			))
			return nil
		}
		ctx.Log().Error(err.Error())
		handleErr(err)
		return nil
	}

	outputPath, err := ctx.BuildOutputFile()
	if err != nil {
		ctx.Log().Error(fmt.Sprintf("build output file: %s", err))
		handleErr(err)
		return nil
	}

	sendOutputFile(sendOutputFileParams{
		ctx:              ctx,
		outputPath:       outputPath,
		extraHttpHeaders: extraHeaders,
		headers:          headers,
		client:           client,
		handleError:      handleErr,
	})
	return c.NoContent(http.StatusNoContent)
}

func handleAsyncWebhook(c echo.Context, next echo.HandlerFunc, w *Webhook, ctx *api.Context, cancel context.CancelFunc, deadline time.Time, client *client, headers http.Header, extraHeaders map[string]string, handleErr func(error)) error {
	// Detach context for async processing.
	detachedCtx := context.WithoutCancel(ctx.Context)
	detachedCtx, detachedCancel := context.WithDeadline(detachedCtx, deadline)
	ctx.Context = detachedCtx

	originalCancel := cancel
	cancel = func() {
		detachedCancel()
		originalCancel()
	}

	w.asyncCount.Add(1)
	go func() {
		defer cancel()
		defer w.asyncCount.Add(-1)

		if err := next(c); err != nil {
			if errors.Is(err, api.ErrNoOutputFile) {
				handleErr(api.WrapError(
					fmt.Errorf("%w - the webhook middleware cannot handle the result of this route", err),
					api.NewSentinelHttpError(http.StatusBadRequest, "The webhook middleware can only work with multipart/form-data routes that results in output files"),
				))
				return
			}
			ctx.Log().Error(err.Error())
			handleErr(err)
			return
		}

		outputPath, err := ctx.BuildOutputFile()
		if err != nil {
			ctx.Log().Error(fmt.Sprintf("build output file: %s", err))
			handleErr(err)
			return
		}

		sendOutputFile(sendOutputFileParams{
			ctx:              ctx,
			outputPath:       outputPath,
			extraHttpHeaders: extraHeaders,
			headers:          headers,
			client:           client,
			handleError:      handleErr,
		})
	}()

	return api.ErrAsyncProcess
}

type webhookConfig struct {
	URL              string
	ErrorURL         string
	Method           string
	ErrorMethod      string
	ExtraHTTPHeaders map[string]string
}

func parseWebhookConfig(c echo.Context, w *Webhook, deadline time.Time, webhookUrl string) (*webhookConfig, error) {
	errorUrl := c.Request().Header.Get("Gotenberg-Webhook-Error-Url")
	if errorUrl == "" {
		return nil, api.WrapError(
			errors.New("empty webhook error URL"),
			api.NewSentinelHttpError(http.StatusBadRequest, "Invalid 'Gotenberg-Webhook-Error-Url' header: empty value or header not provided"),
		)
	}

	if err := gotenberg.FilterDeadline(w.allowList, w.denyList, webhookUrl, deadline); err != nil {
		return nil, fmt.Errorf("filter webhook URL: %w", err)
	}
	if err := gotenberg.FilterDeadline(w.errorAllowList, w.errorDenyList, errorUrl, deadline); err != nil {
		return nil, fmt.Errorf("filter webhook error URL: %w", err)
	}

	method, err := methodFromHeader(c, "Gotenberg-Webhook-Method")
	if err != nil {
		return nil, fmt.Errorf("get method to use for webhook: %w", err)
	}

	errorMethod, err := methodFromHeader(c, "Gotenberg-Webhook-Error-Method")
	if err != nil {
		return nil, fmt.Errorf("get method to use for webhook error: %w", err)
	}

	var extraHeaders map[string]string
	if extraJson := c.Request().Header.Get("Gotenberg-Webhook-Extra-Http-Headers"); extraJson != "" {
		if err := json.Unmarshal([]byte(extraJson), &extraHeaders); err != nil {
			return nil, api.WrapError(
				fmt.Errorf("unmarshal webhook extra HTTP headers: %w", err),
				api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'Gotenberg-Webhook-Extra-Http-Headers' header value: %s", err.Error())),
			)
		}
	}

	return &webhookConfig{
		URL:              webhookUrl,
		ErrorURL:         errorUrl,
		Method:           method,
		ErrorMethod:      errorMethod,
		ExtraHTTPHeaders: extraHeaders,
	}, nil
}

func methodFromHeader(c echo.Context, header string) (string, error) {
	method := c.Request().Header.Get(header)
	if method == "" {
		return http.MethodPost, nil
	}

	method = strings.ToUpper(method)
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut:
		return method, nil
	}

	return "", api.WrapError(
		fmt.Errorf("webhook method '%s' is not '%s', '%s' or '%s'", method, http.MethodPost, http.MethodPatch, http.MethodPut),
		api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid '%s' header value: expected '%s', '%s' or '%s', but got '%s'", header, http.MethodPost, http.MethodPatch, http.MethodPut, method)),
	)
}

func sendWebhookError(ctx *api.Context, c *client, headers http.Header, processErr error) {
	status, message := api.ParseError(processErr)

	body := struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}{
		Status:  status,
		Message: message,
	}

	b, err := json.Marshal(body)
	if err != nil {
		ctx.Log().Error(fmt.Sprintf("marshal JSON: %s", err.Error()))
		return
	}

	headers.Set(echo.HeaderContentLength, strconv.Itoa(len(b)))

	if err := c.send(ctx, bytes.NewReader(b), headers, true); err != nil {
		ctx.Log().Error(fmt.Sprintf("send error response to webhook: %s", err.Error()))
	}
}

type sendOutputFileParams struct {
	ctx              *api.Context
	outputPath       string
	extraHttpHeaders map[string]string
	headers          http.Header
	client           *client
	handleError      func(error)
}

func sendOutputFile(params sendOutputFileParams) {
	outputFile, err := os.Open(params.outputPath)
	if err != nil {
		params.ctx.Log().Error(fmt.Sprintf("open output file: %s", err))
		params.handleError(err)
		return
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			params.ctx.Log().Error(fmt.Sprintf("close output file: %s", err))
		}
	}()

	fileHeader := make([]byte, 512)
	_, err = outputFile.Read(fileHeader)
	if err != nil {
		params.ctx.Log().Error(fmt.Sprintf("read header of output file: %s", err))
		params.handleError(err)
		return
	}

	fileStat, err := outputFile.Stat()
	if err != nil {
		params.ctx.Log().Error(fmt.Sprintf("get stat from output file: %s", err))
		params.handleError(err)
		return
	}

	_, err = outputFile.Seek(0, 0)
	if err != nil {
		params.ctx.Log().Error(fmt.Sprintf("reset output file reader: %s", err))
		params.handleError(err)
		return
	}

	params.headers.Set(echo.HeaderContentType, http.DetectContentType(fileHeader))
	params.headers.Set(echo.HeaderContentLength, strconv.FormatInt(fileStat.Size(), 10))

	_, ok := params.extraHttpHeaders[echo.HeaderContentDisposition]
	if !ok {
		params.headers.Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", params.ctx.OutputFilename(params.outputPath)))
	}

	err = params.client.send(params.ctx, bufio.NewReader(outputFile), params.headers, false)
	if err != nil {
		params.ctx.Log().Error(fmt.Sprintf("send output file to webhook: %s", err))
		params.handleError(err)
	}
}
