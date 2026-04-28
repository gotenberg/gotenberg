package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dlclark/regexp2"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// outboundProxyOptions configures a [libreOfficeProxy].
type outboundProxyOptions struct {
	allowList      []*regexp2.Regexp
	denyList       []*regexp2.Regexp
	denyPrivateIPs bool
	denyPublicIPs  bool
}

// libreOfficeProxy is an HTTP/HTTPS forward proxy that LibreOffice routes
// outbound requests through. Every proxied request goes through
// [gotenberg.DecideOutbound] so the same allow/deny lists and IP-class
// filters that protect chromium and webhook fetches also apply to
// soffice's own libcurl-driven fetches.
//
// soffice triggers an outbound request whenever a document references
// external content (OOXML images via TargetMode="External", RTF
// INCLUDEPICTURE, ODT linked images). Without a filtering proxy in the
// path those fetches bypass every Go-side SSRF guard because they
// originate inside the soffice subprocess.
type libreOfficeProxy struct {
	listener net.Listener
	server   *http.Server
	client   *http.Client
	opts     outboundProxyOptions
	logger   *slog.Logger

	stopOnce sync.Once
}

// newLibreOfficeProxy binds a proxy listener to a free local port and
// applies opts to every proxied request. Callers must call [Start]
// before pointing soffice at the proxy and [Stop] on shutdown.
func newLibreOfficeProxy(logger *slog.Logger, opts outboundProxyOptions) (*libreOfficeProxy, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("bind LibreOffice proxy listener: %w", err)
	}

	decideOpts := []gotenberg.DecideOption{
		gotenberg.WithDenyPrivateIPs(opts.denyPrivateIPs),
		gotenberg.WithDenyPublicIPs(opts.denyPublicIPs),
	}

	p := &libreOfficeProxy{
		listener: listener,
		client:   gotenberg.NewOutboundHttpClient(0, opts.allowList, opts.denyList, decideOpts...),
		opts:     opts,
		logger:   logger.With(slog.String("logger", "libreoffice-proxy")),
	}
	p.server = &http.Server{
		Handler:           p,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return p, nil
}

// Addr returns the host:port the proxy listens on.
func (p *libreOfficeProxy) Addr() string {
	return p.listener.Addr().String()
}

// Start serves proxy requests in a background goroutine until [Stop] is
// called.
func (p *libreOfficeProxy) Start() {
	go func() {
		err := p.server.Serve(p.listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.logger.ErrorContext(context.Background(), fmt.Sprintf("LibreOffice proxy serve: %s", err))
		}
	}()
}

// Stop gracefully shuts the proxy down. Subsequent calls are no-ops.
func (p *libreOfficeProxy) Stop(ctx context.Context) error {
	var err error
	p.stopOnce.Do(func() {
		err = p.server.Shutdown(ctx)
	})
	if err != nil {
		return fmt.Errorf("shutdown LibreOffice proxy: %w", err)
	}
	return nil
}

// ServeHTTP dispatches between CONNECT (HTTPS tunnels) and the absolute
// URL form (HTTP forward).
func (p *libreOfficeProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
		return
	}
	p.handleHttp(w, r)
}

// handleHttp forwards a plain HTTP request whose URL line is absolute
// (RFC 7230 5.3.2) through the outbound HTTP client, which validates
// the destination and pins the dial.
func (p *libreOfficeProxy) handleHttp(w http.ResponseWriter, r *http.Request) {
	if r.URL == nil || !r.URL.IsAbs() {
		http.Error(w, "proxy: expected absolute URI", http.StatusBadRequest)
		return
	}

	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	removeHopByHopHeaders(outReq.Header)

	// gosec G704: outReq.URL is exactly what the proxy is here to filter; the
	// http.Client returned by NewOutboundHttpClient validates and pins it.
	resp, err := p.client.Do(outReq) //nolint:gosec
	if err != nil {
		p.logger.WarnContext(r.Context(), fmt.Sprintf("LibreOffice proxy rejected forward to '%s': %s", r.URL.String(), err))
		http.Error(w, "proxy: destination rejected", http.StatusForbidden)
		return
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			p.logger.DebugContext(r.Context(), fmt.Sprintf("close upstream response body: %s", closeErr))
		}
	}()

	removeHopByHopHeaders(resp.Header)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, copyErr := io.Copy(w, resp.Body)
	if copyErr != nil {
		p.logger.DebugContext(r.Context(), fmt.Sprintf("copy proxied response body: %s", copyErr))
	}
}

// handleConnect implements an HTTPS tunnel. It validates the destination
// host through [gotenberg.DecideOutbound] (synthesizing an https URL),
// dials the pinned IPs returned by the decision, and splices bytes
// between client and server.
func (p *libreOfficeProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		http.Error(w, "proxy: invalid CONNECT target", http.StatusBadRequest)
		return
	}

	deadline, ok := r.Context().Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}

	rawURL := (&url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}).String()

	decision, err := gotenberg.DecideOutbound(r.Context(), rawURL, p.opts.allowList, p.opts.denyList, deadline,
		gotenberg.WithDenyPrivateIPs(p.opts.denyPrivateIPs),
		gotenberg.WithDenyPublicIPs(p.opts.denyPublicIPs),
	)
	if err != nil {
		p.logger.WarnContext(r.Context(), fmt.Sprintf("LibreOffice proxy rejected CONNECT to '%s': %s", rawURL, err))
		http.Error(w, "proxy: destination rejected", http.StatusForbidden)
		return
	}

	var dest net.Conn
	switch {
	case len(decision.Pinned) > 0:
		dest, err = gotenberg.DialPinned(r.Context(), "tcp", decision.Pinned, port)
	default:
		// Bypass (allow-list match) or non-http-like scheme: dial directly.
		// gosec G704: host:port has cleared DecideOutbound above.
		dest, err = net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second) //nolint:gosec
	}
	if err != nil {
		p.logger.WarnContext(r.Context(), fmt.Sprintf("LibreOffice proxy CONNECT dial to '%s' failed: %s", rawURL, err))
		http.Error(w, "proxy: dial failed", http.StatusBadGateway)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		_ = dest.Close()
		http.Error(w, "proxy: hijack unsupported", http.StatusInternalServerError)
		return
	}
	client, _, err := hijacker.Hijack()
	if err != nil {
		_ = dest.Close()
		p.logger.WarnContext(r.Context(), fmt.Sprintf("LibreOffice proxy hijack failed: %s", err))
		return
	}

	_, writeErr := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if writeErr != nil {
		_ = client.Close()
		_ = dest.Close()
		return
	}

	go pipeAndClose(client, dest)
	go pipeAndClose(dest, client)
}

// pipeAndClose copies bytes from src to dst and closes both ends when
// the copy finishes.
func pipeAndClose(dst, src net.Conn) {
	defer func() {
		_ = dst.Close()
		_ = src.Close()
	}()
	_, _ = io.Copy(dst, src)
}

// hopByHopHeaders is the set of hop-by-hop headers from RFC 7230 6.1
// plus the ones soffice adds when acting as a forward-proxy client.
var hopByHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// sofficeProxyConfigTmpl is the registrymodifications.xcu fragment that
// tells soffice's UCB layer to route every HTTP and HTTPS fetch through
// proxyHost:proxyPort. The %s placeholders accept the proxy host and
// port respectively (host first, port second, repeated for HTTP and
// HTTPS).
const sofficeProxyConfigTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<oor:items xmlns:oor="http://openoffice.org/2001/registry" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetProxyType" oor:op="fuse"><value>1</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPProxyName" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPProxyPort" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPSProxyName" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetHTTPSProxyPort" oor:op="fuse"><value>%s</value></prop></item>
  <item oor:path="/org.openoffice.Inet/Settings"><prop oor:name="ooInetNoProxy" oor:op="fuse"><value></value></prop></item>
</oor:items>
`

// writeSofficeProxyConfig drops a registrymodifications.xcu file into
// userProfileDirPath/user/ that points soffice's UCB layer at proxyAddr
// for both HTTP and HTTPS. proxyAddr must be a host:port pair.
func writeSofficeProxyConfig(userProfileDirPath, proxyAddr string) error {
	host, port, err := net.SplitHostPort(proxyAddr)
	if err != nil {
		return fmt.Errorf("split proxy address %q: %w", proxyAddr, err)
	}

	userDir := userProfileDirPath + "/user"
	err = os.MkdirAll(userDir, 0o755)
	if err != nil {
		return fmt.Errorf("create soffice user profile directory: %w", err)
	}

	body := fmt.Sprintf(sofficeProxyConfigTmpl, host, port, host, port)
	err = os.WriteFile(userDir+"/registrymodifications.xcu", []byte(body), 0o600)
	if err != nil {
		return fmt.Errorf("write registrymodifications.xcu: %w", err)
	}

	return nil
}

// sofficeProxyEnv overlays http_proxy/https_proxy on env so soffice's
// libcurl path also routes through proxyAddr. The environment variables
// supplement the registrymodifications.xcu config so coverage stays
// intact if soffice upgrades and one of the two paths regresses.
func sofficeProxyEnv(env []string, proxyAddr string) []string {
	proxyURL := "http://" + proxyAddr

	filtered := env[:0:0]
	for _, kv := range env {
		switch strings.ToLower(strings.SplitN(kv, "=", 2)[0]) {
		case "http_proxy", "https_proxy", "no_proxy":
			continue
		}
		filtered = append(filtered, kv)
	}

	return append(filtered,
		"http_proxy="+proxyURL,
		"https_proxy="+proxyURL,
		"HTTP_PROXY="+proxyURL,
		"HTTPS_PROXY="+proxyURL,
		"no_proxy=",
		"NO_PROXY=",
	)
}

func removeHopByHopHeaders(h http.Header) {
	if connection := h.Get("Connection"); connection != "" {
		for name := range strings.SplitSeq(connection, ",") {
			h.Del(strings.TrimSpace(name))
		}
	}
	for _, name := range hopByHopHeaders {
		h.Del(name)
	}
}
