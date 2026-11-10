package chromium

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func TestResolvePdfOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario           string
		generateOutline    bool
		generateTaggedIn   bool
		generateTaggedWant bool
	}{
		{
			scenario:           "outline requested forces tagged PDF",
			generateOutline:    true,
			generateTaggedIn:   false,
			generateTaggedWant: true,
		},
		{
			scenario:           "outline requested keeps tagged PDF on",
			generateOutline:    true,
			generateTaggedIn:   true,
			generateTaggedWant: true,
		},
		{
			scenario:           "no outline leaves tagged PDF off",
			generateOutline:    false,
			generateTaggedIn:   false,
			generateTaggedWant: false,
		},
		{
			scenario:           "no outline keeps tagged PDF on",
			generateOutline:    false,
			generateTaggedIn:   true,
			generateTaggedWant: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			options := DefaultPdfOptions()
			options.GenerateDocumentOutline = tc.generateOutline
			options.GenerateTaggedPdf = tc.generateTaggedIn

			got := resolvePdfOptions(options)

			if got.GenerateTaggedPdf != tc.generateTaggedWant {
				t.Errorf("expected GenerateTaggedPdf=%t, got %t", tc.generateTaggedWant, got.GenerateTaggedPdf)
			}
			if got.GenerateDocumentOutline != tc.generateOutline {
				t.Errorf("expected GenerateDocumentOutline=%t, got %t", tc.generateOutline, got.GenerateDocumentOutline)
			}
		})
	}
}

func TestWindowStatusExpression(t *testing.T) {
	for _, tc := range []struct {
		status string
		expect string
	}{
		{status: "", expect: ""},
		{status: "ready", expect: `window.status === "ready"`},
		{status: `x" || alert(1) || "`, expect: `window.status === "x\" || alert(1) || \""`},
	} {
		got := windowStatusExpression(tc.status)
		if got != tc.expect {
			t.Errorf("windowStatusExpression(%q) = %q, want %q", tc.status, got, tc.expect)
		}
	}
}

// TestWaitTimeout pins the error of a wait that the request's time limit
// ends: a [ErrWaitTimeout] naming the form field and the value, rather than a
// context error, while a client that went away keeps the cancellation.
func TestWaitTimeout(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	// A tab-less executor makes chromedp's selector fail at once, as a wait
	// does when its context ends.
	tabless := func(ctx context.Context) context.Context {
		return cdp.WithExecutor(ctx, (*chromedp.Target)(nil))
	}

	ended := func(cause error) context.Context {
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		return tabless(ctx)
	}

	for _, tc := range []struct {
		name        string
		wait        func(ctx context.Context) error
		ctx         context.Context
		wantTimeout string
		wantErr     error
	}{
		{
			name:        "expression out of time",
			wait:        waitForExpressionBeforePrintActionFunc(logger, false, "window.ready === true"),
			ctx:         ended(context.DeadlineExceeded),
			wantTimeout: "The expression 'window.ready === true' (waitForExpression) did not become true within the time limit. Increase it with --api-timeout (API_TIMEOUT), or check the expression.",
		},
		{
			name:        "window status out of time",
			wait:        waitForWindowStatusBeforePrintActionFunc(logger, false, "ready"),
			ctx:         ended(context.DeadlineExceeded),
			wantTimeout: "The window status 'ready' (waitForWindowStatus) was not reached within the time limit. Increase it with --api-timeout (API_TIMEOUT), or check that the page sets it.",
		},
		{
			name:        "selector out of time",
			wait:        waitForSelectorVisibleBeforePrintActionFunc(logger, "#never"),
			ctx:         ended(context.DeadlineExceeded),
			wantTimeout: "The selector '#never' (waitForSelector) did not become visible within the time limit. Increase it with --api-timeout (API_TIMEOUT), or check the selector.",
		},
		{
			name:    "expression, client gone",
			wait:    waitForExpressionBeforePrintActionFunc(logger, false, "window.ready === true"),
			ctx:     ended(context.Canceled),
			wantErr: context.Canceled,
		},
		{
			name:    "selector, client gone",
			wait:    waitForSelectorVisibleBeforePrintActionFunc(logger, "#never"),
			ctx:     ended(context.Canceled),
			wantErr: ErrInvalidSelectorQuery,
		},
		{
			name:    "selector failing on its own",
			wait:    waitForSelectorVisibleBeforePrintActionFunc(logger, "#never"),
			ctx:     tabless(context.Background()),
			wantErr: ErrInvalidSelectorQuery,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.wait(tc.ctx)

			if errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("err = %v, carries the deadline", err)
			}

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || errors.Is(err, ErrWaitTimeout) {
					t.Fatalf("err = %v, want %v and no wait timeout", err, tc.wantErr)
				}
				return
			}

			var timeout *waitTimeoutError
			if !errors.As(err, &timeout) || !errors.Is(err, ErrWaitTimeout) {
				t.Fatalf("err = %v, want a wait timeout", err)
			}

			if got := timeout.message(); got != tc.wantTimeout {
				t.Errorf("message = %q, want %q", got, tc.wantTimeout)
			}
		})
	}
}

// TestRequestError pins the error a run reports once the request ended: the
// request's own error first, but for a wait that already says it ran out of
// time.
func TestRequestError(t *testing.T) {
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	timeout := &waitTimeoutError{field: "waitForSelector", value: "#never"}
	if err := requestError(expired, timeout); err != timeout {
		t.Errorf("wait timeout = %v, want it unchanged", err)
	}

	canceled := errors.New("context canceled")
	if err := requestError(expired, canceled); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("run error = %v, want the deadline first", err)
	}

	if err := requestError(context.Background(), canceled); err != canceled {
		t.Errorf("run error of a live request = %v, want it unchanged", err)
	}
}

// cookieExecutor answers every DevTools command with err.
type cookieExecutor struct{ err error }

func (e cookieExecutor) Execute(context.Context, string, any, any) error { return e.err }

// TestSetCookiesActionFuncKeepsValuesOutOfLogsAndErrors pins that a cookie's
// value, often a session, appears neither in the debug log of a cookie set nor
// in the error of one Chromium refuses, which the access log writes at the
// error level. See GHSA-g85v-3gpx-4gw5.
func TestSetCookiesActionFuncKeepsValuesOutOfLogsAndErrors(t *testing.T) {
	const secret = "COOKIE_SECRET_SENTINEL"
	cookies := []Cookie{{Name: "session", Value: secret, Domain: "example.invalid"}}

	for _, tc := range []struct {
		scenario string
		err      error
	}{
		{scenario: "cookie set", err: nil},
		{scenario: "cookie refused", err: errors.New("Sanitizing cookie failed (-32602)")},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			handler := &recordingHandler{}
			logger := slog.New(handler)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			ctx = cdp.WithExecutor(ctx, cookieExecutor{err: tc.err})

			err := setCookiesActionFunc(logger, cookies)(ctx)
			if tc.err != nil && err == nil {
				t.Fatal("expected an error, got none")
			}
			if tc.err == nil && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), secret) {
				t.Errorf("the error holds the cookie value: %v", err)
			}
			records := handler.snapshot()
			for _, r := range records {
				if strings.Contains(r.Message, secret) {
					t.Errorf("a log message holds the cookie value: %s", r.Message)
				}
			}
			if tc.err == nil && !slices.ContainsFunc(records, func(r slog.Record) bool { return strings.Contains(r.Message, "Name: 'session'") }) {
				t.Errorf("no log message names the cookie set")
			}
		})
	}
}
