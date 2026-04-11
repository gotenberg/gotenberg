package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
)

// ErrNonPublicIP indicates that an outbound URL targets an IP address that
// is not reachable on the public internet. This covers loopback, RFC1918
// private, link-local, unspecified, multicast, and IPv6 unique-local
// (fc00::/7) addresses, as well as their IPv4-mapped IPv6 wrappers (for
// example [::ffff:127.0.0.1]).
var ErrNonPublicIP = errors.New("non-public IP")

// netipResolver is the subset of [net.Resolver] used by
// [ResolveAndCheckPublic]. Defining it as an interface allows tests to
// substitute a stub resolver.
type netipResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// outboundResolver is the resolver used by [ResolveAndCheckPublic]. It is a
// package-level variable so that tests can substitute a stub resolver.
var outboundResolver netipResolver = net.DefaultResolver

// outboundDialer is the underlying dialer used by [secureDialContext]. It is
// a package-level variable so that tests can replace it.
var outboundDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// IsPublicIP reports whether addr is reachable on the public internet. It
// returns false for loopback, private (RFC1918), link-local, unspecified,
// multicast, and unique-local addresses. IPv4-mapped IPv6 addresses are
// unmapped before evaluation so that [::ffff:127.0.0.1] is correctly
// identified as loopback.
func IsPublicIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}
	addr = addr.Unmap()
	switch {
	case addr.IsLoopback(),
		addr.IsPrivate(),
		addr.IsLinkLocalUnicast(),
		addr.IsLinkLocalMulticast(),
		addr.IsMulticast(),
		addr.IsUnspecified(),
		addr.IsInterfaceLocalMulticast():
		return false
	}
	return true
}

// ResolveAndCheckPublic resolves host and returns the resolved addresses,
// or an error if any resolved address fails [IsPublicIP]. If host is itself
// an IP literal, it is checked directly without performing a DNS lookup.
// The returned slice can be used to pin a subsequent dial to a specific IP
// and prevent DNS rebinding between this validation and the connect.
func ResolveAndCheckPublic(ctx context.Context, host string) ([]netip.Addr, error) {
	if host == "" {
		return nil, errors.New("empty host")
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if !IsPublicIP(addr) {
			return nil, fmt.Errorf("%q: %w", addr, ErrNonPublicIP)
		}
		return []netip.Addr{addr}, nil
	}
	addrs, err := outboundResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("resolve %q: no addresses returned", host)
	}
	for _, a := range addrs {
		if !IsPublicIP(a) {
			return nil, fmt.Errorf("%q resolves to non-public address %q: %w", host, a, ErrNonPublicIP)
		}
	}
	return addrs, nil
}

// outboundDecision is the result of validating an outbound URL. It is
// stashed in the request context by [outboundRoundTripper] so that
// [secureDialContext] can either bypass the IP check (allow-list match) or
// pin the dial to the IPs that were resolved at validation time.
type outboundDecision struct {
	// bypass is true when an allow-list pattern matched the URL. In that
	// case the operator has explicitly opted into the destination and the
	// dial should proceed without an IP check.
	bypass bool

	// pinned holds the IPs resolved by [ResolveAndCheckPublic] for the URL
	// host. The dial should be pinned to one of these to prevent DNS
	// rebinding between validation and connect.
	pinned []netip.Addr
}

// outboundDecisionKey is the context key under which an [outboundDecision]
// is stored.
type outboundDecisionKey struct{}

// httpLikeScheme reports whether scheme is one of http, https, ws, or wss.
// Only these schemes go through the IP-based public-address check; data,
// blob, file, and other schemes are filtered by the regex layer alone.
func httpLikeScheme(scheme string) bool {
	switch scheme {
	case "http", "https", "ws", "wss":
		return true
	}
	return false
}

// decideOutbound parses rawURL, runs the regex allow/deny lists against the
// normalized form, and (when no allow-list match) resolves the host and
// rejects any non-public address. It returns the resulting
// [outboundDecision] which the caller can stash in a context for the dial.
func decideOutbound(ctx context.Context, rawURL string, allowList, denyList []*regexp2.Regexp, deadline time.Time) (outboundDecision, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return outboundDecision{}, fmt.Errorf("parse URL %q: %w", rawURL, ErrFiltered)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	normalized := parsed.String()

	allowMatched := false
	if len(allowList) > 0 {
		for _, pattern := range allowList {
			clone := regexp2.MustCompile(pattern.String(), 0)
			clone.MatchTimeout = time.Until(deadline)

			ok, err := clone.MatchString(normalized)
			if err != nil {
				if time.Now().After(deadline) {
					return outboundDecision{}, context.DeadlineExceeded
				}
				return outboundDecision{}, fmt.Errorf("'%s' cannot handle '%s': %w", clone.String(), normalized, err)
			}

			if ok {
				allowMatched = true
				break
			}
		}

		if !allowMatched {
			return outboundDecision{}, fmt.Errorf("'%s' does not match any expression from the allowed list: %w", normalized, ErrFiltered)
		}
	}

	for _, pattern := range denyList {
		clone := regexp2.MustCompile(pattern.String(), 0)
		clone.MatchTimeout = time.Until(deadline)

		ok, err := clone.MatchString(normalized)
		if err != nil {
			if time.Now().After(deadline) {
				return outboundDecision{}, context.DeadlineExceeded
			}
			return outboundDecision{}, fmt.Errorf("'%s' cannot handle '%s': %w", clone.String(), normalized, err)
		}

		if ok {
			return outboundDecision{}, fmt.Errorf("'%s' matches the expression from the denied list: %w", normalized, ErrFiltered)
		}
	}

	if allowMatched {
		return outboundDecision{bypass: true}, nil
	}

	if !httpLikeScheme(parsed.Scheme) {
		return outboundDecision{}, nil
	}

	host := parsed.Hostname()
	if host == "" {
		return outboundDecision{}, fmt.Errorf("URL %q has no host: %w", rawURL, ErrFiltered)
	}

	addrs, err := ResolveAndCheckPublic(ctx, host)
	if err != nil {
		if errors.Is(err, ErrNonPublicIP) {
			return outboundDecision{}, fmt.Errorf("'%s' targets a non-public address: %w", normalized, ErrFiltered)
		}
		return outboundDecision{}, fmt.Errorf("validate '%s' host: %w", normalized, err)
	}

	return outboundDecision{pinned: addrs}, nil
}

// FilterOutboundURL validates that rawURL is acceptable for an outbound
// request from Gotenberg. It is the URL-aware replacement for
// [FilterDeadline] and should be preferred for any new code that filters a
// URL before issuing or instructing an outbound request.
//
// The function:
//
//  1. Parses rawURL with [net/url] and lowercases the scheme and host. This
//     prevents case-variant bypasses such as HTTP://127.0.0.1 from evading
//     case-sensitive deny-list regexes.
//  2. Applies allowList and denyList against the normalized form using the
//     same OR semantics as [FilterDeadline].
//  3. When no allow-list entry explicitly matched and the scheme is one of
//     http, https, ws, or wss, resolves the host and verifies every
//     resolved address with [IsPublicIP]. This blocks loopback, private,
//     link-local, and other internal targets even when the regex layer
//     does not cover the textual form (for example IPv4-mapped IPv6 like
//     [::ffff:127.0.0.1], or hostnames that resolve to a private address).
//
// An allow-list match bypasses the IP check, allowing operators to opt
// into specific internal destinations via --*-allow-list flags. The
// deny-list always applies and cannot be bypassed by an allow-list match.
func FilterOutboundURL(ctx context.Context, rawURL string, allowList, denyList []*regexp2.Regexp, deadline time.Time) error {
	_, err := decideOutbound(ctx, rawURL, allowList, denyList, deadline)
	return err
}

// outboundRoundTripper is an [http.RoundTripper] that validates each request
// URL via [decideOutbound] and stashes the resulting [outboundDecision] in
// the request context so that [secureDialContext] can pin the dial or
// bypass the IP check as appropriate. Because the http.Client invokes
// RoundTrip again for each redirect hop, this also re-validates redirect
// targets without a separate CheckRedirect.
type outboundRoundTripper struct {
	base      http.RoundTripper
	allowList []*regexp2.Regexp
	denyList  []*regexp2.Regexp
}

// RoundTrip validates req.URL and delegates to the base transport.
func (rt *outboundRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}

	decision, err := decideOutbound(req.Context(), req.URL.String(), rt.allowList, rt.denyList, deadline)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(req.Context(), outboundDecisionKey{}, decision)
	return rt.base.RoundTrip(req.WithContext(ctx))
}

// NewOutboundHttpClient returns an [http.Client] that validates every
// outbound request URL via the same logic as [FilterOutboundURL] and pins
// the resulting dial to a resolved public IP. An allow-list match
// (operator opt-in to a specific destination) bypasses the IP check.
//
// The client re-validates redirect targets automatically because the
// underlying [http.Client] invokes the wrapping [http.RoundTripper] once
// per hop. This closes the redirect-based SSRF bypass that affects raw
// [http.Client] usage when no CheckRedirect is set.
func NewOutboundHttpClient(timeout time.Duration, allowList, denyList []*regexp2.Regexp) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.DialContext = secureDialContext
	return &http.Client{
		Timeout: timeout,
		Transport: &outboundRoundTripper{
			base:      base,
			allowList: allowList,
			denyList:  denyList,
		},
	}
}

// secureDialContext consumes the [outboundDecision] stashed in ctx by
// [outboundRoundTripper]. When the decision is to bypass (allow-list
// match), it dials directly. When the decision contains pinned IPs, it
// dials each in turn until one connects. When no decision is present (the
// dialer was used outside of [outboundRoundTripper]), it falls back to
// resolving and checking the destination itself.
func secureDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split host:port %q: %w", addr, err)
	}

	if decision, ok := ctx.Value(outboundDecisionKey{}).(outboundDecision); ok {
		if decision.bypass {
			return outboundDialer.DialContext(ctx, network, addr)
		}
		if len(decision.pinned) > 0 {
			return dialPinned(ctx, network, decision.pinned, port)
		}
	}

	addrs, err := ResolveAndCheckPublic(ctx, host)
	if err != nil {
		return nil, err
	}
	return dialPinned(ctx, network, addrs, port)
}

// dialPinned dials each addr in turn until one connects, returning the
// first successful connection or the last error.
func dialPinned(ctx context.Context, network string, addrs []netip.Addr, port string) (net.Conn, error) {
	var lastErr error
	for _, a := range addrs {
		conn, err := outboundDialer.DialContext(ctx, network, net.JoinHostPort(a.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		return nil, errors.New("no addresses to dial")
	}
	return nil, lastErr
}
