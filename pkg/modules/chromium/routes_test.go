package chromium

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
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

// TestHandleChromiumError_ConsoleMessages pins the response of
// "failOnConsoleMessageTypes": a 409 that lists the recorded messages.
// See https://github.com/gotenberg/gotenberg/issues/1667.
func TestHandleChromiumError_ConsoleMessages(t *testing.T) {
	// Mirror the wrapping done by [chromiumBrowser.do].
	err := handleChromiumError(fmt.Errorf("%s: %w", "[warning] foo\n[error] bar", ErrConsoleMessages), Options{})
	if err == nil {
		t.Fatal("expected an error, got none")
	}

	status, message := api.ParseError(err)
	if status != http.StatusConflict {
		t.Errorf("status = %d, want %d (message: %s)", status, http.StatusConflict, message)
	}

	want := "Chromium console messages:\n[warning] foo\n[error] bar"
	if message != want {
		t.Errorf("message = %q, want %q", message, want)
	}
}

// TestValidateUrlScheme pins the schemes the URL routes accept. Chromium's
// own schemes must never reach the browser: devtools:// exposes privileged
// bindings to waitForExpression, and chrome:// can crash the browser.
func TestValidateUrlScheme(t *testing.T) {
	for _, tc := range []struct {
		url     string
		wantErr bool
	}{
		{"http://example.com", false},
		{"https://example.com/path?q=1", false},
		{"HTTPS://example.com", false},
		{"file:///tmp/foo/index.html", true},
		{"devtools://devtools/bundled/devtools_app.html", true},
		{"DevTools://devtools/bundled/devtools_app.html", true},
		{"chrome://crash", true},
		{"chrome-untrusted://terminal/", true},
		{"chrome-extension://abc/index.html", true},
		{"view-source:https://example.com", true},
		{"javascript:alert(1)", true},
		{"data:text/html,foo", true},
		{"about:blank", true},
		{"example.com", true},
		{"", true},
	} {
		t.Run(tc.url, func(t *testing.T) {
			err := validateUrlScheme(tc.url)
			if tc.wantErr != (err != nil) {
				t.Fatalf("validateUrlScheme(%q) error = %v, want error: %t", tc.url, err, tc.wantErr)
			}
			if err == nil {
				return
			}

			status, _ := api.ParseError(err)
			if status != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", status, http.StatusBadRequest)
			}
		})
	}
}
