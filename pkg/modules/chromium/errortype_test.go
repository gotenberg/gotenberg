package chromium

import (
	"context"
	"errors"
	"fmt"
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
		{"wait out of time", &waitTimeoutError{field: "waitForSelector", value: "#never"}, "timeout"},
		{"canceled", context.Canceled, "context_cancelled"},
		{"wrapped deadline", fmt.Errorf("handle tasks: %w", context.DeadlineExceeded), "timeout"},
		{"wrapped canceled", fmt.Errorf("handle tasks: %w", context.Canceled), "context_cancelled"},
		{"unresolvable host", fmt.Errorf("navigate: %w", &gotenberg.UnresolvableHostError{Host: "gotenberg.invalid", Err: errors.New("no such host")}), "invalid_input"},
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
		{"unclassified", errors.New("boom"), "_OTHER"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := chromiumErrorType(tc.err); got != tc.want {
				t.Errorf("chromiumErrorType(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
