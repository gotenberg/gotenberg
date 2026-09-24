package webhook

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// TestDetachAsyncContext_TraceContinuity asserts that an asynchronous webhook
// conversion stays in the inbound request's trace: the worker-root
// webhook.Async span and the downstream conversion span share the server
// span's trace id, and webhook.Async links back to the server span.
func TestDetachAsyncContext_TraceContinuity(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	serverCtx, serverSpan := provider.Tracer("test").Start(
		context.Background(),
		"POST /forms/chromium/convert/html",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	serverSpanCtx := serverSpan.SpanContext()

	reqCtx, reqCancel := context.WithDeadline(serverCtx, time.Now().Add(time.Hour))
	defer reqCancel()

	ctx := &api.Context{Context: reqCtx}
	cancel := detachAsyncContext(ctx, func() {})

	// Simulate a downstream conversion span using the detached context, as the
	// chromium/libreoffice engines would.
	_, conversionSpan := gotenberg.Tracer().Start(ctx.Context, "chromium.Pdf", trace.WithSpanKind(trace.SpanKindClient))
	conversionSpan.End()

	// The server span ends when the synchronous handler returns, before the
	// asynchronous worker finishes.
	serverSpan.End()
	cancel()

	var asyncSpan, conversion sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		switch s.Name() {
		case "webhook.Async":
			asyncSpan = s
		case "chromium.Pdf":
			conversion = s
		}
	}

	if asyncSpan == nil {
		t.Fatal("expected a webhook.Async span to be recorded")
	}
	if conversion == nil {
		t.Fatal("expected a chromium.Pdf span to be recorded")
	}

	if conversion.SpanContext().TraceID() != serverSpanCtx.TraceID() {
		t.Errorf("conversion span trace id = %s, want %s", conversion.SpanContext().TraceID(), serverSpanCtx.TraceID())
	}
	if asyncSpan.SpanContext().TraceID() != serverSpanCtx.TraceID() {
		t.Errorf("webhook.Async trace id = %s, want %s", asyncSpan.SpanContext().TraceID(), serverSpanCtx.TraceID())
	}
	if conversion.Parent().SpanID() != asyncSpan.SpanContext().SpanID() {
		t.Errorf("conversion span parent = %s, want webhook.Async %s", conversion.Parent().SpanID(), asyncSpan.SpanContext().SpanID())
	}

	links := asyncSpan.Links()
	if len(links) != 1 {
		t.Fatalf("expected webhook.Async to have 1 link, got %d", len(links))
	}
	if links[0].SpanContext.SpanID() != serverSpanCtx.SpanID() {
		t.Errorf("webhook.Async link span id = %s, want %s", links[0].SpanContext.SpanID(), serverSpanCtx.SpanID())
	}
}
