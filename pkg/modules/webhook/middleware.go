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

					headers := map[string]string{
						echo.HeaderContentType:     http.DetectContentType(fileHeader),
						echo.HeaderContentLength:   strconv.FormatInt(fileStat.Size(), 10),
						params.correlationIdHeader: params.correlationId,
					}
					_, ok := params.extraHttpHeaders[echo.HeaderContentDisposition]
					if !ok {
						headers[echo.HeaderContentDisposition] = fmt.Sprintf("attachment; filename=%q", params.ctx.OutputFilename(params.outputPath))
					}

					err = params.client.send(params.ctx, bufio.NewReader(outputFile), headers, false)
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

				return func(c echo.Context) error {
					webhookUrl := c.Request().Header.Get("Gotenberg-Webhook-Url")
					if webhookUrl == "" {
						// No webhook URL, call the next middleware in the chain.
						return next(c)
					}

					ctx := c.Get("context").(*api.Context)
					cancel := c.Get("cancel").(context.CancelFunc)

					// Do we have a webhook error URL and/or an events URL?
					// At least one must be provided.
					webhookErrorUrl := c.Request().Header.Get("Gotenberg-Webhook-Error-Url")
					webhookEventsUrl := c.Request().Header.Get("Gotenberg-Webhook-Events-Url")

					if webhookErrorUrl == "" && webhookEventsUrl == "" {
						return api.WrapError(
							errors.New("empty webhook error URL and events URL"),
							api.NewSentinelHttpError(http.StatusBadRequest, "At least one of 'Gotenberg-Webhook-Error-Url' or 'Gotenberg-Webhook-Events-Url' headers must be provided"),
						)
					}

					if webhookErrorUrl != "" {
						ctx.Log().Warn("'Gotenberg-Webhook-Error-Url' header is deprecated, use 'Gotenberg-Webhook-Events-Url' instead")
					}

					deadline, ok := ctx.Deadline()
					if !ok {
						return errors.New("context has no deadline")
					}

					// Let's check if the webhook URLs are acceptable according to our
					// allowed/denied lists, and against the IP-based outbound URL
					// guard. See [gotenberg.FilterOutboundURL].
					err := gotenberg.FilterOutboundURL(ctx, webhookUrl, w.allowList, w.denyList, deadline)
					if err != nil {
						return fmt.Errorf("filter webhook URL: %w", err)
					}

					if webhookErrorUrl != "" {
						err = gotenberg.FilterOutboundURL(ctx, webhookErrorUrl, w.errorAllowList, w.errorDenyList, deadline)
						if err != nil {
							return fmt.Errorf("filter webhook error URL: %w", err)
						}
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

					var webhookErrorMethod string
					if webhookErrorUrl != "" {
						webhookErrorMethod, err = methodFromHeader("Gotenberg-Webhook-Error-Method")
						if err != nil {
							return fmt.Errorf("get method to use for webhook error: %w", err)
						}
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

					// Filter the events URL if provided.
					if webhookEventsUrl != "" {
						err = gotenberg.FilterOutboundURL(ctx, webhookEventsUrl, w.allowList, w.denyList, deadline)
						if err != nil {
							return fmt.Errorf("filter webhook events URL: %w", err)
						}
					}

					// Retrieve values from echo.Context before it gets recycled.
					// See https://github.com/gotenberg/gotenberg/issues/1000.
					startTime := c.Get("startTime").(time.Time)
					correlationIdHeader := c.Get("correlationIdHeader").(string)
					correlationId := c.Get("correlationId").(string)

					client := &client{
						url:              webhookUrl,
						method:           webhookMethod,
						errorUrl:         webhookErrorUrl,
						errorMethod:      webhookErrorMethod,
						eventsUrl:        webhookEventsUrl,
						extraHttpHeaders: extraHttpHeaders,
						startTime:        startTime,

						client: &retryablehttp.Client{
							HTTPClient:   gotenberg.NewOutboundHttpClient(w.clientTimeout, w.allowList, w.denyList),
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

						headers := map[string]string{
							echo.HeaderContentType: echo.MIMEApplicationJSON,
							correlationIdHeader:    correlationId,
						}

						err = client.send(ctx, bytes.NewReader(b), headers, true)
						if err != nil {
							ctx.Log().Error(fmt.Sprintf("send error response to webhook: %s", err.Error()))
						}

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
							ctx:                 ctx,
							outputPath:          outputPath,
							extraHttpHeaders:    extraHttpHeaders,
							correlationIdHeader: correlationIdHeader,
							correlationId:       correlationId,
							client:              client,
							handleError:         handleError,
						})
						return c.NoContent(http.StatusNoContent)
					}

					if deadline, ok := ctx.Deadline(); ok {
						// Create a new context derived from Background (detached from Request)
						// but with the same deadline as the original context.
						detachedCtx, detachedCancel := context.WithDeadline(context.Background(), deadline)

						// Replace the embedded context in the api.Context struct.
						// The modules downstream will now use this detached context.
						ctx.Context = detachedCtx

						// We must wrap the cancel function.
						// 1. detachedCancel() cleans up our new detached context.
						// 2. originalCancel() (captured from c.Get("cancel")) cleans up the working directory.
						originalCancel := cancel
						cancel = func() {
							detachedCancel()
							originalCancel()
						}
					} else {
						// Fallback if no deadline was set (rare, as newContext enforces it).
						ctx.Context = context.Background()
					}

					// As a webhook URL has been given, we handle the request in a
					// goroutine and return immediately.
					//
					// Echo returns the echo.Context back to its sync.Pool as
					// soon as this synchronous handler returns ErrAsyncProcess.
					// A concurrent request can then claim the recycled context
					// and c.Reset() wipes the shared store, which would cause
					// any c.Get("...").(T) assertion downstream of the webhook
					// goroutine to panic on a nil value and crash the process.
					// Snapshot the keys downstream reads onto a detached
					// wrapper before spawning the goroutine so pool reuse
					// cannot reach into our async work.
					detached := newPoolSafeContext(c, "logger", "context", "correlationId", "correlationIdHeader", "startTime")

					w.asyncCount.Add(1)
					go func() {
						defer cancel()
						defer w.asyncCount.Add(-1)

						// Defense in depth: any panic that escapes the
						// downstream chain (including future regressions of
						// the pool-reuse bug) routes through handleError and
						// leaves the process running.
						defer func() {
							r := recover()
							if r == nil {
								return
							}
							ctx.Log().Error(fmt.Sprintf("webhook goroutine panic: %v", r))
							handleError(fmt.Errorf("internal error: %v", r))
						}()

						// Call the next middleware in the chain.
						err := next(detached)
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
							ctx:                 ctx,
							outputPath:          outputPath,
							extraHttpHeaders:    extraHttpHeaders,
							correlationIdHeader: correlationIdHeader,
							correlationId:       correlationId,
							client:              client,
							handleError:         handleError,
						})
					}()

					return api.ErrAsyncProcess
				}
			}
		}(),
	}
}
