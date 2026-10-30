package chromium

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/chromedp/cdproto"
	"github.com/dlclark/regexp2/v2"
)

// TestChromiumBrowser_Start_honorsDoneContext checks that a start whose
// context is already done touches no startup resource, so the supervisor never
// leaves a half-started browser behind a request that gave up.
// See https://github.com/gotenberg/gotenberg/issues/1599.
func TestChromiumBrowser_Start_honorsDoneContext(t *testing.T) {
	b := &chromiumBrowser{initialCtx: context.Background()}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.Start(ctx, slog.New(slog.DiscardHandler))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected a context.Canceled error, got %v", err)
	}

	if b.userProfileDirPath != "" {
		t.Fatalf("expected no user profile directory to be created, got %q", b.userProfileDirPath)
	}
	if b.isStarted.Load() {
		t.Fatal("expected the browser to stay not started")
	}
}

// TestChromiumDisableFeatures guards the override described in
// https://github.com/gotenberg/gotenberg/issues/1656. Gotenberg replaces
// chromedp's --disable-features value rather than extending it, as
// chromedp.Flag keys its flags by switch name. Dropping one of chromedp's own
// entries while editing this list would silently re-enable it.
func TestChromiumDisableFeatures(t *testing.T) {
	for _, feature := range []string{
		// chromedp.DefaultExecAllocatorOptions.
		"site-per-process",
		"Translate",
		"BlinkGenPropertyTrees",
		// The address-bar popup WebUI, built even in headless.
		"WebUIOmniboxPopup",
		"WebUIOmniboxAimPopup",
	} {
		if !slices.Contains(strings.Split(chromiumDisableFeatures, ","), feature) {
			t.Errorf("expected %q to be disabled, got %q", feature, chromiumDisableFeatures)
		}
	}
}

// A WebSocket the outbound policy rejects must reach the pinning proxy, so
// the implicit loopback proxy bypass is removed whenever a policy can reject
// a network URL. See GHSA-73vq-mh8c-27wg.
func TestBrowserArguments_restrictsNetwork(t *testing.T) {
	re := func(patterns ...string) []*regexp2.Regexp {
		var res []*regexp2.Regexp
		for _, p := range patterns {
			res = append(res, regexp2.MustCompile(p, regexp2.None))
		}
		return res
	}

	for _, tc := range []struct {
		scenario string
		args     browserArguments
		expect   bool
	}{
		{scenario: "default deny-list only", args: browserArguments{denyList: re(defaultDenyListPattern)}, expect: false},
		{scenario: "no lists", args: browserArguments{}, expect: false},
		{scenario: "allow-list", args: browserArguments{allowList: re(`^https://example\.com`), denyList: re(defaultDenyListPattern)}, expect: true},
		{scenario: "custom deny-list", args: browserArguments{denyList: re(`^http://internal`)}, expect: true},
		{scenario: "default and custom deny-list", args: browserArguments{denyList: re(defaultDenyListPattern, `^http://internal`)}, expect: true},
		{scenario: "deny private IPs", args: browserArguments{denyList: re(defaultDenyListPattern), denyPrivateIPs: true}, expect: true},
		{scenario: "deny public IPs", args: browserArguments{denyPublicIPs: true}, expect: true},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if got := tc.args.restrictsNetwork(); got != tc.expect {
				t.Fatalf("restrictsNetwork() = %t, want %t", got, tc.expect)
			}
		})
	}
}

func TestIsInvalidPrinterSettingsError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"invalid printer settings", &cdproto.Error{Code: -32000, Message: "Show invalid printer settings error"}, true},
		{"scale out of range", &cdproto.Error{Code: -32602, Message: "scale is outside of [0.1 - 2] range"}, true},
		{"negative left margin", &cdproto.Error{Code: -32602, Message: "left margin is negative"}, true},
		{"negative bottom margin", &cdproto.Error{Code: -32602, Message: "bottom margin is negative"}, true},
		{"zero paper width", &cdproto.Error{Code: -32602, Message: "paper width is zero or negative"}, true},
		{"zero paper height", &cdproto.Error{Code: -32602, Message: "paper height is zero or negative"}, true},
		{"empty printable area", &cdproto.Error{Code: -32602, Message: "invalid print parameters: printable area is empty"}, true},
		{"empty content area", &cdproto.Error{Code: -32602, Message: "invalid print parameters: content area is empty"}, true},
		{"invalid print parameters", &cdproto.Error{Code: -32602, Message: "invalid print parameters"}, true},
		{"printing failed", &cdproto.Error{Code: -32000, Message: "Printing failed"}, false},
		{"page range exceeds page count", &cdproto.Error{Code: -32000, Message: "Page range exceeds page count"}, false},
		{"unknown", errors.New("boom"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Mirrors how the print task wraps the error returned by Chromium.
			err := fmt.Errorf("print to PDF: %w", tc.err)
			if got := isInvalidPrinterSettingsError(err.Error()); got != tc.want {
				t.Errorf("isInvalidPrinterSettingsError(%q) = %t, want %t", err.Error(), got, tc.want)
			}
		})
	}
}
