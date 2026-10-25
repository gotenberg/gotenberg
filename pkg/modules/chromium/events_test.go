package chromium

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/chromedp/cdproto/runtime"
)

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "spaces", in: "   ", want: ""},
		{name: "simple", in: "example.com", want: "example.com"},
		{name: "mixed case", in: "ExAmPlE.Com", want: "example.com"},
		{name: "leading wildcard", in: "*.example.com", want: "example.com"},
		{name: "leading dot", in: ".example.com", want: "example.com"},
		{name: "with scheme and path", in: "https://example.com/foo/bar", want: "example.com"},
		{name: "with port", in: "example.com:443", want: "example.com"},
		{name: "with scheme and port", in: "https://example.com:443/foo", want: "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeDomain(tt.in); got != tt.want {
				t.Fatalf("normalizeDomain(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMatchesAnyDomain(t *testing.T) {
	host := "browser.sentry-cdn.com"

	if !matchesAnyDomain(host, []string{"sentry-cdn.com"}) {
		t.Fatalf("expected %q to match %q", host, "sentry-cdn.com")
	}

	if matchesAnyDomain("not-sentry-cdn.com", []string{"sentry-cdn.com"}) {
		t.Fatalf("expected %q to not match %q", "not-sentry-cdn.com", "sentry-cdn.com")
	}
}

func TestShouldCheckResourceHttpStatusCode_IgnoreDomains(t *testing.T) {
	ignore := normalizeDomains([]string{"sentry.io"})

	if shouldCheckResourceHttpStatusCode("https://sentry.io/api/123", ignore) {
		t.Fatalf("expected ignored domain to be skipped")
	}

	if shouldCheckResourceHttpStatusCode("https://sub.sentry.io/api/123", ignore) {
		t.Fatalf("expected ignored subdomain to be skipped")
	}

	if !shouldCheckResourceHttpStatusCode("https://other.com/api/123", ignore) {
		t.Fatalf("expected non-ignored domain to be checked")
	}
}

func TestShouldCheckResourceHttpStatusCode_NonHTTPURL(t *testing.T) {
	if !shouldCheckResourceHttpStatusCode("data:text/plain,hello", nil) {
		t.Fatalf("expected data: URL to be checked (no host filtering possible)")
	}
}

func TestIsAllowedFileSubResource(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rawURL   string
		prefixes []string
		want     bool
	}{
		{
			name:     "empty prefix list default denies",
			rawURL:   "file:///tmp/work-uuid/request-uuid/index.html",
			prefixes: nil,
			want:     false,
		},
		{
			name:     "match within the sole prefix",
			rawURL:   "file:///tmp/work-uuid/request-uuid/index.html",
			prefixes: []string{"/tmp/work-uuid/request-uuid"},
			want:     true,
		},
		{
			name:     "sibling request directory rejected",
			rawURL:   "file:///tmp/work-uuid/other-request-uuid/secret.html",
			prefixes: []string{"/tmp/work-uuid/request-uuid"},
			want:     false,
		},
		{
			name:     "parent tmp directory rejected",
			rawURL:   "file:///tmp/",
			prefixes: []string{"/tmp/work-uuid/request-uuid"},
			want:     false,
		},
		{
			name:     "match among several prefixes",
			rawURL:   "file:///tmp/work-uuid/request-b/asset.css",
			prefixes: []string{"/tmp/work-uuid/request-a", "/tmp/work-uuid/request-b"},
			want:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAllowedFileSubResource(tc.rawURL, tc.prefixes); got != tc.want {
				t.Fatalf("isAllowedFileSubResource(%q, %v) = %v, want %v", tc.rawURL, tc.prefixes, got, tc.want)
			}
		})
	}
}

// TestFormatConsoleMessage pins how console call arguments render in the
// error response of "failOnConsoleMessageTypes".
// See https://github.com/gotenberg/gotenberg/issues/1667.
func TestFormatConsoleMessage(t *testing.T) {
	tests := []struct {
		name string
		args []*runtime.RemoteObject
		want string
	}{
		{name: "no arguments", args: nil, want: ""},
		{name: "string is unquoted", args: []*runtime.RemoteObject{{Type: runtime.TypeString, Value: []byte(`"an error message"`)}}, want: "an error message"},
		{name: "escaped string", args: []*runtime.RemoteObject{{Type: runtime.TypeString, Value: []byte(`"line 1\nline 2"`)}}, want: "line 1\nline 2"},
		{name: "number", args: []*runtime.RemoteObject{{Type: runtime.TypeNumber, Value: []byte(`42`)}}, want: "42"},
		{name: "unserializable", args: []*runtime.RemoteObject{{Type: runtime.TypeNumber, UnserializableValue: "NaN"}}, want: "NaN"},
		{name: "object", args: []*runtime.RemoteObject{{Type: runtime.TypeObject, Description: "Error: boom"}}, want: "Error: boom"},
		{name: "undefined", args: []*runtime.RemoteObject{{Type: runtime.TypeUndefined}}, want: "undefined"},
		{name: "nil argument", args: []*runtime.RemoteObject{nil, {Type: runtime.TypeString, Value: []byte(`"foo"`)}}, want: "foo"},
		{
			name: "several arguments",
			args: []*runtime.RemoteObject{
				{Type: runtime.TypeString, Value: []byte(`"total:"`)},
				{Type: runtime.TypeNumber, Value: []byte(`3`)},
				{Type: runtime.TypeBoolean, Value: []byte(`true`)},
			},
			want: "total: 3 true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatConsoleMessage(tt.args); got != tt.want {
				t.Fatalf("formatConsoleMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatConsoleMessage_Truncates guards the size cap: a long message is cut
// to [maxConsoleMessageLength] bytes on a rune boundary.
func TestFormatConsoleMessage_Truncates(t *testing.T) {
	// "é" is two bytes, so the cut lands in the middle of a rune.
	long := "a" + strings.Repeat("é", maxConsoleMessageLength)
	got := formatConsoleMessage([]*runtime.RemoteObject{{Type: runtime.TypeString, Value: []byte(`"` + long + `"`)}})

	if !strings.HasSuffix(got, "...") {
		t.Fatalf("formatConsoleMessage() = %q, want a truncated message", got)
	}
	if len(got) > maxConsoleMessageLength+len("...") {
		t.Errorf("len = %d, want at most %d", len(got), maxConsoleMessageLength+len("..."))
	}
	if !utf8.ValidString(got) {
		t.Error("truncated message is not valid UTF-8")
	}
}
