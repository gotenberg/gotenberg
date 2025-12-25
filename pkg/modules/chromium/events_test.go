package chromium

import "testing"

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
