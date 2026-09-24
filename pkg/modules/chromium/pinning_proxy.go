package chromium

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
	"sync"
	"sync/atomic"
	"time"

	"github.com/dlclark/regexp2"
	"golang.org/x/net/http/httpproxy"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// pinningProxy is a loopback-bound HTTP/1.1 forward and CONNECT proxy
// placed between Chromium and the outbound network. It runs the same
// allow/deny/IP-public validation as [gotenberg.FilterOutboundURL] on
// every request and dials the destination using the IPs resolved at that
// moment. Routing Chromium through this proxy eliminates the Chromium-side
// DNS lookup that otherwise opens a DNS rebinding window between
// Gotenberg's validation and Chromium's TCP connect.
//
// The proxy is transparent to the caller. HTTPS sub-resources tunnel
// through CONNECT with Chromium performing its own TLS handshake using
// the original hostname, preserving SNI and certificate validation.
type pinningProxy struct {
	allowList []*regexp2.Regexp
	denyList  []*regexp2.Regexp

	// decide resolves and validates a URL. Tests may override it.
	decide func(ctx context.Context, rawURL string, allowList, denyList []*regexp2.Regexp, deadline time.Time) (gotenberg.OutboundDecision, error)

	// dialPinned dials the pinned IPs for a decision. Tests may override
	// it to connect to a stub upstream regardless of decision.
	dialPinned func(ctx context.Context, network string, addrs []netip.Addr, port string) (net.Conn, error)

	// dialBypass dials the destination hostname directly (operator
	// allow-list opt-in). Tests may override it.
	dialBypass func(ctx context.Context, network, addr string) (net.Conn, error)

	// upstreamProxy resolves the upstream (corporate) proxy for a
	// destination URL from the standard proxy environment variables, or
	// returns a nil URL to connect directly. It is nil unless the operator
	// opted into proxy-environment honoring. When set, the pinning proxy
	// performs the authenticated proxy handshake that Chromium cannot. See
	// https://github.com/gotenberg/gotenberg/issues/1592.
	upstreamProxy func(*url.URL) (*url.URL, error)

	listener net.Listener
	server   *http.Server
	wg       sync.WaitGroup

	// closing is closed by Stop to force in-flight CONNECT tunnels shut.
	// [http.Server.Shutdown] cannot do it: net/http untracks a connection once
	// a handler hijacks it, so a tunnel would otherwise outlive the proxy that
	// created it. Recreated on every Start.
	closing chan struct{}

	// maxTunnels ceilings the CONNECT handlers in flight. Tests may lower it.
	maxTunnels int64

	// tunnels counts the CONNECT handlers in flight.
	tunnels atomic.Int64

	logger  *slog.Logger
	started bool
	mu      sync.Mutex
}

// newPinningProxy returns a pinning proxy configured with the given
// allow/deny lists and IP-class policy. The policy bools are applied via
// [gotenberg.DecideOutbound] on every request the proxy sees, so
// Chromium inherits whatever posture the operator selected. The
// returned proxy is not yet listening; call Start.
func newPinningProxy(allowList, denyList []*regexp2.Regexp, denyPrivateIPs, denyPublicIPs, enableEnvironmentProxy bool) *pinningProxy {
	p := &pinningProxy{
		allowList: allowList,
		denyList:  denyList,
		decide: func(ctx context.Context, rawURL string, allow, deny []*regexp2.Regexp, deadline time.Time) (gotenberg.OutboundDecision, error) {
			return gotenberg.DecideOutbound(ctx, rawURL, allow, deny, deadline,
				gotenberg.WithDenyPrivateIPs(denyPrivateIPs),
				gotenberg.WithDenyPublicIPs(denyPublicIPs),
			)
		},
		dialPinned: gotenberg.DialPinned,
		dialBypass: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, network, addr)
		},
		maxTunnels: maxConcurrentTunnels,
	}

	if enableEnvironmentProxy {
		// Honor the standard proxy environment variables, credentials
		// included. httpproxy reads the environment now and applies NO_PROXY.
		p.upstreamProxy = httpproxy.FromEnvironment().ProxyFunc()
	}

	return p
}

// Start binds the proxy to 127.0.0.1 on an ephemeral port and serves in a
// background goroutine. Bind failures return an error; the caller must
// not proceed to start Chromium with --proxy-server.
func (p *pinningProxy) Start(logger *slog.Logger) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return errors.New("pinning proxy already started")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("bind pinning proxy: %w", err)
	}

	p.listener = l
	p.closing = make(chan struct{})
	p.logger = logger.With(slog.String("logger", "pinning-proxy"))
	p.server = &http.Server{
		Handler: http.HandlerFunc(p.serveHTTP),
		// Guard against slow header attacks. Body reads are controlled
		// per-handler.
		ReadHeaderTimeout: 15 * time.Second,
		ErrorLog:          slog.NewLogLogger(p.logger.Handler(), slog.LevelWarn),
	}

	p.wg.Go(func() {
		serveErr := p.server.Serve(l)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			p.logger.ErrorContext(context.Background(), fmt.Sprintf("pinning proxy serve: %s", serveErr))
		}
	})

	p.started = true
	p.logger.DebugContext(context.Background(), fmt.Sprintf("pinning proxy listening on %s", l.Addr()))
	return nil
}

// Stop shuts the proxy down and waits for in-flight handlers to complete.
// Safe to call on a non-started proxy.
func (p *pinningProxy) Stop(logger *slog.Logger) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	srv := p.server
	closing := p.closing
	p.closing = nil
	p.started = false
	p.mu.Unlock()

	// Force in-flight tunnels shut before draining the server. Shutdown does
	// not reach them, so a tunnel whose upstream never answers would otherwise
	// survive the proxy, and with it every Chromium restart.
	if closing != nil {
		close(closing)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(ctx)
	p.wg.Wait()

	if shutdownErr != nil {
		return fmt.Errorf("shutdown pinning proxy: %w", shutdownErr)
	}
	logger.DebugContext(context.Background(), "pinning proxy stopped")
	return nil
}

// URL returns the proxy URL suitable for Chromium's --proxy-server flag.
// Returns an empty string when the proxy is not listening.
func (p *pinningProxy) URL() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener == nil {
		return ""
	}
	return "http://" + p.listener.Addr().String()
}

func (p *pinningProxy) serveHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		p.handleConnect(w, req)
		return
	}
	p.handleForward(w, req)
}

// handleConnect handles HTTPS (and any other CONNECT) tunnels. Chromium
// issues CONNECT host:port; the proxy validates the host, dials the
// pinned IP, and splices the client socket with the upstream socket.
// Chromium then negotiates TLS end-to-end with the original hostname in
// SNI.
func (p *pinningProxy) handleConnect(w http.ResponseWriter, req *http.Request) {
	// A ceiling, not a tuning knob: it bounds what a tunnel that refuses to end
	// can accumulate, whatever keeps it alive. [spliceIdleTimeout] ends a silent
	// tunnel, but a peer trickling a byte just under it stays "active" forever,
	// and a compromised renderer can hold the client side open to match.
	//
	// Chromium caps itself well below this. Its socket pool manager allows 128
	// sockets per proxy chain for normal traffic plus 128 for WebSocket
	// traffic, and every request Gotenberg's Chromium makes traverses this one
	// proxy chain, so an honest browser cannot exceed 256 tunnels here. At
	// double that, a real page never meets the ceiling and a hostile one stops
	// at it.
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

	// The validation URL uses https:// so that http-like scheme checks
	// apply in [gotenberg.DecideOutbound]. The scheme does not influence
	// the CONNECT handling beyond filtering.
	decision, err := p.decide(req.Context(), "https://"+req.Host, p.allowList, p.denyList, deadline)
	if err != nil {
		if isClientCancellation(req.Context(), err) {
			p.logger.DebugContext(req.Context(), fmt.Sprintf("CONNECT abandoned by client for '%s': %s", req.Host, err))
		} else {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("CONNECT blocked for '%s': %s", req.Host, err))
		}
		http.Error(w, "CONNECT blocked", http.StatusForbidden)
		return
	}

	// When the operator routes egress through an authenticated proxy,
	// Chromium cannot supply the credentials itself, so the pinning proxy
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

	var upstream net.Conn
	switch {
	case proxyURL != nil:
		upstream, err = p.dialThroughUpstreamProxy(req.Context(), proxyURL, req.Host)
	case decision.Bypass:
		upstream, err = p.dialBypass(req.Context(), "tcp", req.Host)
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

	p.mu.Lock()
	closing := p.closing
	p.mu.Unlock()

	spliceTunnel(client, upstream, closing, spliceIdleTimeout)
}

// maxConcurrentTunnels is the default for [pinningProxy.maxTunnels]. See
// [pinningProxy.handleConnect] for how the value is derived.
const maxConcurrentTunnels = 512

// acquireTunnel reserves a slot for one CONNECT handler, reporting false when
// the proxy is already at [pinningProxy.maxTunnels]. The compare-and-swap loop
// keeps the check and the increment atomic, so concurrent handlers cannot
// overshoot the ceiling between them.
func (p *pinningProxy) acquireTunnel() bool {
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

// releaseTunnel returns a slot taken by [pinningProxy.acquireTunnel].
func (p *pinningProxy) releaseTunnel() {
	p.tunnels.Add(-1)
}

// spliceIdleTimeout bounds a CONNECT tunnel in which no byte has moved in
// either direction.
//
// Nothing else bounds one. The hijacked connections carry no deadline: the
// server clears the header read deadline once the request line is in, and
// net.Dialer.Timeout only covers the connect. net/http also untracks a
// connection once it is hijacked, so neither Server.Shutdown nor a Chromium
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
func (p *pinningProxy) handleForward(w http.ResponseWriter, req *http.Request) {
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
		if isClientCancellation(req.Context(), err) {
			p.logger.DebugContext(req.Context(), fmt.Sprintf("forward abandoned by client for '%s': %s", req.URL, err))
		} else {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("forward blocked for '%s': %s", req.URL, err))
		}
		http.Error(w, "request blocked", http.StatusForbidden)
		return
	}

	var proxyURL *url.URL
	if p.upstreamProxy != nil {
		proxyURL, err = p.upstreamProxy(req.URL)
		if err != nil {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("resolve upstream proxy for '%s': %s", req.URL.Redacted(), err))
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
			p.logger.DebugContext(req.Context(), fmt.Sprintf("forward RoundTrip abandoned by client for '%s': %s", req.URL, err))
		} else {
			p.logger.WarnContext(req.Context(), fmt.Sprintf("forward RoundTrip failed for '%s': %s", req.URL, err))
		}
		http.Error(w, "upstream error", http.StatusBadGateway)
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
// the proxy's [outbound.resolveHost] call returns. The [net.DNSError]
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
// letting [gotenberg.DialThroughProxy] perform the authenticated CONNECT that
// Chromium cannot. dialBypass dials the proxy itself and is overridable in
// tests. See https://github.com/gotenberg/gotenberg/issues/1592.
func (p *pinningProxy) dialThroughUpstreamProxy(ctx context.Context, proxyURL *url.URL, target string) (net.Conn, error) {
	return gotenberg.DialThroughProxy(ctx, proxyURL, target, p.dialBypass)
}
