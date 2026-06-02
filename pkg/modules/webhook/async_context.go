package webhook

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// detachAsyncContext detaches ctx from the inbound request lifecycle so the
// webhook goroutine survives echo recycling the request, while preserving the
// conversion deadline and the caller's trace.
//
// Echo cancels the request context as soon as the synchronous handler returns
// [api.ErrAsyncProcess], which would abort the asynchronous work. Detaching via
// [context.WithoutCancel] severs that cancellation while keeping the context
// values. [context.WithoutCancel] also drops the deadline, so it is re-layered.
//
// The server span ends as soon as that handler returns, so its span context is
// re-seated as a remote, non-recording parent: the asynchronous worker keeps
// the same trace without recording into a span that is about to end. A
// worker-root [webhook.Async] span is then opened, linked back to the
// originating request span, and stays open for the whole delivery so downstream
// conversion and webhook spans have a live parent in the caller's trace.
//
// The returned cancel function ends the worker span and cleans up both the
// detached context and the original working directory.
func detachAsyncContext(ctx *api.Context, cancel context.CancelFunc) context.CancelFunc {
	deadline, hasDeadline := ctx.Deadline()

	serverSpanCtx := trace.SpanContextFromContext(ctx.Context)
	base := context.WithoutCancel(ctx.Context)
	if serverSpanCtx.IsValid() {
		base = trace.ContextWithRemoteSpanContext(base, serverSpanCtx)
	}

	var detachedCtx context.Context
	var detachedCancel context.CancelFunc
	if hasDeadline {
		detachedCtx, detachedCancel = context.WithDeadline(base, deadline)
	} else {
		// Fallback if no deadline was set (rare, as newContext enforces it).
		detachedCtx, detachedCancel = context.WithCancel(base)
	}

	startOpts := []trace.SpanStartOption{trace.WithSpanKind(trace.SpanKindInternal)}
	if serverSpanCtx.IsValid() {
		startOpts = append(startOpts, trace.WithLinks(trace.Link{SpanContext: serverSpanCtx}))
	}

	workerCtx, workerSpan := gotenberg.Tracer().Start(detachedCtx, "webhook.Async", startOpts...)
	ctx.Context = workerCtx

	originalCancel := cancel
	return func() {
		workerSpan.End()
		detachedCancel()
		originalCancel()
	}
}
