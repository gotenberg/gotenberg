package gotenberg

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	"golang.org/x/net/http/httpproxy"
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

// nonPublicIPv6Prefixes lists IPv6 ranges that the standard library does
// not classify via [netip.Addr] helpers but that must not be considered
// public:
//
//   - 2002::/16    6to4 (RFC 3056, deprecated by RFC 7526). Bits 16-47
//     embed an IPv4 destination, including private ones.
//   - 2001::/32    Teredo (RFC 4380). Bits 96-127 embed an IPv4
//     destination, including private ones.
//   - 64:ff9b::/96 NAT64 well-known prefix (RFC 6052). Low 32 bits
//     embed an IPv4 destination translated by a NAT64 gateway.
//   - 64:ff9b:1::/48 NAT64 local-use prefix (RFC 8215). Same risk.
//   - fec0::/10    Deprecated site-local (RFC 3879). Not covered by
//     [netip.Addr.IsPrivate] which only handles fc00::/7.
//   - ::/96        IPv4-compatible IPv6 (deprecated). Embeds an IPv4
//     destination and is not handled by [netip.Addr.Unmap].
//   - 2001:db8::/32 Documentation range (RFC 3849). Never routable.
//   - 100::/64     Discard prefix (RFC 6666).
var nonPublicIPv6Prefixes = []netip.Prefix{
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("fec0::/10"),
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("100::/64"),
}

// IsPublicIP reports whether addr is reachable on the public internet. It
// returns false for loopback, private (RFC1918), link-local, unspecified,
// multicast, and unique-local addresses. IPv4-mapped IPv6 addresses are
// unmapped before evaluation so that [::ffff:127.0.0.1] is correctly
// identified as loopback.
//
// IPv6 prefixes that tunnel or translate to an embedded IPv4 destination
// (6to4, Teredo, NAT64) are rejected wholesale rather than recursed into,
// because a host that routes them implicitly trusts the IPv4 mapping and
// the prefixes themselves are deprecated or translation-only. See
// [nonPublicIPv6Prefixes] for the full list and rationale.
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
	if addr.Is6() {
		for _, p := range nonPublicIPv6Prefixes {
			if p.Contains(addr) {
				return false
			}
		}
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

// outboundProxiedKey is the context key under which [outboundRoundTripper]
// records that the environment proxy will carry this request, so that the
// dialer knows the address it receives is the proxy's rather than the
// destination's.
type outboundProxiedKey struct{}

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
//  1. The URL is parsed, its scheme and host lowercased, and any userinfo
//     dropped from the form the regexes see. The request still carries the
//     credentials.
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

	// Match on a credential-free form. [url.URL.String] re-emits userinfo
	// between "scheme://" and the host, so keeping it would let any
	// "^https?://<host>" pattern be shifted past its own anchor:
	// http://a@127.0.0.1/ escapes a deny-list anchored on 127\. and
	// http://trusted.example.com@10.0.0.1/ satisfies an allow-list anchored on
	// trusted\.example\.com. The host checks below already read
	// [url.URL.Hostname], which ignores userinfo, so only the regex layer was
	// affected. Dropping the credentials here also keeps them out of the error
	// strings below, which reach operator logs and any OTEL log exporter.
	matchable := *parsed
	matchable.User = nil
	normalized := matchable.String()

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
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			// A cancellation or timeout is not a policy decision; surface it
			// as-is so callers do not report it as a filtered request.
			return OutboundDecision{}, fmt.Errorf("validate '%s' host: %w", normalized, err)
		default:
			// The host could not be resolved, so its address class cannot be
			// verified. Fail closed and treat it as filtered, the same as a
			// host that resolves to a blocked address, so clients get a
			// generic 403 rather than a 500. This also denies alternate IP
			// encodings such as http://2130706433/ that the resolver rejects
			// as a hostname but Chromium would read as a private IP.
			return OutboundDecision{}, fmt.Errorf("validate '%s' host: %v: %w", normalized, err, ErrFiltered)
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

	// proxyFunc mirrors the transport's own proxy resolution. It is nil unless
	// the environment proxy is enabled.
	proxyFunc func(*url.URL) (*url.URL, error)
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

	// A request the proxy will not carry is dialed directly, so it still gets
	// pinned. Without this, enabling the environment proxy would silently drop
	// DNS-rebinding protection for every NO_PROXY host, and for all traffic
	// when no proxy variable is set at all.
	if rt.proxyFunc != nil {
		proxyURL, proxyErr := rt.proxyFunc(req.URL)
		if proxyErr == nil && proxyURL != nil {
			ctx = context.WithValue(ctx, outboundProxiedKey{}, true)
		}
	}

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
//
// When enableEnvironmentProxy is true, the client routes through the proxy
// defined by the standard HTTP_PROXY, HTTPS_PROXY, and NO_PROXY variables,
// including any credentials embedded in those URLs. Dial pinning does not apply
// to a hop the proxy carries, since the proxy owns DNS and egress there; a hop
// the proxy declines, such as a NO_PROXY host, is dialed directly and stays
// pinned. The URL allow/deny and IP-class validation runs either way. Callers
// gate this behind their module's opt-in flag. See
// https://github.com/gotenberg/gotenberg/issues/1592.
func NewOutboundHttpClient(timeout time.Duration, allowList, denyList []*regexp2.Regexp, enableEnvironmentProxy bool, opts ...DecideOption) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()

	var proxyFunc func(*url.URL) (*url.URL, error)

	if enableEnvironmentProxy {
		// Route through the operator's proxy (standard env vars, credentials
		// included). httpproxy.FromEnvironment reads the environment now rather
		// than caching it process-wide like http.ProxyFromEnvironment.
		proxyFunc = httpproxy.FromEnvironment().ProxyFunc()
		base.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyFunc(req.URL)
		}
		// Only a hop the proxy actually carries skips pinning: there the dial
		// targets the proxy, not the destination, and the proxy owns DNS. A hop
		// the proxy declines is dialed directly and stays pinned.
		base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			if proxied, _ := ctx.Value(outboundProxiedKey{}).(bool); proxied {
				return outboundDialer.DialContext(ctx, network, addr)
			}
			return secureDialContext(ctx, network, addr)
		}
	} else {
		// Default: ignore any proxy environment variables and pin the dial to
		// the IPs resolved during validation, closing the DNS-rebinding
		// window. Clearing Proxy is deliberate: the cloned default transport
		// carries http.ProxyFromEnvironment, which combined with the pinned
		// dialer would connect to the destination IP on the proxy's port.
		base.Proxy = nil
		base.DialContext = secureDialContext
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &outboundRoundTripper{
			base:      base,
			allowList: allowList,
			denyList:  denyList,
			opts:      opts,
			proxyFunc: proxyFunc,
		},
	}
}

// environmentProxyVariables are the variables golang.org/x/net/http/httpproxy
// reads, in the casing precedence it applies.
var environmentProxyVariables = []string{
	"HTTP_PROXY", "http_proxy",
	"HTTPS_PROXY", "https_proxy",
	"ALL_PROXY", "all_proxy",
}

// ValidateEnvironmentProxyVariables checks that every proxy variable currently
// set can be parsed as a proxy URL.
//
// httpproxy discards a parse error and falls back to a direct connection, so an
// operator who mistypes a proxy URL would silently lose the egress path they
// meant to enforce. Modules exposing an environment proxy flag call this from
// their Validate so that startup fails loudly instead.
//
// Values are never included in the error: a proxy URL may carry credentials.
func ValidateEnvironmentProxyVariables() error {
	var err error

	for _, name := range environmentProxyVariables {
		if os.Getenv(name) == "" {
			continue
		}

		if !isUsableProxyURL(os.Getenv(name)) {
			err = errors.Join(err, fmt.Errorf("environment variable %s is not a usable proxy URL; unset it, or set it to a value like 'http://user:password@host:3128'", name))
		}
	}

	return err
}

// isUsableProxyURL mirrors httpproxy's own parsing: a URL with a proxy scheme,
// or anything that becomes one once a scheme is prefixed.
func isUsableProxyURL(value string) bool {
	proxyURL, err := url.Parse(value)
	if err == nil {
		switch proxyURL.Scheme {
		case "http", "https", "socks5", "socks5h":
			return true
		}
	}

	// httpproxy retries bare values such as "host:3128" with a scheme.
	_, err = url.Parse("http://" + value)
	return err == nil
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

// DialThroughProxy opens a TCP tunnel to target (a host:port) through the
// HTTP CONNECT proxy at proxyURL, authenticating with any credentials
// embedded in proxyURL. dialProxy dials the proxy's own address; callers pass
// a plain dialer. Chromium and soffice cannot authenticate to a proxy
// themselves, so Gotenberg performs the CONNECT handshake on their behalf.
// The returned connection carries the raw tunnel for the caller to splice
// with the client. See https://github.com/gotenberg/gotenberg/issues/1592.
func DialThroughProxy(ctx context.Context, proxyURL *url.URL, target string, dialProxy func(ctx context.Context, network, addr string) (net.Conn, error)) (net.Conn, error) {
	conn, err := dialProxy(ctx, "tcp", proxyHostPort(proxyURL))
	if err != nil {
		return nil, fmt.Errorf("dial proxy: %w", err)
	}

	if proxyURL.Scheme == "https" {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: proxyURL.Hostname()})
		err = tlsConn.HandshakeContext(ctx)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("TLS handshake with proxy: %w", err)
		}
		conn = tlsConn
	}

	// Bound the CONNECT handshake by the request deadline; cleared once the
	// tunnel is established so splicing manages its own lifetime.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	connectReq := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: target},
		Host:   target,
		Header: make(http.Header),
	}
	if user := proxyURL.User; user != nil {
		password, _ := user.Password()
		connectReq.Header.Set("Proxy-Authorization", proxyAuthHeader(user.Username(), password))
	}

	err = connectReq.Write(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("write CONNECT to proxy: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("read CONNECT response from proxy: %w", err)
	}
	// A CONNECT response carries no body; discard defensively.
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("proxy refused CONNECT to %q with status %d", target, resp.StatusCode)
	}

	_ = conn.SetDeadline(time.Time{})

	// The reader may hold bytes the proxy sent right after the response;
	// overlay it so those tunnel bytes are not lost when splicing.
	return &bufferedConn{Conn: conn, r: br}, nil
}

// proxyHostPort returns proxyURL's host:port, defaulting the port from the
// scheme when the URL omits it.
func proxyHostPort(proxyURL *url.URL) string {
	port := proxyURL.Port()
	if port == "" {
		port = "80"
		if proxyURL.Scheme == "https" {
			port = "443"
		}
	}
	return net.JoinHostPort(proxyURL.Hostname(), port)
}

// proxyAuthHeader builds a Basic Proxy-Authorization header value.
func proxyAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

// bufferedConn overlays a [bufio.Reader] on a [net.Conn] so that bytes
// buffered while reading a proxy's CONNECT response are not lost when the
// tunnel is spliced.
type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

// CloseWrite half-closes the underlying connection. Embedding [net.Conn] hides
// the method, so a CONNECT splice over this connection could never signal EOF
// to the upstream and both sides waited for the other until a timeout.
func (c *bufferedConn) CloseWrite() error {
	cw, ok := c.Conn.(interface{ CloseWrite() error })
	if !ok {
		return fmt.Errorf("underlying %T does not support half-close", c.Conn)
	}
	return cw.CloseWrite()
}
