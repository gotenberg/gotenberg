package chromium

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dlclark/regexp2"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

// newRawTCPServer starts a TCP server on 127.0.0.1:0 that calls handle for
// every accepted connection. It returns the listener address and a cleanup
// function.
func newRawTCPServer(t *testing.T, handle func(net.Conn)) (string, func()) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go handle(conn)
		}
	}()

	return l.Addr().String(), func() { _ = l.Close() }
}

// newProxyForTest returns a pinning proxy whose decide and dial functions
// are set to test stubs. The proxy is started on a loopback ephemeral
// port and stopped during test cleanup.
func newProxyForTest(t *testing.T, p *pinningProxy) string {
	t.Helper()
	err := p.Start(testLogger())
	if err != nil {
		t.Fatalf("start pinning proxy: %v", err)
	}
	t.Cleanup(func() {
		_ = p.Stop(testLogger())
	})
	return p.URL()
}

func TestPinningProxy_Forward_Pinned_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "example.com" {
			t.Errorf("upstream expected Host=example.com, got %q", r.Host)
		}
		_, _ = fmt.Fprint(w, "hello-from-upstream")
	}))
	t.Cleanup(upstream.Close)
	upstreamURL := mustParseURL(t, upstream.URL)

	var decideCalls atomic.Int32
	p := newPinningProxy(nil, nil, false, false)
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		decideCalls.Add(1)
		return gotenberg.OutboundDecision{Pinned: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}, nil
	}
	p.dialPinned = func(ctx context.Context, network string, _ []netip.Addr, _ string) (net.Conn, error) {
		return net.Dial(network, upstreamURL.Host)
	}
	proxyURL := newProxyForTest(t, p)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(t, proxyURL)),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://example.com/")
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(body) != "hello-from-upstream" {
		t.Fatalf("body = %q, want %q", body, "hello-from-upstream")
	}
	if got := decideCalls.Load(); got != 1 {
		t.Fatalf("decide called %d times, want 1", got)
	}
}

func TestPinningProxy_Forward_BlockedByDecide(t *testing.T) {
	p := newPinningProxy(nil, nil, false, false)
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		return gotenberg.OutboundDecision{}, fmt.Errorf("nope: %w", gotenberg.ErrFiltered)
	}
	p.dialPinned = func(_ context.Context, _ string, _ []netip.Addr, _ string) (net.Conn, error) {
		t.Fatal("dialPinned must not be called when decide returns an error")
		return nil, errors.New("unreachable")
	}
	proxyURL := newProxyForTest(t, p)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(t, proxyURL)),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://blocked.example/")
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestPinningProxy_Forward_Bypass(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "bypassed")
	}))
	t.Cleanup(upstream.Close)
	upstreamURL := mustParseURL(t, upstream.URL)

	var bypassCalls atomic.Int32
	p := newPinningProxy(nil, nil, false, false)
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		return gotenberg.OutboundDecision{Bypass: true}, nil
	}
	p.dialBypass = func(_ context.Context, network, _ string) (net.Conn, error) {
		bypassCalls.Add(1)
		return net.Dial(network, upstreamURL.Host)
	}
	p.dialPinned = func(_ context.Context, _ string, _ []netip.Addr, _ string) (net.Conn, error) {
		t.Fatal("dialPinned must not be called on bypass")
		return nil, errors.New("unreachable")
	}
	proxyURL := newProxyForTest(t, p)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(t, proxyURL)),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://internal.example/")
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := bypassCalls.Load(); got != 1 {
		t.Fatalf("dialBypass called %d times, want 1", got)
	}
}

func TestPinningProxy_Forward_StripsHopByHopHeaders(t *testing.T) {
	var upstreamSawProxyAuth bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" {
			upstreamSawProxyAuth = true
		}
		w.Header().Set("Connection", "close")
		w.Header().Set("Proxy-Connection", "close")
		w.Header().Set("X-Downstream", "ok")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)
	upstreamURL := mustParseURL(t, upstream.URL)

	p := newPinningProxy(nil, nil, false, false)
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		return gotenberg.OutboundDecision{Pinned: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}, nil
	}
	p.dialPinned = func(ctx context.Context, network string, _ []netip.Addr, _ string) (net.Conn, error) {
		return net.Dial(network, upstreamURL.Host)
	}
	proxyURL := newProxyForTest(t, p)

	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Proxy-Authorization", "Basic Zm9vOmJhcg==")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(t, proxyURL)),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	defer resp.Body.Close()

	if upstreamSawProxyAuth {
		t.Fatalf("upstream received Proxy-Authorization, proxy did not strip it")
	}
	if resp.Header.Get("Proxy-Connection") != "" {
		t.Fatalf("response retained Proxy-Connection, proxy did not strip it")
	}
	if resp.Header.Get("X-Downstream") != "ok" {
		t.Fatalf("response missing X-Downstream header")
	}
}

func TestPinningProxy_Forward_RejectsNonAbsoluteURL(t *testing.T) {
	p := newPinningProxy(nil, nil, false, false)
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		t.Fatal("decide must not be called for malformed proxy request")
		return gotenberg.OutboundDecision{}, nil
	}
	proxyURL := newProxyForTest(t, p)

	conn, err := net.Dial("tcp", strings.TrimPrefix(proxyURL, "http://"))
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	// Send a request with a path-only target, not an absolute URI, which
	// the proxy should reject with 400.
	_, err = fmt.Fprint(conn, "GET /path HTTP/1.1\r\nHost: example.com\r\n\r\n")
	if err != nil {
		t.Fatalf("write request: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestPinningProxy_CONNECT_Pinned_Success(t *testing.T) {
	upstreamAddr, stop := newRawTCPServer(t, func(c net.Conn) {
		defer c.Close()
		_, _ = c.Write([]byte("HI"))
		buf := make([]byte, 4)
		n, _ := io.ReadFull(c, buf)
		_, _ = c.Write(buf[:n])
	})
	t.Cleanup(stop)

	var decideCalls atomic.Int32
	p := newPinningProxy(nil, nil, false, false)
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		decideCalls.Add(1)
		return gotenberg.OutboundDecision{Pinned: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}, nil
	}
	p.dialPinned = func(_ context.Context, network string, _ []netip.Addr, _ string) (net.Conn, error) {
		return net.Dial(network, upstreamAddr)
	}
	proxyURL := newProxyForTest(t, p)

	// Connect to the proxy, send CONNECT, splice raw bytes.
	conn, err := net.Dial("tcp", strings.TrimPrefix(proxyURL, "http://"))
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	_, err = fmt.Fprintf(conn, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n")
	if err != nil {
		t.Fatalf("write CONNECT: %v", err)
	}

	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !strings.Contains(statusLine, " 200 ") {
		t.Fatalf("CONNECT status = %q, want 200", statusLine)
	}
	// Consume the blank line after headers.
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("read headers: %v", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	hi := make([]byte, 2)
	_, err = io.ReadFull(br, hi)
	if err != nil {
		t.Fatalf("read greeting: %v", err)
	}
	if string(hi) != "HI" {
		t.Fatalf("greeting = %q, want HI", hi)
	}

	_, err = conn.Write([]byte("PONG"))
	if err != nil {
		t.Fatalf("write PONG: %v", err)
	}
	echo := make([]byte, 4)
	_, err = io.ReadFull(br, echo)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(echo) != "PONG" {
		t.Fatalf("echo = %q, want PONG", echo)
	}
	if got := decideCalls.Load(); got != 1 {
		t.Fatalf("decide called %d times, want 1", got)
	}
}

func TestPinningProxy_CONNECT_BlockedByDecide(t *testing.T) {
	p := newPinningProxy(nil, nil, false, false)
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		return gotenberg.OutboundDecision{}, fmt.Errorf("nope: %w", gotenberg.ErrFiltered)
	}
	p.dialPinned = func(_ context.Context, _ string, _ []netip.Addr, _ string) (net.Conn, error) {
		t.Fatal("dialPinned must not be called when decide returns an error")
		return nil, errors.New("unreachable")
	}
	proxyURL := newProxyForTest(t, p)

	conn, err := net.Dial("tcp", strings.TrimPrefix(proxyURL, "http://"))
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	_, err = fmt.Fprintf(conn, "CONNECT rebind.example:443 HTTP/1.1\r\nHost: rebind.example:443\r\n\r\n")
	if err != nil {
		t.Fatalf("write CONNECT: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("CONNECT status = %d, want 403", resp.StatusCode)
	}
}

// TestPinningProxy_DNSRebind_SingleResolution is the regression test for
// the DNS rebinding window. It simulates a DNS authority that returns a
// public IP on the first lookup and a loopback IP on subsequent lookups.
// The proxy must resolve the host exactly once per request and dial the
// IP validated at that moment, so that a second resolution by any later
// layer cannot pivot the connection to an internal target.
func TestPinningProxy_DNSRebind_SingleResolution(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "public-upstream")
	}))
	t.Cleanup(upstream.Close)
	upstreamURL := mustParseURL(t, upstream.URL)

	var lookupCount atomic.Int32
	stubDecide := func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		n := lookupCount.Add(1)
		if n == 1 {
			// First lookup: returns a public IP, validation passes, the
			// proxy pins it for the dial.
			return gotenberg.OutboundDecision{Pinned: []netip.Addr{netip.MustParseAddr("93.184.216.34")}}, nil
		}
		// Any subsequent lookup for the same host would return a
		// loopback IP. This return value must not influence the dial
		// because the proxy must not call decide again for this request.
		return gotenberg.OutboundDecision{}, fmt.Errorf("rebind lookup: %w", gotenberg.ErrFiltered)
	}

	p := newPinningProxy(nil, nil, false, false)
	p.decide = stubDecide
	p.dialPinned = func(_ context.Context, network string, addrs []netip.Addr, _ string) (net.Conn, error) {
		if len(addrs) != 1 || addrs[0].String() != "93.184.216.34" {
			t.Errorf("dialPinned got addrs %v, want [93.184.216.34]", addrs)
		}
		return net.Dial(network, upstreamURL.Host)
	}
	proxyURL := newProxyForTest(t, p)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(t, proxyURL)),
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://rebind.example/")
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(body) != "public-upstream" {
		t.Fatalf("body = %q, want %q", body, "public-upstream")
	}
	if got := lookupCount.Load(); got != 1 {
		t.Fatalf("decide called %d times, want exactly 1 (rebind protection)", got)
	}
}

func TestPinningProxy_StartTwice(t *testing.T) {
	p := newPinningProxy(nil, nil, false, false)
	err := p.Start(testLogger())
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}
	t.Cleanup(func() { _ = p.Stop(testLogger()) })

	err = p.Start(testLogger())
	if err == nil {
		t.Fatal("second Start: expected error, got nil")
	}
}

func TestPinningProxy_StopIdempotent(t *testing.T) {
	p := newPinningProxy(nil, nil, false, false)
	// Stop on a never-started proxy is a no-op.
	if err := p.Stop(testLogger()); err != nil {
		t.Fatalf("Stop on never-started proxy: %v", err)
	}

	if err := p.Start(testLogger()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Stop(testLogger()); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := p.Stop(testLogger()); err != nil {
		t.Fatalf("second Stop on stopped proxy: %v", err)
	}
}
