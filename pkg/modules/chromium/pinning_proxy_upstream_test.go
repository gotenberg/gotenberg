package chromium

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dlclark/regexp2"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// TestPinningProxy_Forward_ThroughUpstreamProxy verifies that when the
// operator opts into proxy-environment honoring, a plain HTTP request is
// forwarded through the upstream (corporate) proxy with the credentials
// Chromium cannot supply. See https://github.com/gotenberg/gotenberg/issues/1592.
func TestPinningProxy_Forward_ThroughUpstreamProxy(t *testing.T) {
	var gotAuth atomic.Value
	gotAuth.Store("")

	// Stand-in for the corporate proxy: an HTTP server that receives the
	// forwarded request and records the injected Proxy-Authorization.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth.Store(r.Header.Get("Proxy-Authorization"))
		_, _ = fmt.Fprint(w, "via-corporate-proxy")
	}))
	t.Cleanup(upstream.Close)

	upstreamURL := mustParseURL(t, upstream.URL)
	upstreamURL.User = url.UserPassword("bob", "pw")

	p := newPinningProxy(nil, nil, false, false, true)
	// Force every destination through our stub upstream proxy.
	p.upstreamProxy = func(_ *url.URL) (*url.URL, error) { return upstreamURL, nil }
	p.decide = func(_ context.Context, _ string, _, _ []*regexp2.Regexp, _ time.Time) (gotenberg.OutboundDecision, error) {
		return gotenberg.OutboundDecision{Pinned: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}, nil
	}
	p.dialPinned = func(_ context.Context, _ string, _ []netip.Addr, _ string) (net.Conn, error) {
		t.Fatal("dialPinned must not be called when routing through an upstream proxy")
		return nil, nil
	}
	proxyURL := newProxyForTest(t, p)

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(mustParseURL(t, proxyURL))},
		Timeout:   5 * time.Second,
	}

	resp, err := client.Get("http://example.com/")
	if err != nil {
		t.Fatalf("GET via proxy: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "via-corporate-proxy" {
		t.Fatalf("body = %q, want via-corporate-proxy", body)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("bob:pw"))
	if got := gotAuth.Load().(string); got != wantAuth {
		t.Fatalf("upstream proxy saw Proxy-Authorization %q, want %q", got, wantAuth)
	}
}
