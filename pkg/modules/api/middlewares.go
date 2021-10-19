package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// ErrAsyncProcess happens when a handler or middleware handles a request in an
// asynchronous fashion.
var ErrAsyncProcess = errors.New("async process")

// ParseError parses an error and returns the corresponding HTTP status and
// HTTP message.
func ParseError(err error) (int, string) {
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

// httpErrorHandler is the centralized HTTP error handler. It parses the error,
// returns a response as "text/plain; charset=UTF-8".
func httpErrorHandler() echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		logger := c.Get("logger").(*zap.Logger)
		status, message := ParseError(err)

		c.Response().Header().Add(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)

		err = c.String(status, message)
		if err != nil {
			logger.Error(fmt.Sprintf("send error response: %s", err.Error()))
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
//  healthURI := fmt.Sprintf("%s/health", rootPath)
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
//  traceHeader := c.Get("traceHeader").(string).
func traceMiddleware(header string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get or create the request identifier.
			trace := c.Request().Header.Get(header)

			if trace == "" {
				trace = uuid.New().String()
			}

			c.Set("trace", trace)
			c.Set("traceHeader", header)
			c.Response().Header().Add(header, trace)

			// Call the next middleware in the chain.
			return next(c)
		}
	}
}

// timeoutsMiddleware sets the read, process and write timeouts in the
// echo.Context under "readTimeout", "processTimeout" and "writeTimeout".
//
//  readTimeout := c.Get("readTimeout").(time.Duration)
//  processTimeout := c.Get("processTimeout").(time.Duration)
//  writeTimeout := c.Get("writeTimeout").(time.Duration)
func timeoutsMiddleware(readTimeout, processTimeout, writeTimeout time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("readTimeout", readTimeout)
			c.Set("processTimeout", processTimeout)
			c.Set("writeTimeout", writeTimeout)

			// Call the next middleware in the chain.
			return next(c)
		}
	}
}

// loggerMiddleware sets the logger in the echo.Context under "logger" and logs
// a synchronous request result.
//
//  logger := c.Get("logger").(*zap.Logger)
func loggerMiddleware(logger *zap.Logger, disableLoggingForPaths []string) echo.MiddlewareFunc {
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

			for _, path := range disableLoggingForPaths {
				rootPath := c.Get("rootPath").(string)
				URI := fmt.Sprintf("%s%s", rootPath, path)

				if c.Request().RequestURI == URI {
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

// contextMiddleware, a middleware for "multipart/form-data" requests, sets the
// Context and related context.CancelFunc in the echo.Context under "context"
// and "cancel". If the process is synchronous, it also handles the result of a
// "multipart/form-data" request.
//
//  ctx := c.Get("context").(*api.Context)
//  cancel := c.Get("cancel").(context.CancelFunc)
func contextMiddleware(processTimeout time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := c.Get("logger").(*zap.Logger)

			// We create a context with a timeout so that underlying processes are
			// able to stop early and handle correctly a timeout scenario.
			ctx, cancel, err := newContext(c, logger, processTimeout)
			if err != nil {
				cancel()

				return fmt.Errorf("create request context: %w", err)
			}
			c.Set("context", ctx)
			c.Set("cancel", cancel)

			// Call the next middleware in the chain.
			err = next(c)

			if errors.Is(err, ErrAsyncProcess) {
				// A middleware/handler tells us that it's handling the process
				// in an asynchronous fashion. Therefore, we must not cancel
				// the context nor send an output file.
				return c.NoContent(http.StatusNoContent)
			}

			defer cancel()

			if err != nil {
				return err
			}

			// No error, let's build the output file.
			outputPath, err := ctx.BuildOutputFile()
			if err != nil {
				return fmt.Errorf("build output file: %w", err)
			}

			// Send the output file.
			err = c.Attachment(outputPath, ctx.OutputFilename(outputPath))
			if err != nil {
				return fmt.Errorf("send response: %w", err)
			}

			return nil
		}
	}
}

// hardTimeoutMiddleware manages hard timeout scenarios, i.e., when a route
// handler fails to timeout as expected.
func hardTimeoutMiddleware(hardTimeout time.Duration) echo.MiddlewareFunc {
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
