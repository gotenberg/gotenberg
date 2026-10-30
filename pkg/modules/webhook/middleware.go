package webhook

import (
	"bufio"
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
	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

type sendOutputFileParams struct {
	ctx                 *api.Context
	outputPath          string
	extraHttpHeaders    map[string]string
	correlationIdHeader string
	correlationId       string
	client              *client
	handleError         func(error)
}

func webhookMiddleware(w *Webhook) api.Middleware {
	return api.Middleware{
		Stack: api.MultipartStack,
		Handler: func() echo.MiddlewareFunc {
			return func(next echo.HandlerFunc) echo.HandlerFunc {
				sendOutputFile := func(params sendOutputFileParams) {
					outputFile, err := os.Open(params.outputPath)
					if err != nil {
						params.ctx.Log().Error(fmt.Sprintf("open output file: %s", err))
						params.handleError(err)
						return
					}
					defer func() {
						err := outputFile.Close()
						if err != nil {
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

					headers := params.ctx.OutputHeaders()
					if headers == nil {
						headers = make(map[string]string)
					}
					headers[echo.HeaderContentType] = http.DetectContentType(fileHeader)
					headers[echo.HeaderContentLength] = strconv.FormatInt(fileStat.Size(), 10)
					headers[params.correlationIdHeader] = params.correlationId
					_, ok := params.extraHttpHeaders[echo.HeaderContentDisposition]
					if !ok {
						headers[echo.HeaderContentDisposition] = fmt.Sprintf("attachment; filename=%q", params.ctx.OutputFilename(params.outputPath))
					}

					err = params.client.send(params.ctx, bufio.NewReader(outputFile), headers)
					if err != nil {
						params.ctx.Log().Error(fmt.Sprintf("send output file to webhook: %s", err))
						params.handleError(err)
						return
					}

					params.client.sendEvent(params.ctx, params.correlationIdHeader, params.correlationId, map[string]any{
						"event":         "webhook.success",
						"correlationId": params.correlationId,
						"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
					})
				}

				return func(c *echo.Context) error {
					webhookUrl := c.Request().Header.Get("Gotenberg-Webhook-Url")
					if webhookUrl == "" {
						// No webhook URL, call the next middleware in the chain.
						return next(c)
					}

					ctx := c.Get("context").(*api.Context)
					cancel := c.Get("cancel").(context.CancelFunc)

					// The events URL receives the outcome, including errors.
					webhookEventsUrl := c.Request().Header.Get("Gotenberg-Webhook-Events-Url")
					if webhookEventsUrl == "" {
						return api.WrapError(
							errors.New("empty webhook events URL"),
							api.NewSentinelHttpError(http.StatusBadRequest, "The 'Gotenberg-Webhook-Events-Url' header is required with 'Gotenberg-Webhook-Url'"),
						)
					}

					deadline, ok := ctx.Deadline()
					if !ok {
						return errors.New("context has no deadline")
					}

					// Let's check if the webhook URLs are acceptable according to our
					// allowed/denied lists, and against the IP-class options.
					// See [gotenberg.FilterOutboundURL].
					ipOpts := []gotenberg.DecideOption{
						gotenberg.WithDenyPrivateIPs(w.denyPrivateIPs),
						gotenberg.WithDenyPublicIPs(w.denyPublicIPs),
					}
					err := gotenberg.FilterOutboundURL(ctx, webhookUrl, w.allowList, w.denyList, deadline, ipOpts...)
					if err != nil {
						return fmt.Errorf("filter webhook URL: %w", err)
					}

					// Let's check the HTTP methods for calling the webhook URLs.
					methodFromHeader := func(header string) (string, error) {
						method := c.Request().Header.Get(header)

						if method == "" {
							return http.MethodPost, nil
						}

						method = strings.ToUpper(method)

						switch method {
						case http.MethodPost:
							return method, nil
						case http.MethodPatch:
							return method, nil
						case http.MethodPut:
							return method, nil
						}

						return "", api.WrapError(
							fmt.Errorf("webhook method '%s' is not '%s', '%s' or '%s'", method, http.MethodPost, http.MethodPatch, http.MethodPut),
							api.NewSentinelHttpError(
								http.StatusBadRequest,
								fmt.Sprintf("Invalid '%s' header value: expected '%s', '%s' or '%s', but got '%s'", header, http.MethodPost, http.MethodPatch, http.MethodPut, method),
							),
						)
					}

					webhookMethod, err := methodFromHeader("Gotenberg-Webhook-Method")
					if err != nil {
						return fmt.Errorf("get method to use for webhook: %w", err)
					}

					// What about extra HTTP headers?
					var extraHttpHeaders map[string]string

					extraHttpHeadersJson := c.Request().Header.Get("Gotenberg-Webhook-Extra-Http-Headers")
					if extraHttpHeadersJson != "" {
						err = json.Unmarshal([]byte(extraHttpHeadersJson), &extraHttpHeaders)
						if err != nil {
							return api.WrapError(
								fmt.Errorf("unmarshal webhook extra HTTP headers: %w", err),
								api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'Gotenberg-Webhook-Extra-Http-Headers' header value: %s", err.Error())),
							)
						}
					}

					err = gotenberg.FilterOutboundURL(ctx, webhookEventsUrl, w.allowList, w.denyList, deadline, ipOpts...)
					if err != nil {
						return fmt.Errorf("filter webhook events URL: %w", err)
					}

					// Retrieve values from [echo.Context] before it gets recycled.
					// See https://github.com/gotenberg/gotenberg/issues/1000.
					startTime := c.Get("startTime").(time.Time)
					correlationIdHeader := c.Get("correlationIdHeader").(string)
					correlationId := c.Get("correlationId").(string)

					client := &client{
						url:              webhookUrl,
						method:           webhookMethod,
						eventsUrl:        webhookEventsUrl,
						extraHttpHeaders: extraHttpHeaders,
						startTime:        startTime,
						deliveryTimeout:  w.deliveryTimeout(),

						client: &retryablehttp.Client{
							HTTPClient:   gotenberg.NewOutboundHttpClient(w.clientTimeout, w.allowList, w.denyList, w.enableEnvironmentProxy, ipOpts...),
							RetryMax:     w.maxRetry,
							RetryWaitMin: w.retryMinWait,
							RetryWaitMax: w.retryMaxWait,
							Logger:       gotenberg.NewLeveledLogger(ctx.Log()),
							CheckRetry:   retryablehttp.DefaultRetryPolicy,
							// Not DefaultBackoff: it returns a remote Retry-After
							// verbatim, ignoring --webhook-retry-max-wait (env
							// WEBHOOK_RETRY_MAX_WAIT).
							Backoff: gotenberg.ClampedBackoff,
						},
						logger: ctx.Log(),
					}

					// This method parses an "asynchronous" error and sends it to
					// the events URL with its status and message.
					handleError := func(err error) {
						status, message := api.ParseError(err)

						client.sendEvent(ctx, correlationIdHeader, correlationId, map[string]any{
							"event":         "webhook.error",
							"correlationId": correlationId,
							"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
							"error": map[string]any{
								"status":  status,
								"message": message,
							},
						})
					}

					// The downstream chain writes into a detached response, never
					// the client's: the client only ever gets the 204 below, and the
					// outcome goes to the webhook URLs. See newDetachedContext.
					detached := newDetachedContext(c, "logger", "context", "correlationId", "correlationIdHeader", "startTime")

					process := func() {
						err := next(detached)
						if errors.Is(err, api.ErrNoOutputFile) {
							handleError(api.WrapError(
								fmt.Errorf("%w - the webhook middleware cannot handle the result of this route", err),
								api.NewSentinelHttpError(
									http.StatusBadRequest,
									"The webhook middleware can only work with multipart/form-data routes that results in output files",
								),
							))
							return
						}
						if err != nil {
							ctx.Log().Error(err.Error())
							handleError(err)
							return
						}

						outputPath, err := ctx.BuildOutputFile()
						if err != nil {
							ctx.Log().Error(fmt.Sprintf("build output file: %s", err))
							handleError(err)
							return
						}

						sendOutputFile(sendOutputFileParams{
							ctx:                 ctx,
							outputPath:          outputPath,
							extraHttpHeaders:    extraHttpHeaders,
							correlationIdHeader: correlationIdHeader,
							correlationId:       correlationId,
							client:              client,
							handleError:         handleError,
						})
					}

					if w.enableSyncMode {
						process()

						err = c.NoContent(http.StatusNoContent)
						if err != nil {
							return fmt.Errorf("send response: %w", err)
						}

						// The response is out: contextMiddleware must only clean up.
						return api.ErrNoOutputFile
					}

					cancel = detachAsyncContext(ctx, cancel)

					w.asyncCount.Add(1)
					go func() {
						defer cancel()
						defer w.asyncCount.Add(-1)

						// Defense in depth: a panic escaping the downstream chain
						// reaches the events URL and leaves the process running.
						defer func() {
							r := recover()
							if r == nil {
								return
							}
							ctx.Log().Error(fmt.Sprintf("webhook goroutine panic: %v", r))
							handleError(fmt.Errorf("internal error: %v", r))
						}()

						process()
					}()

					return api.ErrAsyncProcess
				}
			}
		}(),
	}
}
