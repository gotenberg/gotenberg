package chromium

import (
	"context"
	"errors"
	"testing"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

func TestChromiumErrorType(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"deadline", context.DeadlineExceeded, "timeout"},
		{"canceled", context.Canceled, "context_cancelled"},
		{"invalid http status", ErrInvalidHttpStatusCode, "invalid_input"},
		{"invalid resource http status", ErrInvalidResourceHttpStatusCode, "invalid_input"},
		{"loading failed", ErrLoadingFailed, "invalid_input"},
		{"resource loading failed", ErrResourceLoadingFailed, "invalid_input"},
		{"console messages", ErrConsoleMessages, "invalid_input"},
		{"crashed", ErrChromiumCrashed, "chromium_unavailable"},
		{"invalid evaluation expression", ErrInvalidEvaluationExpression, "invalid_input"},
		{"invalid selector query", ErrInvalidSelectorQuery, "invalid_input"},
		{"console exceptions", ErrConsoleExceptions, "invalid_input"},
		{"screenshot selector not found", ErrScreenshotSelectorNotFound, "invalid_input"},
		{"omit background", ErrOmitBackgroundWithoutPrintBackground, "invalid_input"},
		{"printing failed", ErrPrintingFailed, "invalid_input"},
		{"invalid printer settings", ErrInvalidPrinterSettings, "invalid_input"},
		{"page ranges syntax", ErrPageRangesSyntaxError, "invalid_input"},
		{"page ranges exceed count", ErrPageRangesExceedsPageCount, "invalid_input"},
		{"queue", gotenberg.ErrMaximumQueueSizeExceeded, "chromium_unavailable"},
		{"restarting", gotenberg.ErrProcessAlreadyRestarting, "chromium_unavailable"},
		{"unknown", errors.New("boom"), "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := chromiumErrorType(tc.err); got != tc.want {
				t.Errorf("chromiumErrorType(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
