package webhook

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestDetachAsyncContext_PreservesTraceContext(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})

	reqCtx, reqCancel := context.WithDeadline(
		trace.ContextWithSpanContext(context.Background(), sc),
		time.Now().Add(2*time.Hour),
	)
	defer reqCancel()

	ctx := &api.Context{Context: reqCtx}
	cancel := detachAsyncContext(ctx, func() {})
	defer cancel()

	got := trace.SpanContextFromContext(ctx.Context)
	if got.TraceID() != sc.TraceID() {
		t.Errorf("expected the detached context to keep trace id %s, got %s", sc.TraceID(), got.TraceID())
	}
}

func TestDetachAsyncContext_PreservesDeadline(t *testing.T) {
	deadline := time.Now().Add(2 * time.Hour)
	reqCtx, reqCancel := context.WithDeadline(context.Background(), deadline)
	defer reqCancel()

	ctx := &api.Context{Context: reqCtx}
	cancel := detachAsyncContext(ctx, func() {})
	defer cancel()

	got, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected the detached context to keep a deadline")
	}
	if !got.Equal(deadline) {
		t.Errorf("expected deadline %v, got %v", deadline, got)
	}
}

func TestDetachAsyncContext_SurvivesRequestCancellation(t *testing.T) {
	reqCtx, reqCancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Hour))

	ctx := &api.Context{Context: reqCtx}
	cancel := detachAsyncContext(ctx, func() {})
	defer cancel()

	// Cancelling the inbound request must not abort the detached context.
	reqCancel()

	if err := ctx.Err(); err != nil {
		t.Errorf("expected the detached context to survive request cancellation, got %v", err)
	}
}

func TestDetachAsyncContext_CancelInvokesOriginal(t *testing.T) {
	reqCtx, reqCancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	defer reqCancel()

	called := 0
	ctx := &api.Context{Context: reqCtx}
	cancel := detachAsyncContext(ctx, func() { called++ })

	cancel()
	if called != 1 {
		t.Errorf("expected the original cancel to be invoked once, got %d", called)
	}
}
