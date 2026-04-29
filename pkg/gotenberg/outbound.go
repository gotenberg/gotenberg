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

// ErrPublicIP indicates that an outbound URL targets an IP address that is
// reachable on the public internet. It is returned when a caller opts
// into denying public destinations via [WithDenyPublicIPs]; typical use
// cases are air-gapped or data-governed deployments where Gotenberg must
// only talk to hosts on a private network.
var ErrPublicIP = errors.New("public IP")

// netipResolver is the subset of [net.Resolver] used by [resolveHost].
// Defining it as an interface allows tests to substitute a stub resolver.
type netipResolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// outboundResolver is the resolver used by [resolveHost]. It is a
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

// ResolveAndCheckPublic resolves host and rejects any resolved address
// that fails [IsPublicIP] with [ErrNonPublicIP]. It is the strict
// equivalent of [DecideOutbound] with [WithDenyPrivateIPs] true for a
// bare host. Callers that need a different policy should use
// [DecideOutbound] directly.
func ResolveAndCheckPublic(ctx context.Context, host string) ([]netip.Addr, error) {
	return resolveHost(ctx, host, true, false)
}

// resolveHost resolves host and returns the addresses. When denyPrivate
// is true, a non-public address is rejected with [ErrNonPublicIP]. When
// denyPublic is true, a public address is rejected with [ErrPublicIP].
// Both checks may be active at the same time, in which case any
// resolved address fails and the caller must rely on an allow-list
// bypass.
func resolveHost(ctx context.Context, host string, denyPrivate, denyPublic bool) ([]netip.Addr, error) {
	if host == "" {
		return nil, errors.New("empty host")
	}

	check := func(a netip.Addr) error {
		public := IsPublicIP(a)
		if denyPublic && public {
			return fmt.Errorf("%q: %w", a, ErrPublicIP)
		}
		if denyPrivate && !public {
			return fmt.Errorf("%q: %w", a, ErrNonPublicIP)
		}
		return nil
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if err := check(addr); err != nil {
			return nil, err
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
		if err := check(a); err != nil {
			return nil, fmt.Errorf("%q resolves to rejected address %w", host, err)
		}
	}
	return addrs, nil
}

// OutboundDecision is the result of validating an outbound URL via
// [DecideOutbound]. Callers use it to dial the destination either directly
// (operator-approved allow-list match, Bypass true) or via [DialPinned] so
// that the connect targets the IPs resolved at validation time. Passing
// the decision to the dialer closes the window between validation and
// connect that DNS rebinding exploits.
type OutboundDecision struct {
	// Bypass is true when an allow-list pattern matched the URL. The
	// operator has explicitly opted into the destination; the caller
	// should dial directly without an additional IP check.
	Bypass bool

	// Pinned holds the IPs resolved for the URL host. The caller should
	// dial one of these via [DialPinned] to prevent DNS rebinding between
	// validation and connect.
	Pinned []netip.Addr
}

// outboundDecisionKey is the context key under which an [OutboundDecision]
// is stored.
type outboundDecisionKey struct{}

// decideConfig carries optional settings for [DecideOutbound] and
// [FilterOutboundURL]. See [DecideOption] for how callers configure it.
type decideConfig struct {
	denyPrivateIPs bool
	denyPublicIPs  bool
}

// DecideOption customizes how [DecideOutbound] and [FilterOutboundURL]
// validate a URL. Options are applied in order on top of the permissive
// defaults (no IP-class rejection).
type DecideOption func(*decideConfig)

// WithDenyPrivateIPs rejects URLs whose host resolves to a non-public IP
// address (loopback, RFC1918, link-local, unique-local, multicast,
// unspecified). DNS still runs and the returned [OutboundDecision] still
// carries the resolved IPs for dial pinning, so enabling or disabling
// this option does not affect DNS-rebinding protection. Use it on
// internet-exposed deployments to mitigate SSRF against internal
// services.
func WithDenyPrivateIPs(deny bool) DecideOption {
	return func(c *decideConfig) { c.denyPrivateIPs = deny }
}

// WithDenyPublicIPs rejects URLs whose host resolves to a public IP
// address. Use it on air-gapped or data-governed deployments where
// Gotenberg must only reach hosts on a private network; the option
// prevents data exfiltration to attacker-controlled public servers via
// webhook callbacks, downloadFrom URLs, or user-supplied stamp sources.
// May be combined with [WithDenyPrivateIPs]; in that case every resolved
// address fails and only an allow-list bypass permits a destination.
func WithDenyPublicIPs(deny bool) DecideOption {
	return func(c *decideConfig) { c.denyPublicIPs = deny }
}

// httpLikeScheme reports whether scheme is one of http, https, ws, or wss.
// Only these schemes go through the IP-based address check; data, blob,
// file, and other schemes are filtered by the regex layer alone.
func httpLikeScheme(scheme string) bool {
	switch scheme {
	case "http", "https", "ws", "wss":
		return true
	}
	return false
}

// DecideOutbound parses rawURL, runs the regex allow/deny lists against
// the normalized form, and (when no allow-list match) resolves the host
// and applies the IP-class checks selected by opts. It returns the
// resulting [OutboundDecision] so the caller can pin the dial to the IPs
// that were resolved here and skip a second DNS lookup later, which
// closes the DNS rebinding window that affects callers that only receive
// an error from [FilterOutboundURL].
//
// The semantics:
//
//  1. The URL is parsed and its scheme and host lowercased.
//  2. allowList and denyList apply against the normalized form with OR
//     semantics. The deny-list always applies.
//  3. For http, https, ws, and wss, the host is resolved and every
//     resolved address must satisfy the enabled IP-class checks
//     ([WithDenyPrivateIPs], [WithDenyPublicIPs]). An allow-list match
//     bypasses the IP-class checks and the returned decision carries
//     Bypass true. Otherwise the decision carries Pinned with the
//     resolved addresses.
//
// Callers that dial the destination themselves must honor Bypass and
// Pinned: bypassed URLs dial the hostname directly (operator opt-in);
// pinned URLs must dial one of Pinned via [DialPinned].
func DecideOutbound(ctx context.Context, rawURL string, allowList, denyList []*regexp2.Regexp, deadline time.Time, opts ...DecideOption) (OutboundDecision, error) {
	cfg := decideConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return OutboundDecision{}, fmt.Errorf("parse URL %q: %w", rawURL, ErrFiltered)
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
					return OutboundDecision{}, context.DeadlineExceeded
				}
				return OutboundDecision{}, fmt.Errorf("'%s' cannot handle '%s': %w", clone.String(), normalized, err)
			}

			if ok {
				allowMatched = true
				break
			}
		}

		if !allowMatched {
			return OutboundDecision{}, fmt.Errorf("'%s' does not match any expression from the allowed list: %w", normalized, ErrFiltered)
		}
	}

	for _, pattern := range denyList {
		clone := regexp2.MustCompile(pattern.String(), 0)
		clone.MatchTimeout = time.Until(deadline)

		ok, err := clone.MatchString(normalized)
		if err != nil {
			if time.Now().After(deadline) {
				return OutboundDecision{}, context.DeadlineExceeded
			}
			return OutboundDecision{}, fmt.Errorf("'%s' cannot handle '%s': %w", clone.String(), normalized, err)
		}

		if ok {
			return OutboundDecision{}, fmt.Errorf("'%s' matches the expression from the denied list: %w", normalized, ErrFiltered)
		}
	}

	if allowMatched {
		return OutboundDecision{Bypass: true}, nil
	}

	if !httpLikeScheme(parsed.Scheme) {
		return OutboundDecision{}, nil
	}

	host := parsed.Hostname()
	if host == "" {
		return OutboundDecision{}, fmt.Errorf("URL %q has no host: %w", rawURL, ErrFiltered)
	}

	addrs, err := resolveHost(ctx, host, cfg.denyPrivateIPs, cfg.denyPublicIPs)
	if err != nil {
		switch {
		case errors.Is(err, ErrNonPublicIP):
			return OutboundDecision{}, fmt.Errorf("'%s' targets a non-public address: %w", normalized, ErrFiltered)
		case errors.Is(err, ErrPublicIP):
			return OutboundDecision{}, fmt.Errorf("'%s' targets a public address: %w", normalized, ErrFiltered)
		default:
			return OutboundDecision{}, fmt.Errorf("validate '%s' host: %w", normalized, err)
		}
	}

	return OutboundDecision{Pinned: addrs}, nil
}

// FilterOutboundURL validates that rawURL is acceptable for an outbound
// request from Gotenberg. It is the URL-aware replacement for
// [FilterDeadline] and should be preferred for any new code that filters
// a URL before issuing or instructing an outbound request.
//
// The default behavior is permissive: the URL passes as long as it clears
// the regex allow-list and deny-list. Callers that need IP-class checks
// opt in via [WithDenyPrivateIPs] or [WithDenyPublicIPs]. The deny-list
// always applies and cannot be bypassed by an allow-list match.
func FilterOutboundURL(ctx context.Context, rawURL string, allowList, denyList []*regexp2.Regexp, deadline time.Time, opts ...DecideOption) error {
	_, err := DecideOutbound(ctx, rawURL, allowList, denyList, deadline, opts...)
	return err
}

// outboundRoundTripper is an [http.RoundTripper] that validates each
// request URL via [DecideOutbound] and stashes the resulting
// [OutboundDecision] in the request context so that [secureDialContext]
// can pin the dial or bypass the IP check as appropriate. Because the
// http.Client invokes RoundTrip again for each redirect hop, this also
// re-validates redirect targets without a separate CheckRedirect.
type outboundRoundTripper struct {
	base      http.RoundTripper
	allowList []*regexp2.Regexp
	denyList  []*regexp2.Regexp
	opts      []DecideOption
}

// RoundTrip validates req.URL and delegates to the base transport.
func (rt *outboundRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline, ok := req.Context().Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}

	decision, err := DecideOutbound(req.Context(), req.URL.String(), rt.allowList, rt.denyList, deadline, rt.opts...)
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(req.Context(), outboundDecisionKey{}, decision)
	return rt.base.RoundTrip(req.WithContext(ctx))
}

// NewOutboundHttpClient returns an [http.Client] that validates every
// outbound request URL via the same logic as [FilterOutboundURL] and
// pins the resulting dial to the resolved IPs.
//
// The client re-validates redirect targets automatically because the
// underlying [http.Client] invokes the wrapping [http.RoundTripper] once
// per hop. This closes the redirect-based SSRF bypass that affects raw
// [http.Client] usage when no CheckRedirect is set.
//
// The default posture is permissive; callers pass [WithDenyPrivateIPs]
// or [WithDenyPublicIPs] to opt into IP-class rejection.
func NewOutboundHttpClient(timeout time.Duration, allowList, denyList []*regexp2.Regexp, opts ...DecideOption) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.DialContext = secureDialContext
	return &http.Client{
		Timeout: timeout,
		Transport: &outboundRoundTripper{
			base:      base,
			allowList: allowList,
			denyList:  denyList,
			opts:      opts,
		},
	}
}

// secureDialContext consumes the [OutboundDecision] stashed in ctx by
// [outboundRoundTripper]. When the decision is to bypass (allow-list
// match), it dials directly. When the decision contains pinned IPs, it
// dials each in turn until one connects. When no decision is present
// (the dialer was used outside of [outboundRoundTripper]), it falls back
// to resolving the destination without IP-class checks so that the
// fallback matches the permissive default and operators who need
// restrictions configure them at the caller.
func secureDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split host:port %q: %w", addr, err)
	}

	if decision, ok := ctx.Value(outboundDecisionKey{}).(OutboundDecision); ok {
		if decision.Bypass {
			return outboundDialer.DialContext(ctx, network, addr)
		}
		if len(decision.Pinned) > 0 {
			return DialPinned(ctx, network, decision.Pinned, port)
		}
	}

	addrs, err := resolveHost(ctx, host, false, false)
	if err != nil {
		return nil, err
	}
	return DialPinned(ctx, network, addrs, port)
}

// DialPinned dials each addr in turn until one connects, returning the
// first successful connection or the last error. Callers pass the Pinned
// slice from [OutboundDecision] so that the dial targets exactly the IPs
// that [DecideOutbound] resolved, preventing DNS rebinding between
// validation and connect.
func DialPinned(ctx context.Context, network string, addrs []netip.Addr, port string) (net.Conn, error) {
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
