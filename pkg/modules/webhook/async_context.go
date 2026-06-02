package webhook

import (
	"context"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// detachAsyncContext detaches ctx from the inbound request lifecycle so the
// webhook goroutine survives echo recycling the request, while preserving the
// conversion deadline.
//
// Echo cancels the request context as soon as the synchronous handler returns
// [api.ErrAsyncProcess], which would abort the asynchronous work. Detaching via
// [context.WithoutCancel] severs that cancellation while keeping the context
// values, most importantly the active trace span, so downstream conversion and
// webhook spans stay in the caller's trace instead of starting a new one.
// [context.WithoutCancel] also drops the deadline, so it is re-layered. The
// returned cancel function cleans up both the detached context and the original
// working directory.
func detachAsyncContext(ctx *api.Context, cancel context.CancelFunc) context.CancelFunc {
	deadline, hasDeadline := ctx.Deadline()
	base := context.WithoutCancel(ctx.Context)

	var detachedCtx context.Context
	var detachedCancel context.CancelFunc
	if hasDeadline {
		detachedCtx, detachedCancel = context.WithDeadline(base, deadline)
	} else {
		// Fallback if no deadline was set (rare, as newContext enforces it).
		detachedCtx, detachedCancel = context.WithCancel(base)
	}
	ctx.Context = detachedCtx

	originalCancel := cancel
	return func() {
		detachedCancel()
		originalCancel()
	}
}
