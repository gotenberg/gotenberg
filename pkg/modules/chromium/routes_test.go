package chromium

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
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

// TestHandleChromiumError_Wait pins the answer of a wait that fails: a 503
// naming the field and its value when the request's time limit ends it, as the
// API answers its own timeout, and a 400 when the value throws.
func TestHandleChromiumError_Wait(t *testing.T) {
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	// Mirror [chromiumBrowser.do] and [runError].
	run := func(waitErr error) error {
		return fmt.Errorf("handle tasks: %w", requestError(expired, waitErr))
	}

	options := Options{WaitForExpression: "window.nope.x"}

	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "selector out of time",
			err:        run(&waitTimeoutError{field: "waitForSelector", value: "#never"}),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "The selector '#never' (waitForSelector) did not become visible within the time limit. Increase it with --api-timeout (API_TIMEOUT), or check the selector.",
		},
		{
			name:       "expression out of time",
			err:        run(&waitTimeoutError{field: "waitForExpression", value: "window.ready === true"}),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "The expression 'window.ready === true' (waitForExpression) did not become true within the time limit. Increase it with --api-timeout (API_TIMEOUT), or check the expression.",
		},
		{
			name:       "window status out of time",
			err:        run(&waitTimeoutError{field: "waitForWindowStatus", value: "ready"}),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "The window status 'ready' (waitForWindowStatus) was not reached within the time limit. Increase it with --api-timeout (API_TIMEOUT), or check that the page sets it.",
		},
		{
			name:       "expression throws",
			err:        fmt.Errorf("handle tasks: %w", fmt.Errorf("evaluate: %v: %w", errors.New("TypeError"), ErrInvalidEvaluationExpression)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "The expression 'window.nope.x' (waitForExpression) returned an exception or undefined",
		},
		{
			name:       "out of time outside a wait",
			err:        run(fmt.Errorf("wait delay: %w", context.Canceled)),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "The request exceeded the time limit. Increase it with --api-timeout, or reduce the workload.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, message := api.ParseError(handleChromiumError(tc.err, options))
			if status != tc.wantStatus || message != tc.wantBody {
				t.Errorf("got %d %q, want %d %q", status, message, tc.wantStatus, tc.wantBody)
			}
		})
	}
}

// TestHandleChromiumError_UnresolvableHost pins the answer of a URL whose host
// does not resolve, a 400 naming it, while a policy refusal stays a generic
// 403.
func TestHandleChromiumError_UnresolvableHost(t *testing.T) {
	unresolvable := fmt.Errorf("filter URL: validate 'https://nope.invalid/' host: %w", &gotenberg.UnresolvableHostError{Host: "nope.invalid", Err: errors.New("no such host")})

	status, message := api.ParseError(handleChromiumError(unresolvable, Options{}))
	want := "Invalid form data: form field 'url' is invalid: the host 'nope.invalid' does not resolve"
	if status != http.StatusBadRequest || message != want {
		t.Errorf("unresolvable: got %d %q, want 400 %q", status, message, want)
	}

	status, message = api.ParseError(handleChromiumError(fmt.Errorf("filter URL: %w", gotenberg.ErrFiltered), Options{}))
	if status != http.StatusForbidden || message != http.StatusText(http.StatusForbidden) {
		t.Errorf("filtered: got %d %q, want the generic 403", status, message)
	}

	// Chromium reads these hosts as IPv4 addresses: they stay refused as
	// filtered.
	for _, host := range []string{"2130706433", "0x7f000001", "127.1", "0x7f.1.", "10.0.0.010"} {
		err := fmt.Errorf("filter URL: %w", &gotenberg.UnresolvableHostError{Host: host, Err: errors.New("no such host")})

		status, message := api.ParseError(handleChromiumError(err, Options{}))
		if status != http.StatusForbidden || message != http.StatusText(http.StatusForbidden) {
			t.Errorf("%s: got %d %q, want the generic 403", host, status, message)
		}
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

func TestFormDataChromiumPdfOptions_PaperSize(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		values                map[string][]string
		wantWidth, wantHeight float64
	}{
		{name: "default", values: nil, wantWidth: 8.5, wantHeight: 11},
		{name: "named size", values: map[string][]string{"paperSize": {"A4"}}, wantWidth: 210 / 25.4, wantHeight: 297 / 25.4},
		{name: "width overrides the size", values: map[string][]string{"paperSize": {"A4"}, "paperWidth": {"100mm"}}, wantWidth: 100 / 25.4, wantHeight: 297 / 25.4},
		{name: "explicit sizes", values: map[string][]string{"paperWidth": {"4"}, "paperHeight": {"6"}}, wantWidth: 4, wantHeight: 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &api.ContextMock{Context: new(api.Context)}
			ctx.SetValues(tc.values)

			form, options := FormDataChromiumPdfOptions(ctx.Context)
			if err := form.Validate(); err != nil {
				t.Fatalf("validate: %v", err)
			}

			if math.Abs(options.PaperWidth-tc.wantWidth) > 1e-9 || math.Abs(options.PaperHeight-tc.wantHeight) > 1e-9 {
				t.Errorf("paper = %gx%g, want %gx%g", options.PaperWidth, options.PaperHeight, tc.wantWidth, tc.wantHeight)
			}
		})
	}
}

func TestFormDataChromiumPdfOptions_NativeFields(t *testing.T) {
	ctx := &api.ContextMock{Context: new(api.Context)}
	ctx.SetValues(map[string][]string{"nativeBookmarks": {"true"}, "nativeTaggedPdf": {"true"}})

	_, options := FormDataChromiumPdfOptions(ctx.Context)
	if !options.GenerateDocumentOutline || !options.GenerateTaggedPdf {
		t.Fatalf("options = %+v, want the outline and tagging from nativeBookmarks and nativeTaggedPdf", options)
	}
}

// JSON objects are unordered: the form must bind them in a stable order.
func TestFormDataChromiumOptions_SortsObjectFields(t *testing.T) {
	ctx := &api.ContextMock{Context: new(api.Context)}
	ctx.SetValues(map[string][]string{
		"extraHttpHeaders":      {`{"X-C":"c","X-A":"a","X-B":"b"}`},
		"emulatedMediaFeatures": {`{"prefers-reduced-motion":"reduce","color-gamut":"p3","prefers-color-scheme":"dark"}`},
	})

	for range 20 {
		form, options := FormDataChromiumOptions(ctx.Context)
		err := form.Validate()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		var headers, features []string
		for _, h := range options.ExtraHttpHeaders {
			headers = append(headers, h.Name)
		}
		for _, f := range options.EmulatedMediaFeatures {
			features = append(features, f.Name)
		}

		if fmt.Sprint(headers) != "[X-A X-B X-C]" {
			t.Fatalf("extra HTTP headers = %v, want them sorted by name", headers)
		}
		if fmt.Sprint(features) != "[color-gamut prefers-color-scheme prefers-reduced-motion]" {
			t.Fatalf("emulated media features = %v, want them sorted by name", features)
		}
	}
}

// TestFormFieldHints pins the fixes the Chromium routes give a Gotenberg 8
// field: the PDF routes know the stages, the screenshot routes run none.
func TestFormFieldHints(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hint  func(field string, file bool) string
		field string
		file  bool
		want  string
	}{
		{name: "renamed on a PDF route", hint: convertFormFieldHint, field: "nativePageRanges", want: "Gotenberg 9 renamed it 'nativePages'"},
		{name: "renamed on every route", hint: screenshotFormFieldHint, field: "waitWindowStatus", want: "Gotenberg 9 renamed it 'waitForWindowStatus'"},
		{name: "PDF field on a screenshot route", hint: screenshotFormFieldHint, field: "nativePageRanges", want: ""},
		{name: "stage on a PDF route", hint: convertFormFieldHint, field: "pdfa", want: `Gotenberg 9 moved it to a stage: add {"op":"remediate","profile":"PDF/A-2b"} to 'stages'`},
		{name: "stages on a screenshot route", hint: screenshotFormFieldHint, field: "stages", want: "screenshots take no 'stages'"},
		{name: "Gotenberg 8 file field on a PDF route", hint: convertFormFieldHint, field: "facturxXml", file: true, want: `Gotenberg 9 reads every file from 'files': upload it there, and add {"op":"remediate","profile":"Factur-X","xml":"invoice.xml"} to 'stages'`},
		{name: "file fields are free-form on a screenshot route", hint: screenshotFormFieldHint, field: "watermark", file: true, want: ""},
		{name: "free-form file field", hint: convertFormFieldHint, field: "assets", file: true, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.hint(tc.field, tc.file); got != tc.want {
				t.Errorf("hint(%q, %t) = %q, want %q", tc.field, tc.file, got, tc.want)
			}
		})
	}
}

// TestIsLocalPage pins the one URL the user lists do not govern: the exact
// page a route wrote, not another file of its directory.
func TestIsLocalPage(t *testing.T) {
	const local = "file:///tmp/work/request/page.html"

	for _, tc := range []struct {
		name  string
		url   string
		local string
		want  bool
	}{
		{name: "the page", url: local, local: local, want: true},
		{name: "the page, re-encoded", url: "file:///tmp/work/request/%70age.html", local: local, want: true},
		{name: "another file of the directory", url: "file:///tmp/work/request/image.png", local: local, want: false},
		{name: "another directory", url: "file:///tmp/other/page.html", local: local, want: false},
		{name: "the page with a query", url: local + "?x=1", local: local, want: false},
		{name: "a host", url: "file://host/tmp/work/request/page.html", local: local, want: false},
		{name: "http", url: "http://tmp/work/request/page.html", local: local, want: false},
		{name: "no local page", url: local, local: "", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLocalPage(tc.url, tc.local); got != tc.want {
				t.Errorf("isLocalPage(%q, %q) = %t, want %t", tc.url, tc.local, got, tc.want)
			}
		})
	}
}
