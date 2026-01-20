package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	semconvutil "github.com/gotenberg/gotenberg/v8/pkg/gotenberg/semconv"
)

var (
	// ErrAsyncProcess happens when a handler or middleware handles a request
	// in an asynchronous fashion.
	ErrAsyncProcess = errors.New("async process")

	// ErrNoOutputFile happens when a handler or middleware handles a request
	// without sending any output file.
	ErrNoOutputFile = errors.New("no output file")
)

// ParseError parses an error and returns the corresponding HTTP status and
// HTTP message.
func ParseError(err error) (int, string) {
	var echoErr *echo.HTTPError
	ok := errors.As(err, &echoErr)
	if ok {
		return echoErr.Code, http.StatusText(echoErr.Code)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable)
	}

	if errors.Is(err, gotenberg.ErrFiltered) {
		return http.StatusForbidden, http.StatusText(http.StatusForbidden)
	}

	if errors.Is(err, gotenberg.ErrMaximumQueueSizeExceeded) {
		return http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests)
	}

	if errors.Is(err, gotenberg.ErrPdfSplitModeNotSupported) {
		return http.StatusBadRequest, "At least one PDF engine cannot process the requested PDF split mode, while others may have failed to split due to different issues"
	}

	if errors.Is(err, gotenberg.ErrPdfFormatNotSupported) {
		return http.StatusBadRequest, "At least one PDF engine cannot process the requested PDF format, while others may have failed to convert due to different issues"
	}

	if errors.Is(err, gotenberg.ErrPdfEngineMetadataValueNotSupported) {
		return http.StatusBadRequest, "At least one PDF engine cannot process the requested metadata, while others may have failed to convert due to different issues"
	}

	var invalidArgsError *gotenberg.PdfEngineInvalidArgsError
	if errors.As(err, &invalidArgsError) {
		return http.StatusBadRequest, invalidArgsError.Error()
	}

	var httpErr HttpError
	if errors.As(err, &httpErr) {
		return httpErr.HttpError()
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

// latencyMiddleware sets the start time in the [echo.Context] under
// "startTime". Its value will be used later to calculate request latency.
//
//	startTime := c.Get("startTime").(time.Time)
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

// rootPathMiddleware sets the root path in the [echo.Context] under
// "rootPath". Its value may be used to skip a middleware execution based on a
// request URI.
//
//	rootPath := c.Get("rootPath").(string)
//	healthURI := fmt.Sprintf("%s/health", rootPath)
//
//	// Skip the middleware if it's the health check URI.
//	if c.Request().RequestURI == healthURI {
//	  // Call the next middleware in the chain.
//	  return next(c)
//	}
func rootPathMiddleware(rootPath string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("rootPath", rootPath)
			// Call the next middleware in the chain.
			return next(c)
		}
	}
}

// outputFilenameMiddleware sets the output filename in the [echo.Context]
// under "outputFilename".
//
//	outputFilename := c.Get("outputFilename").(string)
func outputFilenameMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			filename := c.Request().Header.Get("Gotenberg-Output-Filename")
			// See https://github.com/gotenberg/gotenberg/issues/1227.
			if filename != "" {
				filename = filepath.Base(filename)
			}
			c.Set("outputFilename", filename)
			// Call the next middleware in the chain.
			return next(c)
		}
	}
}

// telemetryMiddleware manages telemetry. It sets the correlation ID in the
// [echo.Context] under "correlationId".
//
//	correlationIdHeader := c.Get("correlationIdHeader").(string)
//	correlationId := c.Get("correlationId").(string)
func telemetryMiddleware(logger *zap.Logger, serverName, correlationIdHeader string, disableLoggingForPaths []string) echo.MiddlewareFunc {
	// TODO: scope name, service name (?), instrumentation version.
	meter := otel.GetMeterProvider().Meter("gotenberg")
	semconvSrv := semconvutil.NewHTTPServer(meter)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			startTime := c.Get("startTime").(time.Time)
			rootPath := c.Get("rootPath").(string)

			request := c.Request()
			savedCtx := request.Context()
			defer func() {
				request = request.WithContext(savedCtx)
				c.SetRequest(request)
			}()

			routePath := func() string {
				path := c.Request().URL.Path

				if path == "" {
					path = "/"
				}

				return path
			}()

			correlationId := request.Header.Get(correlationIdHeader)
			c.Set("correlationIdHeader", correlationIdHeader)
			c.Set("correlationId", correlationId)

			ctx := otel.GetTextMapPropagator().Extract(savedCtx, propagation.HeaderCarrier(request.Header))
			opts := []trace.SpanStartOption{
				trace.WithAttributes(
					semconvSrv.RequestTraceAttrs(serverName, request, semconvutil.RequestTraceAttrsOpts{})...,
				),
				trace.WithSpanKind(trace.SpanKindServer),
			}
			rAttr := semconvSrv.Route(routePath)
			opts = append(opts, trace.WithAttributes(rAttr))
			spanName := strings.ToUpper(c.Request().Method) + " " + routePath

			tracer := otel.GetTracerProvider().Tracer("gotenberg")
			ctx, span := tracer.Start(ctx, spanName, opts...)
			defer span.End()

			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(c.Response().Header()))
			c.Response().Header().Set(correlationIdHeader, correlationId)
			c.SetRequest(c.Request().WithContext(ctx))

			spanCtx := span.SpanContext()

			var traceFields []zap.Field
			traceFields = append(traceFields, zap.String("correlation_id", correlationId))
			if spanCtx.IsValid() {
				traceFields = append(
					traceFields,
					zap.String("trace_id", spanCtx.TraceID().String()),
					zap.String("span_id", spanCtx.SpanID().String()),
				)
			}

			appLogger := logger.
				With(zap.String("log_type", "application")).
				With(traceFields...)

			c.Set("logger", appLogger.Named(func() string {
				return strings.ReplaceAll(
					strings.ReplaceAll(c.Request().URL.Path, rootPath, ""),
					"/",
					"",
				)
			}()))

			// Call the next middleware in the chain.
			err := next(c)
			finishTime := time.Now()
			if err != nil {
				span.SetAttributes(attribute.String("error", err.Error()))
				c.Error(err)
			}

			status := c.Response().Status
			span.SetStatus(semconvSrv.Status(status))
			span.SetAttributes(semconvSrv.ResponseTraceAttrs(semconvutil.ResponseTelemetry{
				StatusCode: status,
				WriteBytes: c.Response().Size,
			})...)

			// Record the server-side attributes.
			var additionalAttributes []attribute.KeyValue
			additionalAttributes = append(additionalAttributes, semconvSrv.Route(routePath))

			accessLogger := logger.
				With(zap.String("log_type", "access")).
				With(traceFields...)

			for _, path := range disableLoggingForPaths {
				URI := fmt.Sprintf("%s%s", rootPath, path)

				if c.Request().RequestURI == URI {
					return nil
				}
			}

			fields := []zap.Field{
				zap.String("remote_ip", c.RealIP()),
				zap.String("host", c.Request().Host),
				zap.String("uri", c.Request().RequestURI),
				zap.String("method", c.Request().Method),
				zap.String("path", routePath),
				zap.String("referer", c.Request().Referer()),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.Int("status", c.Response().Status),
				zap.Int64("latency", int64(finishTime.Sub(startTime))),
				zap.String("latency_human", finishTime.Sub(startTime).String()),
				zap.Int64("bytes_in", c.Request().ContentLength),
				zap.Int64("bytes_out", c.Response().Size),
			}
			if err != nil {
				accessLogger.Error(err.Error(), fields...)
			} else {
				accessLogger.Info("request handled", fields...)
			}

			semconvSrv.RecordMetrics(ctx, semconvutil.ServerMetricData{
				ServerName:   serverName,
				ResponseSize: c.Response().Size,
				MetricAttributes: semconvutil.MetricAttributes{
					Req:                  request,
					StatusCode:           status,
					AdditionalAttributes: additionalAttributes,
				},
				MetricData: semconvutil.MetricData{
					RequestSize: request.ContentLength,
					ElapsedTime: float64(time.Since(startTime)) / float64(time.Millisecond),
				},
			})

			return nil
		}
	}
}

// basicAuthMiddleware manages basic authentication.
func basicAuthMiddleware(username, password string) echo.MiddlewareFunc {
	return middleware.BasicAuth(func(u string, p string, e echo.Context) (bool, error) {
		if subtle.ConstantTimeCompare([]byte(u), []byte(username)) == 1 &&
			subtle.ConstantTimeCompare([]byte(p), []byte(password)) == 1 {
			return true, nil
		}
		return false, nil
	})
}

// contextMiddleware, middleware for "multipart/form-data" requests, sets the
// [Context] and related context.CancelFunc in the [echo.Context] under
// "context" and "cancel". If the process is synchronous, it also handles the
// result of a "multipart/form-data" request.
//
//	ctx := c.Get("context").(*api.Context)
//	cancel := c.Get("cancel").(context.CancelFunc)
func contextMiddleware(fs *gotenberg.FileSystem, timeout time.Duration, bodyLimit int64, downloadFromCfg downloadFromConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := c.Get("logger").(*zap.Logger)

			// We create a context with a timeout so that underlying processes are
			// able to stop early and correctly handle a timeout scenario.
			ctx, cancel, err := newContext(c, logger, fs, timeout, bodyLimit, downloadFromCfg)
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

			if errors.Is(err, ErrNoOutputFile) {
				// A middleware/handler tells us that it's handling the process
				// in an asynchronous fashion. Therefore, we must not cancel
				// the context nor send an output file.
				return nil
			}

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
