package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlclark/regexp2/v2"
	"golang.org/x/net/http/httpproxy"
)

// OutboundProxy is a loopback-bound HTTP/1.1 forward and CONNECT proxy placed
// between a converter, such as Chromium or LibreOffice, and the outbound
// network. It runs [DecideOutbound] on every request and dials the destination
// using the IPs resolved at that moment. Routing a converter through this
// proxy eliminates the converter-side DNS lookup that otherwise opens a DNS
// rebinding window between Gotenberg's validation and the converter's TCP
// connect.
//
// The proxy is transparent to the caller. HTTPS requests tunnel through
// CONNECT with the converter performing its own TLS handshake using the
// original hostname, preserving SNI and certificate validation. A tunnel only
// exposes its host and port, so the filters see the root URL of the tunnel,
// never its path. See [OutboundProxy.DecideTunnel].
type OutboundProxy struct {
	allowList []*regexp2.Regexp
	denyList  []*regexp2.Regexp

	// decide resolves and validates a URL. Tests may override it.
	decide func(ctx context.Context, rawURL string, allowList, denyList []*regexp2.Regexp, deadline time.Time) (OutboundDecision, error)

	// dialPinned dials the pinned IPs for a decision. Tests may override
	// it to connect to a stub upstream regardless of decision.
	dialPinned func(ctx context.Context, network string, addrs []netip.Addr, port string) (net.Conn, error)

	// dialBypass dials the destination hostname directly (operator
	// allow-list opt-in). Tests may override it.
	dialBypass func(ctx context.Context, network, addr string) (net.Conn, error)

	// upstreamProxy resolves the upstream (corporate) proxy for a
	// destination URL from the standard proxy environment variables, or
	// returns a nil URL to connect directly. It is nil unless the operator
	// opted into proxy-environment honoring. When set, the proxy performs
	// the authenticated proxy handshake that the converter cannot. See
	// https://github.com/gotenberg/gotenberg/issues/1592.
	upstreamProxy func(*url.URL) (*url.URL, error)

	// logRefusal records a request the filters refused.
	logRefusal func(ctx context.Context, rawURL string, err error)

	listener net.Listener
	server   *http.Server
	wg       sync.WaitGroup

	// cancelRequests is called by Stop to force in-flight CONNECT tunnels
	// shut. [http.Server.Shutdown] cannot do it: net/http untracks a
	// connection once a handler hijacks it, so a tunnel would otherwise
	// outlive the proxy that created it. Every request context of a Start
	// derives from the context it cancels, so a tunnel is bound to the server
	// that accepted it, even when it outlives a Stop and the next Start.
	cancelRequests context.CancelFunc

	// maxTunnels ceilings the CONNECT handlers in flight. Tests may lower it.
	maxTunnels int64

	// tunnels counts the CONNECT handlers in flight.
	tunnels atomic.Int64

	logger  *slog.Logger
	started bool
	mu      sync.Mutex
}

// OutboundProxyOptions configures an [OutboundProxy].
type OutboundProxyOptions struct {
	// AllowList and DenyList are the operator's outbound lists, matched as
	// [DecideOutbound] matches them.
	AllowList []*regexp2.Regexp
	DenyList  []*regexp2.Regexp

	// DenyPrivateIPs and DenyPublicIPs refuse destinations of that IP class.
	// See [WithDenyPrivateIPs] and [WithDenyPublicIPs].
	DenyPrivateIPs bool
	DenyPublicIPs  bool

	// EnableEnvironmentProxy sends the requests the filters allow through the
	// proxy the standard proxy environment variables name, credentials
	// included.
	EnableEnvironmentProxy bool

	// HostMapping sends a host where the operator's rules map it. It may be
	// nil. See [WithHostMapping].
	HostMapping HostMapping

	// LogRefusal records a request the filters refused, with the URL the
	// filters saw. When nil, the proxy logs it through [LogOutboundRefusal]
	// with its own logger.
	LogRefusal func(ctx context.Context, rawURL string, err error)
}

// NewOutboundProxy returns a proxy that applies opts to every request it
// sees, so the converter behind it inherits whatever posture the operator
// selected, and connects where the host mapping sends it. The returned proxy
// is not yet listening; call [OutboundProxy.Start].
func NewOutboundProxy(opts OutboundProxyOptions) *OutboundProxy {
	p := &OutboundProxy{
		allowList: opts.AllowList,
		denyList:  opts.DenyList,
		decide: func(ctx context.Context, rawURL string, allow, deny []*regexp2.Regexp, deadline time.Time) (OutboundDecision, error) {
			return DecideOutbound(ctx, rawURL, allow, deny, deadline,
				WithDenyPrivateIPs(opts.DenyPrivateIPs),
				WithDenyPublicIPs(opts.DenyPublicIPs),
				WithHostMapping(opts.HostMapping),
			)
		},
		dialPinned: DialPinned,
		dialBypass: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, network, addr)
		},
		logRefusal: opts.LogRefusal,
		maxTunnels: maxConcurrentTunnels,
	}

	if opts.EnableEnvironmentProxy {
		// Honor the standard proxy environment variables, credentials
		// included. httpproxy reads the environment now and applies NO_PROXY.
		p.upstreamProxy = httpproxy.FromEnvironment().ProxyFunc()
	}

	return p
}

// Start binds the proxy to 127.0.0.1 on an ephemeral port and serves in a
// background goroutine, logging with logger. Bind failures return an error;
// the caller must not point the converter at the proxy then.
func (p *OutboundProxy) Start(logger *slog.Logger) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return errors.New("outbound proxy already started")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("bind outbound proxy: %w", err)
	}

	baseCtx, cancelRequests := context.WithCancel(context.Background())

	p.listener = l
	p.cancelRequests = cancelRequests
	p.logger = logger
	p.server = &http.Server{
		Handler:     http.HandlerFunc(p.serveHTTP),
		BaseContext: func(net.Listener) context.Context { return baseCtx },
		// Guard against slow header attacks. Body reads are controlled
		// per-handler.
		ReadHeaderTimeout: 15 * time.Second,
		ErrorLog:          slog.NewLogLogger(p.logger.Handler(), slog.LevelWarn),
	}

	p.wg.Go(func() {
		serveErr := p.server.Serve(l)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			p.logger.ErrorContext(context.Background(), fmt.Sprintf("outbound proxy serve: %s", serveErr))
		}
	})

	p.started = true
	p.logger.DebugContext(context.Background(), fmt.Sprintf("outbound proxy listening on %s", l.Addr()))
	return nil
}

// Stop shuts the proxy down and waits for in-flight handlers to complete,
// until ctx ends. Safe to call on a non-started proxy.
func (p *OutboundProxy) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	srv := p.server
	cancelRequests := p.cancelRequests
	p.cancelRequests = nil
	p.started = false
	p.mu.Unlock()

	// Force in-flight tunnels shut before draining the server. Shutdown does
	// not reach them, so a tunnel whose upstream never answers would otherwise
	// survive the proxy, and with it every converter restart.
	cancelRequests()

	shutdownErr := srv.Shutdown(ctx)
	p.wg.Wait()

	if shutdownErr != nil {
		return fmt.Errorf("shutdown outbound proxy: %w", shutdownErr)
	}
	p.logger.DebugContext(ctx, "outbound proxy stopped")
	return nil
}

// Addr returns the host:port the proxy listens on, or an empty string when
// the proxy has never listened.
func (p *OutboundProxy) Addr() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener == nil {
		return ""
	}
	return p.listener.Addr().String()
}

func (p *OutboundProxy) serveHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		p.handleConnect(w, req)
		return
	}
	p.handleForward(w, req)
}

// handleConnect handles HTTPS (and any other CONNECT) tunnels. The converter
// issues CONNECT host:port; the proxy validates the host, dials the pinned
// IP, and splices the client socket with the upstream socket. The converter
// then negotiates TLS end-to-end with the original hostname in SNI.
func (p *OutboundProxy) handleConnect(w http.ResponseWriter, req *http.Request) {
	// A ceiling, not a tuning knob: it bounds what a tunnel that refuses to end
	// can accumulate, whatever keeps it alive. [spliceIdleTimeout] ends a silent
	// tunnel, but a peer trickling a byte just under it stays "active" forever,
	// and a compromised renderer can hold the client side open to match.
	//
	// Chromium, the most demanding converter, caps itself well below this. Its
	// socket pool manager allows 128 sockets per proxy chain for normal traffic
	// plus 128 for WebSocket traffic, and every request Gotenberg's Chromium
	// makes traverses this one proxy chain, so an honest browser cannot exceed
	// 256 tunnels here. At double that, a real page never meets the ceiling and
	// a hostile one stops at it. LibreOffice converts one document at a time
	// and fetches far less.
	if !p.acquireTunnel() {
		p.logger.WarnContext(req.Context(), fmt.Sprintf("CONNECT to '%s' refused: %d tunnels already in flight", req.Host, p.maxTunnels))
		http.Error(w, "too many tunnels", http.StatusServiceUnavailable)

		return
	}
	defer p.releaseTunnel()

	_, port, err := net.SplitHostPort(req.Host)
	if err != nil {
		http.Error(w, "bad CONNECT target", http.StatusBadRequest)
		return
	}

	deadline, ok := req.Context().Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}

	decision, err := p.DecideTunnel(req.Context(), req.Host, deadline)
	if err != nil {
		switch {
		case isClientCancellation(req.Context(), err):
			p.logger.DebugContext(req.Context(), fmt.Sprintf("CONNECT abandoned by client for '%s': %s", req.Host, err))
		case p.logRefusal != nil && FilterRule(err) != "":
			p.logRefusal(req.Context(), "https://"+req.Host, err)
		default:
			p.logger.WarnContext(req.Context(), fmt.Sprintf("CONNECT blocked for '%s': %s", req.Host, err))
		}
		http.Error(w, "CONNECT blocked", http.StatusForbidden)
		return
	}

	// When the operator routes egress through an authenticated proxy,
	// the converter cannot supply the credentials itself, so this proxy
	// performs the CONNECT (and authentication) upstream. The decision above
	// still gated the destination through the allow/deny and IP-class rules.
	var proxyURL *url.URL
	if p.upstreamProxy != nil {
		proxyURL, err = p.upstreamProxy(&url.URL{Scheme: "https", Host: req.Host})
		if err != nil {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("resolve upstream proxy for '%s': %s", req.Host, err))
			http.Error(w, "upstream proxy error", http.StatusBadGateway)
			return
		}
	}

	// A host-resolver rule may send the tunnel to another host and port. An
	// upstream proxy still receives the original host, as a browser itself
	// leaves name resolution to the proxy it talks to.
	target := req.Host
	if decision.Target != "" {
		target = decision.Target
		_, port, err = net.SplitHostPort(target)
		if err != nil {
			http.Error(w, "bad CONNECT target", http.StatusBadGateway)
			return
		}
	}

	var upstream net.Conn
	switch {
	case proxyURL != nil:
		upstream, err = p.dialThroughUpstreamProxy(req.Context(), proxyURL, req.Host)
	case decision.Bypass:
		upstream, err = p.dialBypass(req.Context(), "tcp", target)
	case len(decision.Pinned) > 0:
		upstream, err = p.dialPinned(req.Context(), "tcp", decision.Pinned, port)
	default:
		err = errors.New("no pinned addresses and not bypassed")
	}
	if err != nil {
		if isClientCancellation(req.Context(), err) {
			p.logger.DebugContext(req.Context(), fmt.Sprintf("CONNECT dial abandoned by client for '%s': %s", req.Host, err))
		} else {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("CONNECT dial failed for '%s': %s", req.Host, err))
		}
		http.Error(w, "upstream dial failed", http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	client, _, err := hj.Hijack()
	if err != nil {
		p.logger.ErrorContext(req.Context(), fmt.Sprintf("hijack CONNECT: %s", err))
		return
	}
	defer client.Close()

	_, err = client.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	if err != nil {
		if isClientCancellation(req.Context(), err) {
			p.logger.DebugContext(req.Context(), fmt.Sprintf("write CONNECT ack abandoned by client: %s", err))
		} else {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("write CONNECT ack: %s", err))
		}
		return
	}

	spliceTunnel(client, upstream, req.Context().Done(), spliceIdleTimeout)
}

// tunnelSchemes lists the schemes a converter sends through a CONNECT tunnel and
// their default ports: HTTPS sub-resources and WebSocket handshakes, secure or
// not. Plain http:// requests use the forward path instead.
var tunnelSchemes = []struct {
	scheme      string
	defaultPort string
}{
	{"https", "443"},
	{"wss", "443"},
	{"ws", "80"},
}

// DecideTunnel validates the destination of a CONNECT tunnel to hostport.
//
// A tunnel is opaque: the proxy sees the host and port, never the scheme or
// the path of what travels through it. The deny-list is therefore matched
// against the root URL of every scheme a tunnel may carry, such as
// "https://host/" and "wss://host/", so that a rule anchored on a scheme or on
// "host/" refuses the tunnel exactly as it refuses the ordinary request. Such a
// rule refuses every tunnel to that host and port, whatever travels through
// it. Rules that name a path cannot match here: only a layer that sees the
// full URL, such as Chromium's request interception, can apply them.
//
// The allow-list and the IP-class checks then run on the https:// root URL,
// so an allow-list rule such as "^https://cdn\.example\.com/" clears the tunnel
// to cdn.example.com:443 as it clears the ordinary request. The scheme does not
// influence the CONNECT handling beyond filtering.
func (p *OutboundProxy) DecideTunnel(ctx context.Context, hostport string, deadline time.Time) (OutboundDecision, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return OutboundDecision{}, fmt.Errorf("split CONNECT target %q: %v: %w", hostport, err, ErrFiltered)
	}

	var httpsRoot string
	for _, ts := range tunnelSchemes {
		authority := net.JoinHostPort(host, port)
		if port == ts.defaultPort {
			authority = host
			if strings.Contains(host, ":") {
				authority = "[" + host + "]"
			}
		}

		root := ts.scheme + "://" + authority + "/"
		if ts.scheme == "https" {
			httpsRoot = root
		}

		err = FilterDenyList(root, p.denyList, deadline)
		if err != nil {
			return OutboundDecision{}, err
		}
	}

	return p.decide(ctx, httpsRoot, p.allowList, p.denyList, deadline)
}

// maxConcurrentTunnels is the default for [OutboundProxy.maxTunnels]. See
// [OutboundProxy.handleConnect] for how the value is derived.
const maxConcurrentTunnels = 512

// acquireTunnel reserves a slot for one CONNECT handler, reporting false when
// the proxy is already at [OutboundProxy.maxTunnels]. The compare-and-swap loop
// keeps the check and the increment atomic, so concurrent handlers cannot
// overshoot the ceiling between them.
func (p *OutboundProxy) acquireTunnel() bool {
	for {
		current := p.tunnels.Load()
		if current >= p.maxTunnels {
			return false
		}
		if p.tunnels.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

// releaseTunnel returns a slot taken by [OutboundProxy.acquireTunnel].
func (p *OutboundProxy) releaseTunnel() {
	p.tunnels.Add(-1)
}

// spliceIdleTimeout bounds a CONNECT tunnel in which no byte has moved in
// either direction.
//
// Nothing else bounds one. The hijacked connections carry no deadline: the
// server clears the header read deadline once the request line is in, and
// net.Dialer.Timeout only covers the connect. net/http also untracks a
// connection once it is hijacked, so neither Server.Shutdown nor a converter
// restart reaps it. Left alone, an upstream that accepts the tunnel and then
// answers nothing holds two goroutines and two sockets until the process dies.
//
// Sized well above any legitimate pause between a request and its response, so
// a slow origin is never cut off. A transfer that keeps making progress
// refreshes the deadline and runs for as long as it needs.
const spliceIdleTimeout = 2 * time.Minute

// spliceTunnel copies bytes between the two ends of a CONNECT tunnel until
// both directions finish, the tunnel sits idle for idleTimeout, or closing is
// closed because the proxy is shutting down. Callers pass
// [spliceIdleTimeout]; only tests shorten it.
//
// Each direction half-closes its destination once its source reaches EOF, so a
// peer that waits for the request to end before answering still sees the EOF.
// Idleness is tracked across both directions rather than per direction: the
// client sends nothing for the length of a download, and half-closing its write
// side then would tell the origin the client had gone away.
func spliceTunnel(client, upstream net.Conn, closing <-chan struct{}, idleTimeout time.Duration) {
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		copyTracking(upstream, client, &lastActivity, idleTimeout)
		if cw, ok := upstream.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		copyTracking(client, upstream, &lastActivity, idleTimeout)
		if cw, ok := client.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	ticker := time.NewTicker(idleTimeout / 4)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-closing:
		case <-ticker.C:
			if time.Since(time.Unix(0, lastActivity.Load())) < idleTimeout {
				continue
			}
		}

		// Closing both ends unblocks whichever copy is still reading. The
		// caller's own deferred Close calls then become no-ops.
		_ = client.Close()
		_ = upstream.Close()
		<-done

		return
	}
}

// copyTracking copies src into dst, recording the time of every chunk that
// moves so [spliceTunnel] can tell a busy tunnel from an idle one.
func copyTracking(dst, src net.Conn, lastActivity *atomic.Int64, writeTimeout time.Duration) {
	buf := make([]byte, 32*1024)

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			lastActivity.Store(time.Now().UnixNano())

			// Bound the write. A destination that has gone away accepts the
			// first chunk into its send buffer and only fails on the next one,
			// so without a deadline this direction keeps a dead tunnel alive
			// for one more chunk. A destination that stops reading altogether
			// would block here forever.
			_ = dst.SetWriteDeadline(time.Now().Add(writeTimeout))

			_, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return
			}

			lastActivity.Store(time.Now().UnixNano())
		}
		if readErr != nil {
			return
		}
	}
}

// handleForward handles plain HTTP requests sent to the proxy as absolute
// URIs (GET http://host/path). The proxy revalidates the URL, then
// forwards the request via a transport that dials the pinned IP.
func (p *OutboundProxy) handleForward(w http.ResponseWriter, req *http.Request) {
	if req.URL == nil || req.URL.Scheme == "" || req.URL.Host == "" {
		http.Error(w, "absolute URL required", http.StatusBadRequest)
		return
	}

	deadline, ok := req.Context().Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}

	decision, err := p.decide(req.Context(), req.URL.String(), p.allowList, p.denyList, deadline)
	if err != nil {
		switch {
		case isClientCancellation(req.Context(), err):
			p.logger.DebugContext(req.Context(), fmt.Sprintf("forward abandoned by client for '%s': %s", RedactURL(req.URL.String()), err))
		case p.logRefusal != nil && FilterRule(err) != "":
			p.logRefusal(req.Context(), req.URL.String(), err)
		default:
			p.logger.WarnContext(req.Context(), fmt.Sprintf("forward blocked for '%s': %s", RedactURL(req.URL.String()), err))
		}
		http.Error(w, "request blocked", http.StatusForbidden)
		return
	}

	var proxyURL *url.URL
	if p.upstreamProxy != nil {
		proxyURL, err = p.upstreamProxy(req.URL)
		if err != nil {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("resolve upstream proxy for '%s': %s", RedactURL(req.URL.String()), err))
			http.Error(w, "upstream proxy error", http.StatusBadGateway)
			return
		}
	}

	outReq := req.Clone(req.Context())
	outReq.RequestURI = ""
	stripHopByHopHeaders(outReq.Header)

	// Build a fresh transport per request. The decision contains the pinned
	// IPs to dial; reusing a transport across requests would leak the
	// decision's closure across unrelated targets.
	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	if proxyURL != nil {
		// The upstream proxy owns DNS and egress; Go adds Proxy-Authorization
		// from the URL's credentials. The decision above already gated the
		// destination, and dialBypass dials the proxy host directly.
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.DialContext = p.dialBypass
	} else {
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// A host-resolver rule may send the request to another host and
			// port; the request itself keeps its URL and Host header.
			if decision.Target != "" {
				addr = decision.Target
			}
			_, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, fmt.Errorf("split forward addr %q: %w", addr, splitErr)
			}
			switch {
			case decision.Bypass:
				return p.dialBypass(ctx, network, addr)
			case len(decision.Pinned) > 0:
				return p.dialPinned(ctx, network, decision.Pinned, port)
			default:
				return nil, errors.New("no pinned addresses and not bypassed")
			}
		}
	}
	defer transport.CloseIdleConnections()

	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		if isClientCancellation(req.Context(), err) {
			p.logger.DebugContext(req.Context(), fmt.Sprintf("forward RoundTrip abandoned by client for '%s': %s", RedactURL(req.URL.String()), err))
		} else {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("forward RoundTrip failed for '%s': %s", RedactURL(req.URL.String()), err))
		}
		resetClient(w)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	stripHopByHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// Per RFC 7230 section 6.1.
var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Proxy-Connection",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func stripHopByHopHeaders(h http.Header) {
	for _, name := range hopByHopHeaders {
		h.Del(name)
	}
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

// isClientCancellation reports whether err originates from the client (for
// example Chromium) closing the connection or letting the request deadline
// pass before the proxy could finish validating the destination. Such
// errors are not policy refusals: the proxy never reached an allow/deny
// rule decision. Callers downgrade these to debug to avoid alarming
// operators with noise from speculative or aborted browser requests. The
// canonical case is a Chromium DNS prefetch that the browser drops before
// the proxy's [resolveHost] call returns. The [net.DNSError]
// returned by [net.Resolver.LookupNetIP] unwraps to [context.Canceled] or
// [context.DeadlineExceeded] in that case, so an [errors.Is] walk catches
// it.
func isClientCancellation(ctx context.Context, err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return ctx.Err() != nil
}

// dialThroughUpstreamProxy tunnels to target through the upstream proxy,
// letting [DialThroughProxy] perform the authenticated CONNECT that
// the converter cannot. dialBypass dials the proxy itself and is overridable in
// tests. See https://github.com/gotenberg/gotenberg/issues/1592.
func (p *OutboundProxy) dialThroughUpstreamProxy(ctx context.Context, proxyURL *url.URL, target string) (net.Conn, error) {
	return DialThroughProxy(ctx, proxyURL, target, p.dialBypass)
}

// resetClient aborts the client connection with a TCP reset instead of
// answering. A converter treats a proxy response as the origin's own, so a
// 502 would reach the document as an ordinary HTTP error; a reset surfaces as a
// network failure, as it would without the proxy.
func resetClient(w http.ResponseWriter) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetLinger(0)
	}
	_ = conn.Close()
}
