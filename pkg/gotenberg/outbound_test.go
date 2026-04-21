package gotenberg

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
)

func TestIsPublicIP(t *testing.T) {
	for _, tc := range []struct {
		addr   string
		public bool
	}{
		// Public.
		{"1.1.1.1", true},
		{"8.8.8.8", true},
		{"2606:4700:4700::1111", true},

		// Loopback.
		{"127.0.0.1", false},
		{"127.255.255.254", false},
		{"::1", false},

		// IPv4-mapped IPv6 (Issue 2).
		{"::ffff:127.0.0.1", false},
		{"::ffff:10.0.0.1", false},
		{"::ffff:169.254.169.254", false},

		// RFC1918.
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"172.31.255.254", false},
		{"192.168.1.1", false},

		// Link-local.
		{"169.254.169.254", false},
		{"fe80::1", false},

		// Unique-local.
		{"fc00::1", false},
		{"fd12:3456:789a::1", false},

		// Unspecified.
		{"0.0.0.0", false},
		{"::", false},

		// Multicast.
		{"224.0.0.1", false},
		{"ff02::1", false},
	} {
		t.Run(tc.addr, func(t *testing.T) {
			addr, err := netip.ParseAddr(tc.addr)
			if err != nil {
				t.Fatalf("parse %q: %v", tc.addr, err)
			}
			if got := IsPublicIP(addr); got != tc.public {
				t.Fatalf("IsPublicIP(%q) = %v, want %v", tc.addr, got, tc.public)
			}
		})
	}
}

// stubResolver lets tests fake DNS lookups in [ResolveAndCheckPublic].
type stubResolver struct {
	lookup func(host string) ([]netip.Addr, error)
}

func (s stubResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	return s.lookup(host)
}

func withStubResolver(t *testing.T, fn func(host string) ([]netip.Addr, error)) {
	t.Helper()
	prev := outboundResolver
	outboundResolver = stubResolver{lookup: fn}
	t.Cleanup(func() { outboundResolver = prev })
}

func mustAddrs(t *testing.T, ss ...string) []netip.Addr {
	t.Helper()
	out := make([]netip.Addr, 0, len(ss))
	for _, s := range ss {
		a, err := netip.ParseAddr(s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		out = append(out, a)
	}
	return out
}

func TestFilterOutboundURL(t *testing.T) {
	defaultDeny := []*regexp2.Regexp{
		regexp2.MustCompile(`^https?://(10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.|169\.254\.|0\.0\.0\.0|127\.|localhost|\[::1\]|\[fd)`, 0),
	}
	chromiumDeny := []*regexp2.Regexp{
		regexp2.MustCompile(`^file:(?!//\/tmp/).*`, 0),
	}

	for _, tc := range []struct {
		scenario     string
		rawURL       string
		allow        []*regexp2.Regexp
		deny         []*regexp2.Regexp
		stub         func(host string) ([]netip.Addr, error)
		expectErr    bool
		expectIs     error
		expectErrMsg string
	}{
		{
			scenario:  "public IP literal passes",
			rawURL:    "https://1.1.1.1/",
			deny:      defaultDeny,
			expectErr: false,
		},
		{
			scenario:  "loopback IP literal blocked by default deny-list",
			rawURL:    "http://127.0.0.1:8080/",
			deny:      defaultDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "Issue 4: uppercase scheme normalized then blocked by deny-list",
			rawURL:    "HTTP://127.0.0.1:8080/",
			deny:      defaultDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "Issue 2: IPv4-mapped IPv6 evades deny-list but blocked by IP check",
			rawURL:    "http://[::ffff:127.0.0.1]:8080/page.pdf",
			deny:      defaultDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "Issue 2: IPv4-mapped IPv6 to RFC1918 blocked by IP check",
			rawURL:    "http://[::ffff:10.0.0.1]/",
			deny:      defaultDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "hostname resolving to public IP passes",
			rawURL:    "https://example.com/",
			deny:      defaultDeny,
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "93.184.216.34"), nil },
			expectErr: false,
		},
		{
			scenario:  "hostname resolving to loopback blocked",
			rawURL:    "https://rebind.example/",
			deny:      defaultDeny,
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "127.0.0.1"), nil },
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "hostname resolving to mixed public+private blocked",
			rawURL:    "https://mixed.example/",
			deny:      defaultDeny,
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "1.1.1.1", "10.0.0.1"), nil },
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "allow-list match bypasses IP check",
			rawURL:    "http://internal.service/api",
			allow:     []*regexp2.Regexp{regexp2.MustCompile(`^http://internal\.service`, 0)},
			deny:      defaultDeny,
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "10.0.0.1"), nil },
			expectErr: false,
		},
		{
			scenario:  "deny-list still wins over allow-list match",
			rawURL:    "http://internal.service/api",
			allow:     []*regexp2.Regexp{regexp2.MustCompile(`^http://internal`, 0)},
			deny:      []*regexp2.Regexp{regexp2.MustCompile(`/api$`, 0)},
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "allow-list non-empty and no match rejects",
			rawURL:    "https://other.example/",
			allow:     []*regexp2.Regexp{regexp2.MustCompile(`^https://allowed\.example`, 0)},
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "file:// allowed under tmp passes Chromium default",
			rawURL:    "file:///tmp/index.html",
			deny:      chromiumDeny,
			expectErr: false,
		},
		{
			scenario:  "file:// outside tmp blocked by Chromium default",
			rawURL:    "file:///etc/passwd",
			deny:      chromiumDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "Issue 1: Chromium default does not block http to public host (regex layer)",
			rawURL:    "https://example.com/",
			deny:      chromiumDeny,
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "93.184.216.34"), nil },
			expectErr: false,
		},
		{
			scenario:  "Issue 1: Chromium default now blocks http to loopback via IP layer",
			rawURL:    "http://127.0.0.1:3000/health",
			deny:      chromiumDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "Issue 1: Chromium default now blocks cloud metadata via IP layer",
			rawURL:    "http://169.254.169.254/latest/meta-data/",
			deny:      chromiumDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "data: URL passes (non-network scheme)",
			rawURL:    "data:text/html;base64,PGgxPmhpPC9oMT4=",
			expectErr: false,
		},
		{
			scenario:  "URL with no host rejected",
			rawURL:    "http:///path",
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "userinfo cannot mask host",
			rawURL:    "http://example.com@127.0.0.1/",
			deny:      defaultDeny,
			expectErr: true,
			expectIs:  ErrFiltered,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.stub != nil {
				withStubResolver(t, tc.stub)
			} else {
				// Default: any DNS lookup in a non-stubbed test is a bug.
				withStubResolver(t, func(host string) ([]netip.Addr, error) {
					t.Fatalf("unexpected DNS lookup for %q", host)
					return nil, nil
				})
			}

			err := FilterOutboundURL(context.Background(), tc.rawURL, tc.allow, tc.deny, time.Now().Add(5*time.Second))

			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tc.expectIs != nil && !errors.Is(err, tc.expectIs) {
				t.Fatalf("expected error to wrap %v, got: %v", tc.expectIs, err)
			}
		})
	}
}

func TestResolveAndCheckPublic_IPLiteralLoopback(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		t.Fatalf("unexpected DNS lookup for %q", host)
		return nil, nil
	})

	_, err := ResolveAndCheckPublic(context.Background(), "127.0.0.1")
	if !errors.Is(err, ErrNonPublicIP) {
		t.Fatalf("expected ErrNonPublicIP, got: %v", err)
	}
}

func TestResolveAndCheckPublic_HostResolvesToLoopback(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		return mustAddrs(t, "127.0.0.1"), nil
	})

	_, err := ResolveAndCheckPublic(context.Background(), "rebind.example")
	if !errors.Is(err, ErrNonPublicIP) {
		t.Fatalf("expected ErrNonPublicIP, got: %v", err)
	}
}

func TestResolveAndCheckPublic_HostResolvesToPublic(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		return mustAddrs(t, "1.1.1.1"), nil
	})

	addrs, err := ResolveAndCheckPublic(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || addrs[0].String() != "1.1.1.1" {
		t.Fatalf("expected [1.1.1.1], got: %v", addrs)
	}
}

func TestDecideOutbound_AllowPrivateIPs_DenyListStillApplies(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		t.Fatalf("unexpected DNS lookup for %q", host)
		return nil, nil
	})

	// Denial must still win over WithAllowPrivateIPs(true). Gherkin
	// coverage exercises flag on/off against the IP check but not the
	// interaction with the regex deny-list; keep this primitive test for
	// that specific combination.
	deny := []*regexp2.Regexp{regexp2.MustCompile(`^http://evil\.`, 0)}

	_, err := DecideOutbound(
		context.Background(),
		"http://evil.local/",
		nil, deny,
		time.Now().Add(5*time.Second),
		WithAllowPrivateIPs(true),
	)
	if !errors.Is(err, ErrFiltered) {
		t.Fatalf("deny-list must still win with WithAllowPrivateIPs(true), got: %v", err)
	}
}
