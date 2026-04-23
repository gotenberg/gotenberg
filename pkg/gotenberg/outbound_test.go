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
		opts         []DecideOption
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
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "Issue 2: IPv4-mapped IPv6 to RFC1918 blocked by IP check",
			rawURL:    "http://[::ffff:10.0.0.1]/",
			deny:      defaultDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "hostname resolving to public IP passes with deny-private-ips",
			rawURL:    "https://example.com/",
			deny:      defaultDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "93.184.216.34"), nil },
			expectErr: false,
		},
		{
			scenario:  "hostname resolving to loopback blocked with deny-private-ips",
			rawURL:    "https://rebind.example/",
			deny:      defaultDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "127.0.0.1"), nil },
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "hostname resolving to mixed public+private blocked with deny-private-ips",
			rawURL:    "https://mixed.example/",
			deny:      defaultDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "1.1.1.1", "10.0.0.1"), nil },
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "allow-list match bypasses IP check",
			rawURL:    "http://internal.service/api",
			allow:     []*regexp2.Regexp{regexp2.MustCompile(`^http://internal\.service`, 0)},
			deny:      defaultDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
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
			scenario:  "Chromium default permissive passes http to public host",
			rawURL:    "https://example.com/",
			deny:      chromiumDeny,
			stub:      func(string) ([]netip.Addr, error) { return mustAddrs(t, "93.184.216.34"), nil },
			expectErr: false,
		},
		{
			scenario:  "Chromium with deny-private-ips blocks http to loopback",
			rawURL:    "http://127.0.0.1:3000/health",
			deny:      chromiumDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
			expectErr: true,
			expectIs:  ErrFiltered,
		},
		{
			scenario:  "Chromium with deny-private-ips blocks cloud metadata",
			rawURL:    "http://169.254.169.254/latest/meta-data/",
			deny:      chromiumDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
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
			scenario:  "userinfo cannot mask host when deny-private-ips enabled",
			rawURL:    "http://example.com@127.0.0.1/",
			deny:      defaultDeny,
			opts:      []DecideOption{WithDenyPrivateIPs(true)},
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

			err := FilterOutboundURL(context.Background(), tc.rawURL, tc.allow, tc.deny, time.Now().Add(5*time.Second), tc.opts...)

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

func TestDecideOutbound_DenyPrivateIPs_RejectsLoopbackLiteral(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		t.Fatalf("unexpected DNS lookup for %q", host)
		return nil, nil
	})

	_, err := DecideOutbound(
		context.Background(),
		"http://127.0.0.1:8080/",
		nil, nil,
		time.Now().Add(5*time.Second),
		WithDenyPrivateIPs(true),
	)
	if !errors.Is(err, ErrFiltered) {
		t.Fatalf("WithDenyPrivateIPs(true) must reject loopback literal, got: %v", err)
	}
}

func TestDecideOutbound_DenyPrivateIPs_AllowsPublic(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		return mustAddrs(t, "93.184.216.34"), nil
	})

	decision, err := DecideOutbound(
		context.Background(),
		"http://example.com/",
		nil, nil,
		time.Now().Add(5*time.Second),
		WithDenyPrivateIPs(true),
	)
	if err != nil {
		t.Fatalf("expected no error for public host, got: %v", err)
	}
	if len(decision.Pinned) != 1 || decision.Pinned[0].String() != "93.184.216.34" {
		t.Fatalf("decision.Pinned = %v, want [93.184.216.34]", decision.Pinned)
	}
}

func TestDecideOutbound_DenyPublicIPs_RejectsPublic(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		return mustAddrs(t, "1.1.1.1"), nil
	})

	_, err := DecideOutbound(
		context.Background(),
		"http://example.com/",
		nil, nil,
		time.Now().Add(5*time.Second),
		WithDenyPublicIPs(true),
	)
	if !errors.Is(err, ErrFiltered) {
		t.Fatalf("WithDenyPublicIPs(true) must reject public host, got: %v", err)
	}
}

func TestDecideOutbound_DenyPublicIPs_AllowsPrivate(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		return mustAddrs(t, "10.0.0.5"), nil
	})

	decision, err := DecideOutbound(
		context.Background(),
		"http://internal.svc/",
		nil, nil,
		time.Now().Add(5*time.Second),
		WithDenyPublicIPs(true),
	)
	if err != nil {
		t.Fatalf("expected no error for private host, got: %v", err)
	}
	if len(decision.Pinned) != 1 || decision.Pinned[0].String() != "10.0.0.5" {
		t.Fatalf("decision.Pinned = %v, want [10.0.0.5]", decision.Pinned)
	}
}

func TestDecideOutbound_DenyBoth_WhitelistOnly(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		return mustAddrs(t, "1.1.1.1"), nil
	})

	// Both denies active and no allow-list match: every resolved address
	// fails. Only an allow-list match can permit a destination under
	// this posture.
	_, err := DecideOutbound(
		context.Background(),
		"http://example.com/",
		nil, nil,
		time.Now().Add(5*time.Second),
		WithDenyPrivateIPs(true),
		WithDenyPublicIPs(true),
	)
	if !errors.Is(err, ErrFiltered) {
		t.Fatalf("expected ErrFiltered with both denies enabled, got: %v", err)
	}
}

func TestDecideOutbound_DenyLists_WinOverDenyPrivateIPs(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		t.Fatalf("unexpected DNS lookup for %q", host)
		return nil, nil
	})

	// The regex deny-list fires before any resolution; verifies that
	// operator-supplied deny patterns remain effective regardless of
	// IP-class options.
	deny := []*regexp2.Regexp{regexp2.MustCompile(`^http://evil\.`, 0)}

	_, err := DecideOutbound(
		context.Background(),
		"http://evil.local/",
		nil, deny,
		time.Now().Add(5*time.Second),
		WithDenyPrivateIPs(true),
	)
	if !errors.Is(err, ErrFiltered) {
		t.Fatalf("deny-list must still reject, got: %v", err)
	}
}

func TestDecideOutbound_Permissive_AllowsPrivate(t *testing.T) {
	withStubResolver(t, func(host string) ([]netip.Addr, error) {
		return mustAddrs(t, "10.0.0.5"), nil
	})

	// No options passed: default posture is permissive across both
	// IP classes. The caller still gets pinned IPs for dial safety.
	decision, err := DecideOutbound(
		context.Background(),
		"http://internal.svc/",
		nil, nil,
		time.Now().Add(5*time.Second),
	)
	if err != nil {
		t.Fatalf("permissive default must allow private host, got: %v", err)
	}
	if len(decision.Pinned) != 1 || decision.Pinned[0].String() != "10.0.0.5" {
		t.Fatalf("decision.Pinned = %v, want [10.0.0.5]", decision.Pinned)
	}
}
