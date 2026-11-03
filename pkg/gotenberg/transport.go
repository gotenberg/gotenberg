package gotenberg

import (
	"io"
	"net/http"
	"slices"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	semconvutil "github.com/gotenberg/gotenberg/v9/pkg/gotenberg/semconv"
)

// NewTracingTransport wraps base so that every request it sends, each attempt
// of a retried delivery and each redirect hop, gets its own HTTP client span,
// as the semconv defines it. The span is the parent the receiver sees: the
// request carries its trace context. It ends once the response body is
// closed.
//
// The url.full attribute goes through [RedactURL], so that a credential in the
// URL, such as a token in its query, does not reach the traces.
func NewTracingTransport(base http.RoundTripper) http.RoundTripper {
	return &tracingTransport{base: base}
}

type tracingTransport struct {
	base http.RoundTripper
	conv semconvutil.HTTPClient
}

func (t *tracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// A RoundTripper must not modify the request: the trace context goes on a
	// clone.
	out := req.Clone(req.Context())

	attrs := slices.DeleteFunc(t.conv.RequestTraceAttrs(out), func(kv attribute.KeyValue) bool {
		return kv.Key == semconv.URLFullKey
	})
	attrs = append(attrs, semconv.URLFull(RedactURL(out.URL.String())))

	ctx, span := Tracer().Start(out.Context(), clientSpanName(out.Method),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	out = out.WithContext(ctx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(out.Header))

	resp, err := t.base.RoundTrip(out)
	if err != nil {
		span.SetAttributes(t.conv.ErrorType(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()

		return nil, err
	}

	span.SetAttributes(t.conv.ResponseTraceAttrs(resp)...)
	span.SetStatus(t.conv.Status(resp.StatusCode))

	if resp.Body == nil || resp.Body == http.NoBody {
		span.End()
		return resp, nil
	}

	resp.Body = &spanBody{ReadCloser: resp.Body, span: span}

	return resp, nil
}

// clientSpanName returns the name of an HTTP client span without a URL
// template: the method alone, or "HTTP" for a method the semconv does not
// know.
func clientSpanName(method string) string {
	switch method {
	case "":
		return http.MethodGet
	case http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions,
		http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace:
		return method
	default:
		return "HTTP"
	}
}

// spanBody ends the span of a response once its body is closed, so that the
// span covers the transfer of the body.
type spanBody struct {
	io.ReadCloser
	span trace.Span
	once sync.Once
}

func (b *spanBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(func() { b.span.End() })

	return err
}
