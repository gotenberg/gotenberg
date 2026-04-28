package api

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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
)

func compileRegexes(t *testing.T, patterns ...string) []*regexp2.Regexp {
	t.Helper()
	out := make([]*regexp2.Regexp, 0, len(patterns))
	for _, p := range patterns {
		r, err := regexp2.Compile(p, 0)
		if err != nil {
			t.Fatalf("compile %q: %v", p, err)
		}
		out = append(out, r)
	}
	return out
}

func startProxy(t *testing.T, opts outboundProxyOptions) *libreOfficeProxy {
	t.Helper()
	p, err := newLibreOfficeProxy(slog.New(slog.DiscardHandler), opts)
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}
	p.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Stop(ctx)
	})
	return p
}

func TestLibreOfficeProxy_HttpForwardAllowed(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	}))
	defer origin.Close()

	p := startProxy(t, outboundProxyOptions{})

	proxyURL, _ := url.Parse("http://" + p.Addr())
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   5 * time.Second,
	}

	resp, err := client.Get(origin.URL + "/foo")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusTeapot)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Fatalf("body: got %q, want %q", body, "hello")
	}
}

func TestLibreOfficeProxy_HttpForwardDenyListRejects(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("origin must not be reached")
	}))
	defer origin.Close()

	p := startProxy(t, outboundProxyOptions{
		denyList: compileRegexes(t, `.*`),
	})

	proxyURL, _ := url.Parse("http://" + p.Addr())
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   5 * time.Second,
	}

	resp, err := client.Get(origin.URL + "/foo")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestLibreOfficeProxy_HttpForwardDenyPrivateIPsRejects(t *testing.T) {
	// httptest binds on 127.0.0.1 (a private IP), so denyPrivateIPs
	// must reject the forward.
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("origin must not be reached")
	}))
	defer origin.Close()

	p := startProxy(t, outboundProxyOptions{denyPrivateIPs: true})

	proxyURL, _ := url.Parse("http://" + p.Addr())
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   5 * time.Second,
	}

	resp, err := client.Get(origin.URL + "/foo")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestLibreOfficeProxy_ConnectTunnelHappyPath(t *testing.T) {
	// Bring up a tiny TCP echo server.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(conn, conn)
	}()

	p := startProxy(t, outboundProxyOptions{})

	conn, err := net.DialTimeout("tcp", p.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	target := listener.Addr().String()
	_, err = fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	if err != nil {
		t.Fatalf("write CONNECT: %v", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read CONNECT response: %v", err)
	}
	if !strings.Contains(statusLine, "200") {
		t.Fatalf("CONNECT status: got %q, want 200", statusLine)
	}
	// Drain remaining headers.
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			t.Fatalf("read CONNECT headers: %v", readErr)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	// Tunnel established. Round-trip a payload through the echo server.
	want := "ping"
	_, err = conn.Write([]byte(want))
	if err != nil {
		t.Fatalf("write payload: %v", err)
	}

	got := make([]byte, len(want))
	_, err = io.ReadFull(reader, got)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(got) != want {
		t.Fatalf("echo: got %q, want %q", got, want)
	}
}

func TestLibreOfficeProxy_ConnectDenyListRejects(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	p := startProxy(t, outboundProxyOptions{denyList: compileRegexes(t, `.*`)})

	conn, err := net.DialTimeout("tcp", p.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	target := listener.Addr().String()
	_, err = fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	if err != nil {
		t.Fatalf("write CONNECT: %v", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if !strings.Contains(statusLine, "403") {
		t.Fatalf("CONNECT status: got %q, want 403", statusLine)
	}
}

func TestLibreOfficeProxy_ConnectDenyPrivateIPsRejects(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	p := startProxy(t, outboundProxyOptions{denyPrivateIPs: true})

	conn, err := net.DialTimeout("tcp", p.Addr(), 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	// 127.0.0.1 is a private IP under denyPrivateIPs.
	target := listener.Addr().String()
	_, err = fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
	if err != nil {
		t.Fatalf("write CONNECT: %v", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if !strings.Contains(statusLine, "403") {
		t.Fatalf("CONNECT status: got %q, want 403", statusLine)
	}
}

func TestLibreOfficeProxy_StopIsIdempotent(t *testing.T) {
	p, err := newLibreOfficeProxy(slog.New(slog.DiscardHandler), outboundProxyOptions{})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}
	p.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := p.Stop(ctx); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}

func TestWriteSofficeProxyConfig(t *testing.T) {
	dir := t.TempDir()

	if err := writeSofficeProxyConfig(dir, "127.0.0.1:9876"); err != nil {
		t.Fatalf("writeSofficeProxyConfig: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(dir, "user", "registrymodifications.xcu"))
	if err != nil {
		t.Fatalf("read xcu: %v", err)
	}

	for _, want := range []string{
		`ooInetProxyType`, `<value>1</value>`,
		`ooInetHTTPProxyName`, `<value>127.0.0.1</value>`,
		`ooInetHTTPProxyPort`, `<value>9876</value>`,
		`ooInetHTTPSProxyName`, `ooInetHTTPSProxyPort`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("xcu missing %q\nfull body:\n%s", want, body)
		}
	}
}

func TestWriteSofficeProxyConfig_InvalidAddr(t *testing.T) {
	err := writeSofficeProxyConfig(t.TempDir(), "not-a-host-port")
	if err == nil {
		t.Fatal("expected error for malformed proxy address")
	}
	if !errors.Is(err, errors.Unwrap(err)) {
		// Only checking that an error was returned; underlying error type is
		// implementation detail.
		_ = err
	}
}

func TestSofficeProxyEnv_OverridesExisting(t *testing.T) {
	in := []string{
		"PATH=/usr/bin",
		"http_proxy=http://attacker:1",
		"HTTPS_PROXY=http://attacker:1",
		"NO_PROXY=internal",
		"USER=gotenberg",
	}
	out := sofficeProxyEnv(in, "127.0.0.1:9876")

	want := map[string]string{
		"http_proxy":  "http://127.0.0.1:9876",
		"https_proxy": "http://127.0.0.1:9876",
		"HTTP_PROXY":  "http://127.0.0.1:9876",
		"HTTPS_PROXY": "http://127.0.0.1:9876",
		"no_proxy":    "",
		"NO_PROXY":    "",
	}

	got := map[string]string{}
	for _, kv := range out {
		parts := strings.SplitN(kv, "=", 2)
		got[parts[0]] = parts[1]
	}

	for key, value := range want {
		if got[key] != value {
			t.Errorf("env[%s]: got %q, want %q", key, got[key], value)
		}
	}

	// Pre-existing unrelated keys must survive.
	if got["PATH"] != "/usr/bin" {
		t.Errorf("env[PATH]: got %q, want /usr/bin", got["PATH"])
	}
	if got["USER"] != "gotenberg" {
		t.Errorf("env[USER]: got %q, want gotenberg", got["USER"])
	}

	// Old proxy values must be gone, not duplicated. Count exact-case keys.
	counts := map[string]int{}
	for _, kv := range out {
		key := strings.SplitN(kv, "=", 2)[0]
		counts[key]++
	}
	for _, key := range []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY", "no_proxy", "NO_PROXY"} {
		if counts[key] != 1 {
			t.Errorf("env[%s] count: got %d, want 1", key, counts[key])
		}
	}
}
