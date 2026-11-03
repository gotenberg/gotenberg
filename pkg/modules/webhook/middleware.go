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
						// An events URL alone would be ignored, and the client
						// would get the output in the response instead.
						if c.Request().Header.Get("Gotenberg-Webhook-Events-Url") != "" {
							return api.WrapError(
								errors.New("empty webhook URL"),
								api.NewSentinelHttpError(http.StatusBadRequest, "The 'Gotenberg-Webhook-Url' header is required with 'Gotenberg-Webhook-Events-Url'"),
							)
						}

						// No webhook URL, call the next middleware in the chain.
						return next(c)
					}

					// A route that answers JSON directly has no output file to
					// deliver: refuse the request rather than answer 204 and
					// report the failure to the events URL.
					if api.RouteAnswersDirectly(c) {
						return api.WrapError(
							errors.New("webhook on a route without output files"),
							api.NewSentinelHttpError(
								http.StatusBadRequest,
								"The 'Gotenberg-Webhook-Url' and 'Gotenberg-Webhook-Events-Url' headers do not apply to this route: it answers JSON directly, which a webhook cannot deliver. Send the request without them",
							),
						)
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

					// The filters let non-HTTP schemes through, for Chromium's sake.
					// A webhook only speaks HTTP.
					err := requireWebURL("Gotenberg-Webhook-Url", webhookUrl)
					if err != nil {
						return err
					}

					err = requireWebURL("Gotenberg-Webhook-Events-Url", webhookEventsUrl)
					if err != nil {
						return err
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
					err = gotenberg.FilterOutboundURL(ctx, webhookUrl, w.allowList, w.denyList, deadline, ipOpts...)
					if err != nil {
						return fmt.Errorf("filter webhook URL: %w", unresolvableHostError("Gotenberg-Webhook-Url", err))
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
						return fmt.Errorf("filter webhook events URL: %w", unresolvableHostError("Gotenberg-Webhook-Events-Url", err))
					}

					// Retrieve values from [echo.Context] before it gets recycled.
					// See https://github.com/gotenberg/gotenberg/issues/1000.
					startTime := c.Get("startTime").(time.Time)
					correlationIdHeader := c.Get("correlationIdHeader").(string)
					correlationId := c.Get("correlationId").(string)

					// Each attempt of a delivery gets its own HTTP client span.
					httpClient := gotenberg.NewOutboundHttpClient(w.clientTimeout, w.allowList, w.denyList, w.enableEnvironmentProxy, ipOpts...)
					httpClient.Transport = gotenberg.NewTracingTransport(httpClient.Transport)

					client := &client{
						url:              webhookUrl,
						method:           webhookMethod,
						eventsUrl:        webhookEventsUrl,
						extraHttpHeaders: extraHttpHeaders,
						startTime:        startTime,
						deliveryTimeout:  w.deliveryTimeout(),

						client: &retryablehttp.Client{
							HTTPClient:   httpClient,
							RetryMax:     w.maxRetry,
							RetryWaitMin: w.retryMinWait,
							RetryWaitMax: w.retryMaxWait,
							Logger:       gotenberg.NewLeveledLogger(ctx.Log()),
							CheckRetry:   retryablehttp.DefaultRetryPolicy,
							// Not DefaultBackoff: it returns a remote Retry-After
							// verbatim, ignoring --webhook-retry-max-wait (env
							// WEBHOOK_RETRY_MAX_WAIT).
							Backoff:      gotenberg.ClampedBackoff,
							ErrorHandler: giveUp,
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

// unresolvableHostError answers a URL of header whose host does not resolve
// with a 400 naming both, as a resolution failure reveals nothing of the
// filters. It returns any other error, a policy refusal among them, as is: the
// client gets a generic 403 for those. The answer wraps the resolver's error
// alone, so that its 400 is not read as a filtered URL's 403.
func unresolvableHostError(header string, err error) error {
	var unresolvable *gotenberg.UnresolvableHostError
	if !errors.As(err, &unresolvable) {
		return err
	}

	return api.WrapError(
		fmt.Errorf("resolve host %q: %w", unresolvable.Host, unresolvable.Err),
		api.NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid '%s' header: the host '%s' does not resolve", header, unresolvable.Host)),
	)
}

// requireWebURL answers a URL of header that is no absolute http or https URL
// with a host with a 400 naming both. Neither the filters nor the URL, which
// may hold credentials, appear in it.
func requireWebURL(header, rawURL string) error {
	err := gotenberg.RequireWebURL(rawURL)
	if err == nil {
		return nil
	}

	return api.WrapError(
		fmt.Errorf("check %s: %w", header, err),
		api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Invalid '%s' header: want an absolute URL with an http or https scheme and a host, such as https://example.com/upload", header),
		),
	)
}
