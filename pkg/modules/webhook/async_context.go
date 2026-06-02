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
// [api.ErrAsyncProcess], which would abort the asynchronous work. Replacing the
// embedded context severs that cancellation. The returned cancel function
// cleans up both the detached context and the original working directory.
func detachAsyncContext(ctx *api.Context, cancel context.CancelFunc) context.CancelFunc {
	if deadline, ok := ctx.Deadline(); ok {
		detachedCtx, detachedCancel := context.WithDeadline(context.Background(), deadline)
		ctx.Context = detachedCtx

		originalCancel := cancel
		return func() {
			detachedCancel()
			originalCancel()
		}
	}

	// Fallback if no deadline was set (rare, as newContext enforces it).
	ctx.Context = context.Background()
	return cancel
}
