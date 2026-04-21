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
	"sync"
	"time"

	"github.com/dlclark/regexp2"

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

	listener net.Listener
	server   *http.Server
	wg       sync.WaitGroup

	logger  *slog.Logger
	started bool
	mu      sync.Mutex
}

// newPinningProxy returns a pinning proxy configured with the given
// allow/deny lists. When allowPrivateIPs is true, the proxy skips the
// public-IP filter while still pinning resolved IPs to the dial. The
// returned proxy is not yet listening; call Start.
func newPinningProxy(allowList, denyList []*regexp2.Regexp, allowPrivateIPs bool) *pinningProxy {
	return &pinningProxy{
		allowList: allowList,
		denyList:  denyList,
		decide: func(ctx context.Context, rawURL string, allow, deny []*regexp2.Regexp, deadline time.Time) (gotenberg.OutboundDecision, error) {
			return gotenberg.DecideOutbound(ctx, rawURL, allow, deny, deadline, gotenberg.WithAllowPrivateIPs(allowPrivateIPs))
		},
		dialPinned: gotenberg.DialPinned,
		dialBypass: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, network, addr)
		},
	}
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
	p.started = false
	p.mu.Unlock()

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
		p.logger.WarnContext(req.Context(), fmt.Sprintf("CONNECT blocked for '%s': %s", req.Host, err))
		http.Error(w, "CONNECT blocked", http.StatusForbidden)
		return
	}

	var upstream net.Conn
	switch {
	case decision.Bypass:
		upstream, err = p.dialBypass(req.Context(), "tcp", req.Host)
	case len(decision.Pinned) > 0:
		upstream, err = p.dialPinned(req.Context(), "tcp", decision.Pinned, port)
	default:
		err = errors.New("no pinned addresses and not bypassed")
	}
	if err != nil {
		p.logger.WarnContext(req.Context(), fmt.Sprintf("CONNECT dial failed for '%s': %s", req.Host, err))
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
		p.logger.WarnContext(req.Context(), fmt.Sprintf("write CONNECT ack: %s", err))
		return
	}

	// Splice bytes in both directions until either side closes.
	var splice sync.WaitGroup
	splice.Add(2)
	go func() {
		defer splice.Done()
		_, _ = io.Copy(upstream, client)
		if cw, ok := upstream.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}()
	go func() {
		defer splice.Done()
		_, _ = io.Copy(client, upstream)
		if cw, ok := client.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}()
	splice.Wait()
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
		p.logger.WarnContext(req.Context(), fmt.Sprintf("forward blocked for '%s': %s", req.URL, err))
		http.Error(w, "request blocked", http.StatusForbidden)
		return
	}

	outReq := req.Clone(req.Context())
	outReq.RequestURI = ""
	stripHopByHopHeaders(outReq.Header)

	transport := &http.Transport{
		// Build a fresh transport per request. The decision contains the
		// pinned IPs to dial; reusing a transport across requests would
		// leak the decision's closure across unrelated targets.
		DisableKeepAlives: true,
		Proxy:             nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
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
		},
	}
	defer transport.CloseIdleConnections()

	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		p.logger.WarnContext(req.Context(), fmt.Sprintf("forward RoundTrip failed for '%s': %s", req.URL, err))
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
