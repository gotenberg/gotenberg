package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

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
