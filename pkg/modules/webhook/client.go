package webhook

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// client gathers all the data required to send a request to a webhook.
type client struct {
	url              string
	method           string
	errorUrl         string
	errorMethod      string
	extraHttpHeaders map[string]string
	startTime        time.Time

	client *retryablehttp.Client
	logger *slog.Logger
}

// send call the webhook either to send the success response or the error response.
func (c client) send(ctx context.Context, body io.Reader, headers map[string]string, errored bool) error {
	url := c.url
	if errored {
		url = c.errorUrl
	}

	method := c.method
	if errored {
		method = c.errorMethod
	}

	spanName := fmt.Sprintf("%s Webhook", method)
	if errored {
		spanName = fmt.Sprintf("%s Webhook Error", method)
	}

	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	req, err := retryablehttp.NewRequest(method, url, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("create '%s' request to '%s': %w", method, url, err)
	}

	// Inject trace context into outbound request headers.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	req.Header.Set("User-Agent", "Gotenberg")

	// Extra HTTP headers are the custom headers from the user.
	for key, value := range c.extraHttpHeaders {
		req.Header.Set(key, value)
	}

	// Middleware caller's headers > extra HTTP headers from the user.

	contentLength, ok := headers[echo.HeaderContentLength]
	if ok {
		// Golang "http" package should automatically calculate the size of the
		// body. But when using a buffered file reader, it does not work.
		// Worse, the "Content-Length" header is also removed. Therefore,
		// to keep this valuable information, we have to trust the caller
		// by reading the value of the "Content-Length" entry and set it as the
		// content length of the request. It's kinda suboptimal, but hey, at
		// least it works.

		bodySize, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("parse content length entry: %w", err)
		}

		req.ContentLength = bodySize
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("send '%s' request to '%s': %w", method, url, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		err := fmt.Errorf("send '%s' request to '%s': got status: '%s'", method, url, resp.Status)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			c.logger.ErrorContext(ctx, fmt.Sprintf("close response body from '%s': %s", url, err))
		}
	}()

	// Last piece for calculating the latency.
	finishTime := time.Now()

	// Now let's log!
	attrs := []any{
		slog.String("webhook_url", url),
		slog.String("method", method),
		slog.Int64("latency", int64(finishTime.Sub(c.startTime))),
		slog.String("latency_human", finishTime.Sub(c.startTime).String()),
		slog.Int64("bytes_out", req.ContentLength),
	}

	if errored {
		c.logger.WarnContext(ctx, "request to webhook with error details handled", attrs...)
		span.SetStatus(codes.Ok, "")
		return nil
	}

	c.logger.InfoContext(ctx, "request to webhook handled", attrs...)
	span.SetStatus(codes.Ok, "")

	return nil
}
