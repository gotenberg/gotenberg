package gotenberg

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

// connectCapture records the CONNECT request a proxy stub received.
type connectCapture struct {
	mu     sync.Mutex
	method string
	host   string
	auth   string
}

func (c *connectCapture) set(method, host, auth string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.method, c.host, c.auth = method, host, auth
}

func (c *connectCapture) get() (string, string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.method, c.host, c.auth
}

// startConnectProxyStub starts a raw TCP server that behaves like an HTTP
// CONNECT proxy: it reads the CONNECT request, records it, replies 200 with a
// greeting appended to the same write (to exercise buffered-byte handling),
// then echoes tunnel bytes back to the caller.
func startConnectProxyStub(t *testing.T, capture *connectCapture) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		br := bufio.NewReader(conn)
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}
		capture.set(req.Method, req.Host, req.Header.Get("Proxy-Authorization"))

		// The greeting rides along with the response so the client's CONNECT
		// response parser buffers it; bufferedConn must not drop it.
		_, _ = conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\nTUNNEL-HELLO"))
		_, _ = io.Copy(conn, br)
	}()

	return l.Addr().String()
}

func TestDialThroughProxy(t *testing.T) {
	capture := &connectCapture{}
	addr := startConnectProxyStub(t, capture)

	proxyURL := &url.URL{Scheme: "http", Host: addr, User: url.UserPassword("alice", "s3cr3t")}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := DialThroughProxy(ctx, proxyURL, "example.com:443", func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	})
	if err != nil {
		t.Fatalf("DialThroughProxy: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// The greeting buffered while reading the CONNECT response must survive.
	greeting := make([]byte, len("TUNNEL-HELLO"))
	_, err = io.ReadFull(conn, greeting)
	if err != nil {
		t.Fatalf("read greeting: %v", err)
	}
	if string(greeting) != "TUNNEL-HELLO" {
		t.Fatalf("greeting = %q, want TUNNEL-HELLO", greeting)
	}

	// The tunnel must round-trip bytes.
	_, err = conn.Write([]byte("ping"))
	if err != nil {
		t.Fatalf("write to tunnel: %v", err)
	}
	echo := make([]byte, 4)
	_, err = io.ReadFull(conn, echo)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(echo) != "ping" {
		t.Fatalf("echo = %q, want ping", echo)
	}

	method, host, auth := capture.get()
	if method != http.MethodConnect {
		t.Fatalf("proxy saw method %q, want CONNECT", method)
	}
	if host != "example.com:443" {
		t.Fatalf("proxy saw target %q, want example.com:443", host)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:s3cr3t"))
	if auth != wantAuth {
		t.Fatalf("proxy saw Proxy-Authorization %q, want %q", auth, wantAuth)
	}
}

func TestDialThroughProxy_RefusedStatus(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		br := bufio.NewReader(conn)
		_, _ = http.ReadRequest(br)
		_, _ = conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n\r\n"))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = DialThroughProxy(ctx, &url.URL{Scheme: "http", Host: l.Addr().String()}, "example.com:443", func(ctx context.Context, network, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	})
	if err == nil {
		t.Fatal("expected an error when the proxy refuses CONNECT, got nil")
	}
}
