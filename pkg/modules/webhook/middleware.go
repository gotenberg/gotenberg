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

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

type sendOutputFileParams struct {
	ctx              *api.Context
	outputPath       string
	extraHttpHeaders map[string]string
	headers          http.Header
	client           *client
	handleError      func(error)
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

					params.headers.Set(echo.HeaderContentType, http.DetectContentType(fileHeader))
					params.headers.Set(echo.HeaderContentLength, strconv.FormatInt(fileStat.Size(), 10))

					_, ok := params.extraHttpHeaders[echo.HeaderContentDisposition]
					if !ok {
						params.headers.Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", params.ctx.OutputFilename(params.outputPath)))
					}

					err = params.client.send(bufio.NewReader(outputFile), params.headers, false)
					if err != nil {
						params.ctx.Log().Error(fmt.Sprintf("send output file to webhook: %s", err))
						params.handleError(err)
					}
				}

				return func(c echo.Context) error {
					webhookUrl := c.Request().Header.Get("Gotenberg-Webhook-Url")
					if webhookUrl == "" {
						// No webhook URL, call the next middleware in the chain.
						return next(c)
					}

					ctx := c.Get("context").(*api.Context)
					cancel := c.Get("cancel").(context.CancelFunc)

					// Do we have a webhook error URL in case of... error?
					webhookErrorUrl := c.Request().Header.Get("Gotenberg-Webhook-Error-Url")
					if webhookErrorUrl == "" {
						return api.WrapError(
							errors.New("empty webhook error URL"),
							api.NewSentinelHttpError(http.StatusBadRequest, "Invalid 'Gotenberg-Webhook-Error-Url' header: empty value or header not provided"),
						)
					}

					deadline, ok := ctx.Deadline()
					if !ok {
						return errors.New("context has no deadline")
					}

					// Let's check if the webhook URLs are acceptable according to our
					// allowed/denied lists.
					err := gotenberg.FilterDeadline(w.allowList, w.denyList, webhookUrl, deadline)
					if err != nil {
						return fmt.Errorf("filter webhook URL: %w", err)
					}

					err = gotenberg.FilterDeadline(w.errorAllowList, w.errorDenyList, webhookErrorUrl, deadline)
					if err != nil {
						return fmt.Errorf("filter webhook error URL: %w", err)
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

					webhookErrorMethod, err := methodFromHeader("Gotenberg-Webhook-Error-Method")
					if err != nil {
						return fmt.Errorf("get method to use for webhook error: %w", err)
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

					// Retrieve values from echo.Context before it gets recycled.
					// See https://github.com/gotenberg/gotenberg/issues/1000.
					startTime := c.Get("startTime").(time.Time)
					traceHeader := c.Get("traceHeader").(string)
					trace := c.Get("trace").(string)

					var headers http.Header
					w.tracer.Inject(ctx, headers)
					headers.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
					headers.Set(traceHeader, trace)

					client := &client{
						url:              webhookUrl,
						method:           webhookMethod,
						errorUrl:         webhookErrorUrl,
						errorMethod:      webhookErrorMethod,
						extraHttpHeaders: extraHttpHeaders,
						startTime:        startTime,

						client: &retryablehttp.Client{
							HTTPClient: &http.Client{
								Timeout: w.clientTimeout,
							},
							RetryMax:     w.maxRetry,
							RetryWaitMin: w.retryMinWait,
							RetryWaitMax: w.retryMaxWait,
							Logger:       gotenberg.NewLeveledLogger(ctx.Log()),
							CheckRetry:   retryablehttp.DefaultRetryPolicy,
							Backoff:      retryablehttp.DefaultBackoff,
						},
						logger: ctx.Log(),
					}

					// This method parses an "asynchronous" error and sends a
					// request to the webhook error URL with a JSON body
					// containing the status and the error message.
					handleError := func(err error) {
						status, message := api.ParseError(err)

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

						err = client.send(bytes.NewReader(b), headers, true)
						if err != nil {
							ctx.Log().Error(fmt.Sprintf("send error response to webhook: %s", err.Error()))
						}
					}

					if w.enableSyncMode {
						err := next(c)
						if err != nil {
							if errors.Is(err, api.ErrNoOutputFile) {
								errNoOutputFile := fmt.Errorf("%w - the webhook middleware cannot handle the result of this route", err)
								handleError(api.WrapError(
									errNoOutputFile,
									api.NewSentinelHttpError(
										http.StatusBadRequest,
										"The webhook middleware can only work with multipart/form-data routes that results in output files",
									),
								))
								return nil
							}
							ctx.Log().Error(err.Error())
							handleError(err)
							return nil
						}

						outputPath, err := ctx.BuildOutputFile()
						if err != nil {
							ctx.Log().Error(fmt.Sprintf("build output file: %s", err))
							handleError(err)
							return nil
						}
						// No error, let's send the output file to the webhook URL.
						sendOutputFile(sendOutputFileParams{
							ctx:              ctx,
							outputPath:       outputPath,
							extraHttpHeaders: extraHttpHeaders,
							headers:          headers,
							client:           client,
							handleError:      handleError,
						})
						return c.NoContent(http.StatusNoContent)
					}
					// As a webhook URL has been given, we handle the request in a
					// goroutine and return immediately.
					w.asyncCount.Add(1)
					go func() {
						defer cancel()
						defer w.asyncCount.Add(-1)

						// Call the next middleware in the chain.
						err := next(c)
						if err != nil {
							if errors.Is(err, api.ErrNoOutputFile) {
								errNoOutputFile := fmt.Errorf("%w - the webhook middleware cannot handle the result of this route", err)
								handleError(api.WrapError(
									errNoOutputFile,
									api.NewSentinelHttpError(
										http.StatusBadRequest,
										"The webhook middleware can only work with multipart/form-data routes that results in output files",
									),
								))
								return
							}
							// The process failed for whatever reason. Let's send the
							// details to the webhook.
							ctx.Log().Error(err.Error())
							handleError(err)
							return
						}

						// No error, let's get to build the output file.
						outputPath, err := ctx.BuildOutputFile()
						if err != nil {
							ctx.Log().Error(fmt.Sprintf("build output file: %s", err))
							handleError(err)
							return
						}

						sendOutputFile(sendOutputFileParams{
							ctx:              ctx,
							outputPath:       outputPath,
							extraHttpHeaders: extraHttpHeaders,
							headers:          headers,
							client:           client,
							handleError:      handleError,
						})
					}()

					return api.ErrAsyncProcess
				}
			}
		}(),
	}
}
