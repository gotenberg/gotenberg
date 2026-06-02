package chromium

import (
	"context"
	"errors"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestChromiumErrorType(t *testing.T) {
	for _, tc := range []struct {
		name        string
		err         error
		queueReason string
		want        string
	}{
		{"deadline", context.DeadlineExceeded, "chromium_unavailable", "timeout"},
		{"canceled", context.Canceled, "chromium_unavailable", "context_cancelled"},
		{"invalid http status", ErrInvalidHttpStatusCode, "chromium_unavailable", "invalid_input"},
		{"invalid resource http status", ErrInvalidResourceHttpStatusCode, "chromium_unavailable", "invalid_input"},
		{"loading failed", ErrLoadingFailed, "chromium_unavailable", "invalid_input"},
		{"resource loading failed", ErrResourceLoadingFailed, "chromium_unavailable", "invalid_input"},
		{"invalid evaluation expression", ErrInvalidEvaluationExpression, "chromium_unavailable", "invalid_input"},
		{"invalid selector query", ErrInvalidSelectorQuery, "chromium_unavailable", "invalid_input"},
		{"pdf queue", gotenberg.ErrMaximumQueueSizeExceeded, "chromium_unavailable", "chromium_unavailable"},
		{"screenshot queue", gotenberg.ErrMaximumQueueSizeExceeded, "chromium_maximum_queue_size_exceeded", "chromium_maximum_queue_size_exceeded"},
		{"restarting", gotenberg.ErrProcessAlreadyRestarting, "chromium_maximum_queue_size_exceeded", "chromium_unavailable"},
		{"unknown", errors.New("boom"), "chromium_unavailable", "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := chromiumErrorType(tc.err, tc.queueReason); got != tc.want {
				t.Errorf("chromiumErrorType(%v, %q) = %q, want %q", tc.err, tc.queueReason, got, tc.want)
			}
		})
	}
}
