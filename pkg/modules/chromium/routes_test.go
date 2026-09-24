package chromium

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// TestHandleChromiumError_Crashed pins the mapping of a Chromium renderer
// crash to a 503 Service Unavailable. When the renderer crashes mid-conversion,
// the request must fail fast with 503 rather than hang until the deadline and
// surface as a generic timeout.
// See https://github.com/gotenberg/gotenberg/issues/1640.
func TestHandleChromiumError_Crashed(t *testing.T) {
	// Mirror the wrapping done by [chromiumBrowser.do].
	err := handleChromiumError(fmt.Errorf("handle tasks: %w", ErrChromiumCrashed), Options{})
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	status, message := api.ParseError(err)
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d (message: %s)", status, http.StatusServiceUnavailable, message)
	}

	want := "Chromium crashed while processing the request. Retry, or reduce the workload if the problem persists."
	if message != want {
		t.Errorf("message = %q, want %q", message, want)
	}
}

// TestHandleChromiumError_CrashedTakesPrecedence guards the ordering in
// [handleChromiumError]: a crash is a server-side failure and must map to 503
// even when the error chain also carries a marker that another branch would
// map to a client-error status.
func TestHandleChromiumError_CrashedTakesPrecedence(t *testing.T) {
	err := handleChromiumError(
		fmt.Errorf("handle tasks: %w; %w", ErrChromiumCrashed, ErrInvalidHttpStatusCode),
		Options{},
	)
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	status, _ := api.ParseError(err)
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", status, http.StatusServiceUnavailable)
	}
}
