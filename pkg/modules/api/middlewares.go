package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// httpErrorHandler is the centralized HTTP error handler. It parses the error,
// returns either a response as "text/plain; charset=UTF-8" or, if a webhook
// client exists in the echo.Context, sends a request to the webhook error URL
// with a JSON body containing the trace, the status and the error message.
func httpErrorHandler(traceHeader string) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		parseError := func(err error) (int, string) {
			echoErr, ok := err.(*echo.HTTPError)
			if ok {
				return echoErr.Code, http.StatusText(echoErr.Code)
			}

			if errors.Is(err, context.DeadlineExceeded) {
				return http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable)
			}

			var httpErr HTTPError
			if errors.As(err, &httpErr) {
				return httpErr.HTTPError()
			}

			// Default 500 status code.
			return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
		}

		status, message := parseError(err)

		logger := c.Get("logger").(*zap.Logger)
		clientOrNil := c.Get("webhookClient")

		// No webhook client, meaning we can send the error as a response.
		if clientOrNil == nil {
			c.Response().Header().Add(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)

			err = c.String(status, message)

			if err != nil {
				logger.Error(fmt.Sprintf("send error response: %s", err.Error()))
			}

			return
		}

		// We have to send the error to the webhook.
		client := clientOrNil.(*webhookClient)

		body := struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		}{
			Status:  status,
			Message: message,
		}

		b, err := json.Marshal(body)
		if err != nil {
			logger.Error(fmt.Sprintf("marshal JSON: %s", err.Error()))

			return
		}

		headers := map[string]string{
			echo.HeaderContentType: echo.MIMEApplicationJSONCharsetUTF8,
			traceHeader:            c.Get("trace").(string),
		}

		err = client.send(bytes.NewReader(b), headers, true)
		if err != nil {
			logger.Error(fmt.Sprintf("send error response to webhook: %s", err.Error()))
		}
	}
}

// latencyMiddleware sets the start time in the echo.Context under "startTime".
// Its value will be used later to calculate a request latency.
//
//  startTime := c.Get("startTime").(time.Time)
func latencyMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// First piece for calculating the latency.
			startTime := time.Now()
			c.Set("startTime", startTime)

			// Call the next middleware in the chain.
			return next(c)
		}
	}
}

// rootPathMiddleware sets the root path in the echo.Context under "rootPath".
// Its value may be used to skip a middleware execution based on a request
// URI.
//
//  rootPath := c.Get("rootPath").(string)
//  healthURI := fmt.Sprintf("%shealth", rootPath)
//
//  // Skip the middleware if health check URI.
//  if c.Request().RequestURI == healthURI {
//    // Call the next middleware in the chain.
//    return next(c)
//  }
func rootPathMiddleware(rootPath string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("rootPath", rootPath)

			// Call the next middleware in the chain.
			return next(c)
		}
	}
}

// traceMiddleware sets the request identifier in the echo.Context under
// "trace". Its value is either retrieved from the trace header or generated if
// the header is not present / its value is empty.
//
//  trace := c.Get("trace").(string)
func traceMiddleware(header string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get or create the request identifier.
			trace := c.Request().Header.Get(header)

			if trace == "" {
				trace = uuid.New().String()
			}

			c.Set("trace", trace)
			c.Response().Header().Add(header, trace)

			// Call the next middleware in the chain.
			return next(c)
		}
	}
}

// loggerMiddleware sets the logger in the echo.Context under "logger" and logs
// a request result (but does not log a webhook call result, which is the job
// of the webhookClient).
//
//  logger := c.Get("logger").(*zap.Logger)
func loggerMiddleware(logger *zap.Logger, skipHealthRouteLogging bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			startTime := c.Get("startTime").(time.Time)
			trace := c.Get("trace").(string)

			// Create the request logger and add it to our locals.
			reqLogger := logger.With(zap.String("trace", trace))
			c.Set("logger", reqLogger)

			// Call the next middleware in the chain.
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			if skipHealthRouteLogging {
				rootPath := c.Get("rootPath").(string)
				healthURI := fmt.Sprintf("%shealth", rootPath)

				if c.Request().RequestURI == healthURI {
					return nil
				}
			}

			// Last piece for calculating the latency.
			finishTime := time.Now()

			// Now, let's log!
			fields := make([]zap.Field, 12)
			fields[0] = zap.String("remote_ip", c.RealIP())
			fields[1] = zap.String("host", c.Request().Host)
			fields[2] = zap.String("uri", c.Request().RequestURI)
			fields[3] = zap.String("method", c.Request().Method)
			fields[4] = zap.String("path", func() string {
				path := c.Request().URL.Path

				if path == "" {
					path = "/"
				}

				return path
			}())
			fields[5] = zap.String("referer", c.Request().Referer())
			fields[6] = zap.String("user_agent", c.Request().UserAgent())
			fields[7] = zap.Int("status", c.Response().Status)
			fields[8] = zap.Int64("latency", int64(finishTime.Sub(startTime)))
			fields[9] = zap.String("latency_human", finishTime.Sub(startTime).String())
			fields[10] = zap.Int64("bytes_in", c.Request().ContentLength)
			fields[11] = zap.Int64("bytes_out", c.Response().Size)

			if err != nil {
				reqLogger.Error(err.Error(), fields...)
			} else {
				reqLogger.Info("request handled", fields...)
			}

			return nil
		}
	}
}

type contextMiddlewareConfig struct {
	traceHeader string
	timeout     struct {
		process time.Duration
		write   time.Duration
	}
	webhook struct {
		allowList      *regexp.Regexp
		denyList       *regexp.Regexp
		errorAllowList *regexp.Regexp
		errorDenyList  *regexp.Regexp
		maxRetry       int
		retryMinWait   time.Duration
		retryMaxWait   time.Duration
		disable        bool
	}
}

// contextMiddleware handles the result of a "multipart/form-data" request. If
// a webhook URL is present in the headers, exit early and process the result
// in a goroutine.
func contextMiddleware(cfg contextMiddlewareConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			webhookURL := c.Request().Header.Get("Gotenberg-Webhook-Url")
			logger := c.Get("logger").(*zap.Logger).With(zap.Bool("webhook", webhookURL != ""))

			// We create a context with a timeout so that underlying processes are
			// able to stop early and handle correctly a timeout scenario.
			ctx, cancel, err := newContext(c, logger, cfg.timeout.process)
			if err != nil {
				cancel()

				return fmt.Errorf("create request context: %w", err)
			}
			c.Set("context", ctx)

			// Helper function for retrieving/creating the output filename.
			outputFilename := func(outputPath string) string {
				filename := c.Request().Header.Get("Gotenberg-Output-Filename")

				if filename == "" {
					return filepath.Base(outputPath)
				}

				return fmt.Sprintf("%s%s", filename, filepath.Ext(outputPath))
			}

			if webhookURL == "" {
				defer cancel()

				// No webhook URL, call the next middleware in the chain.
				err := next(c)

				if err != nil {
					return err
				}

				// No error, let's build the output file.
				outputPath, err := ctx.buildOutputFile()
				if err != nil {
					return fmt.Errorf("build output file: %w", err)
				}

				// Send the output file.
				err = c.Attachment(outputPath, outputFilename(outputPath))
				if err != nil {
					return fmt.Errorf("send response: %w", err)
				}

				return nil
			}

			// Ok, we got a webhook URL.

			if cfg.webhook.disable {
				// The client requested the webhook feature, but it has been
				// disabled. Let's tell the client about that.
				cancel()

				return WrapError(
					errors.New("webhook feature requested but it is disabled"),
					NewSentinelHTTPError(http.StatusForbidden, "Invalid 'Gotenberg-Webhook-Url' header: feature is disabled"),
				)
			}

			// Do we have a webhook error URL in case of... error?
			webhookErrorURL := c.Request().Header.Get("Gotenberg-Webhook-Error-Url")

			if webhookErrorURL == "" {
				cancel()

				return WrapError(
					errors.New("empty webhook error URL"),
					NewSentinelHTTPError(http.StatusBadRequest, "Invalid 'Gotenberg-Webhook-Error-Url' header: empty value or header not provided"),
				)
			}

			// Let's check if the webhook URLs are acceptable according to our
			// allowed/denied lists.
			filter := func(URL, header string, allowList, denyList *regexp.Regexp) error {
				if !allowList.MatchString(URL) {
					return WrapError(
						fmt.Errorf("'%s' does not match the expression from the allowed list", URL),
						NewSentinelHTTPError(
							http.StatusForbidden,
							fmt.Sprintf("Invalid '%s' header value: '%s' does not match the authorized URLs", header, URL),
						),
					)
				}

				if denyList.String() != "" && denyList.MatchString(URL) {
					return WrapError(
						fmt.Errorf("'%s' matches the expression from the denied list", URL),
						NewSentinelHTTPError(
							http.StatusForbidden,
							fmt.Sprintf("Invalid '%s' header value: '%s' does not match the authorized URLs", header, URL),
						),
					)
				}

				return nil
			}

			err = filter(webhookURL, "Gotenberg-Webhook-Url", cfg.webhook.allowList, cfg.webhook.denyList)
			if err != nil {
				cancel()

				return fmt.Errorf("filter webhook URL: %w", err)
			}

			err = filter(webhookErrorURL, "Gotenberg-Webhook-Error-Url", cfg.webhook.errorAllowList, cfg.webhook.errorDenyList)
			if err != nil {
				cancel()

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

				return "", WrapError(
					fmt.Errorf("webhook method '%s' is not '%s', '%s' or '%s'", method, http.MethodPost, http.MethodPatch, http.MethodPut),
					NewSentinelHTTPError(
						http.StatusBadRequest,
						fmt.Sprintf("Invalid '%s' header value: expected '%s', '%s' or '%s', but got '%s'", header, http.MethodPost, http.MethodPatch, http.MethodPut, method),
					),
				)
			}

			webhookMethod, err := methodFromHeader("Gotenberg-Webhook-Method")
			if err != nil {
				cancel()

				return fmt.Errorf("get method to use for webhook: %w", err)
			}

			webhookErrorMethod, err := methodFromHeader("Gotenberg-Webhook-Error-Method")
			if err != nil {
				cancel()

				return fmt.Errorf("get method to use for webhook error: %w", err)
			}

			// What about extra HTTP headers?
			var extraHTTPHeaders map[string]string

			extraHTTPHeadersJSON := c.Request().Header.Get("Gotenberg-Webhook-Extra-Http-Headers")
			if extraHTTPHeadersJSON != "" {
				err = json.Unmarshal([]byte(extraHTTPHeadersJSON), &extraHTTPHeaders)
				if err != nil {
					cancel()

					return WrapError(
						fmt.Errorf("unmarshal webhook extra HTTP headers: %w", err),
						NewSentinelHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid 'Gotenberg-Webhook-Extra-Http-Headers' header value: %s", err.Error())),
					)
				}
			}

			client := &webhookClient{
				url:              webhookURL,
				method:           webhookMethod,
				errorURL:         webhookErrorURL,
				errorMethod:      webhookErrorMethod,
				extraHTTPHeaders: extraHTTPHeaders,
				startTime:        c.Get("startTime").(time.Time),

				client: &retryablehttp.Client{
					HTTPClient: &http.Client{
						Timeout: cfg.timeout.write,
					},
					RetryMax:     cfg.webhook.maxRetry,
					RetryWaitMin: cfg.webhook.retryMinWait,
					RetryWaitMax: cfg.webhook.retryMaxWait,
					Logger: leveledLogger{
						logger: logger,
					},
					CheckRetry: retryablehttp.DefaultRetryPolicy,
					Backoff:    retryablehttp.DefaultBackoff,
				},
				logger: logger,
			}

			c.Set("webhookClient", client)

			// As a webhook URL has been given, we handle the request in a
			// goroutine and return immediately.
			go func() {
				defer cancel()

				// Call the next middleware in the chain.
				err := next(c)

				if err != nil {
					// The process failed for whatever reason. Let's send the
					// details to the webhook.
					ctx.Log().Error(err.Error())
					c.Error(err)

					return
				}

				// No error, let's get build the output file.
				outputPath, err := ctx.buildOutputFile()
				if err != nil {
					ctx.Log().Error(fmt.Sprintf("build output file: %s", err))
					c.Error(err)

					return
				}

				outputFile, err := os.Open(outputPath)
				if err != nil {
					ctx.Log().Error(fmt.Sprintf("open output file: %s", err))
					c.Error(err)

					return
				}

				defer func() {
					err := outputFile.Close()
					if err != nil {
						ctx.Log().Error(fmt.Sprintf("close output file: %s", err))
					}
				}()

				fileHeader := make([]byte, 512)
				_, err = outputFile.Read(fileHeader)
				if err != nil {
					ctx.Log().Error(fmt.Sprintf("read header of output file: %s", err))
					c.Error(err)

					return
				}

				fileStat, err := outputFile.Stat()
				if err != nil {
					ctx.Log().Error(fmt.Sprintf("get stat from output file: %s", err))
					c.Error(err)

					return
				}

				_, err = outputFile.Seek(0, 0)
				if err != nil {
					ctx.Log().Error(fmt.Sprintf("reset output file reader: %s", err))
					c.Error(err)

					return
				}

				headers := map[string]string{
					echo.HeaderContentDisposition: fmt.Sprintf("attachement; filename=%q", outputFilename(outputPath)),
					echo.HeaderContentType:        http.DetectContentType(fileHeader),
					echo.HeaderContentLength:      strconv.FormatInt(fileStat.Size(), 10),
					cfg.traceHeader:               c.Get("trace").(string),
				}

				// Send the output file to the webhook.
				err = client.send(bufio.NewReader(outputFile), headers, false)
				if err != nil {
					ctx.Log().Error(fmt.Sprintf("send output file to webhook: %s", err))
					c.Error(err)
				}
			}()

			return c.NoContent(http.StatusNoContent)
		}
	}
}

// timeoutMiddleware manages hard timeout scenarios, i.e., when a route handler
// fails to timeout as expected.
func timeoutMiddleware(hardTimeout time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := c.Get("logger").(*zap.Logger)

			// Define a hard timeout if the route handler fails to timeout as
			// expected.
			hardTimeoutCtx, hardTimeoutCancel := context.WithTimeout(
				context.Background(),
				hardTimeout,
			)
			defer hardTimeoutCancel()

			errChan := make(chan error, 1)

			go func() {
				// In case of hard timeout, a panic may occur.
				// This deferred function allows us to recover from such scenarios.
				defer func() {
					if r := recover(); r != nil {
						logger.Debug(fmt.Sprintf("recovering from a panic (possible cause being a hard timeout): %s", r))
					}
				}()

				// Call the next middleware in the chain.
				errChan <- next(c)
			}()

			select {
			case err := <-errChan:
				return err
			case <-hardTimeoutCtx.Done():
				logger.Debug("hard timeout as the route handler did not timeout as expected")

				return fmt.Errorf("hard timeout: %w", hardTimeoutCtx.Err())
			}
		}
	}
}
