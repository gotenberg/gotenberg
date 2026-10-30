package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// writeSelfSignedPair writes a certificate and its key under dir, which
// t.TempDir makes absolute.
func writeSelfSignedPair(t *testing.T, dir string) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	keyDer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}

	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")

	err = os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600)
	if err != nil {
		t.Fatalf("write certificate: %v", err)
	}

	err = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer}), 0o600)
	if err != nil {
		t.Fatalf("write key: %v", err)
	}

	return certFile, keyFile
}

func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

func newTLSTestApi(t *testing.T, certFile, keyFile string) *Api {
	t.Helper()

	return &Api{
		port:         freePort(t),
		bindIp:       "127.0.0.1",
		rootPath:     "/",
		timeout:      10 * time.Second,
		startTimeout: 10 * time.Second,
		tlsCertFile:  certFile,
		tlsKeyFile:   keyFile,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// Absolute paths are how a mounted secret is usually referenced.
func TestApi_Start_TLSWithAbsolutePaths(t *testing.T) {
	certFile, keyFile := writeSelfSignedPair(t, t.TempDir())
	if !filepath.IsAbs(certFile) || !filepath.IsAbs(keyFile) {
		t.Fatalf("expected absolute paths, got %q and %q", certFile, keyFile)
	}

	a := newTLSTestApi(t, certFile, keyFile)

	err := a.Start()
	if err != nil {
		t.Fatalf("expected no error from Start, got %v", err)
	}
	t.Cleanup(func() {
		_ = a.Stop(context.Background())
	})

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}

	url := fmt.Sprintf("https://127.0.0.1:%d/version", a.port)

	var lastErr error
	for range 50 {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}
			return
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("HTTPS server never answered on %s: %v", url, lastErr)
}

// An unreadable certificate must fail the startup, not leave a running
// process with no listener.
func TestApi_Start_TLSWithMissingFiles(t *testing.T) {
	dir := t.TempDir()
	a := newTLSTestApi(t, filepath.Join(dir, "missing.crt"), filepath.Join(dir, "missing.key"))

	err := a.Start()
	t.Cleanup(func() {
		_ = a.Stop(context.Background())
	})

	if err == nil {
		t.Fatal("expected an error from Start, got nil")
	}
}

// A module that never becomes ready must fail the startup once
// --api-start-timeout elapses.
func TestApi_Start_HonorsStartTimeout(t *testing.T) {
	a := newTLSTestApi(t, "", "")
	a.startTimeout = 100 * time.Millisecond
	a.readyFn = append(a.readyFn, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	start := time.Now()
	err := a.Start()
	t.Cleanup(func() {
		_ = a.Stop(context.Background())
	})

	if err == nil {
		t.Fatal("expected an error from Start, got nil")
	}
	if !strings.Contains(err.Error(), "API_START_TIMEOUT") {
		t.Errorf("expected the error to name API_START_TIMEOUT, got %q", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Start returned after %s, want shortly after the start timeout", elapsed)
	}
}

// Start returns once the server listens, so the summary that follows reports
// a server that answers.
func TestApi_Start_ReturnsOnceListening(t *testing.T) {
	a := newTLSTestApi(t, "", "")

	err := a.Start()
	if err != nil {
		t.Fatalf("expected no error from Start, got %v", err)
	}
	t.Cleanup(func() {
		_ = a.Stop(context.Background())
	})

	resp, err := (&http.Client{Timeout: time.Second}).Get(fmt.Sprintf("http://127.0.0.1:%d/version", a.port))
	if err != nil {
		t.Fatalf("expected the server to answer right after Start, got %v", err)
	}
	_ = resp.Body.Close()

	want := fmt.Sprintf("http://127.0.0.1:%d/", a.port)
	if got := a.Report().Fields[0].Value; got != want {
		t.Errorf("reported address = %v, want %q", got, want)
	}
}

// A port in use must fail the startup, not leave a running process with no
// listener.
func TestApi_Start_PortInUse(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	a := newTLSTestApi(t, "", "")
	a.port = l.Addr().(*net.TCPAddr).Port

	err = a.Start()
	t.Cleanup(func() {
		_ = a.Stop(context.Background())
	})

	if err == nil {
		t.Fatal("expected an error from Start, got nil")
	}
}

type asyncCounterStub struct{ count atomic.Int64 }

func (s *asyncCounterStub) AsyncCount() int64 { return s.count.Load() }

// Stop must refuse new requests before it waits for the asynchronous ones, so
// that the wait cannot be stretched by new work.
func TestApi_Stop_RefusesRequestsBeforeWaitingForAsyncOnes(t *testing.T) {
	a := newTLSTestApi(t, "", "")
	counter := new(asyncCounterStub)
	counter.count.Store(1)
	a.asyncCounters = []AsynchronousCounter{counter}

	err := a.Start()
	if err != nil {
		t.Fatalf("expected no error from Start, got %v", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/version", a.port)
	client := &http.Client{Timeout: time.Second}

	var lastErr error
	for range 50 {
		var resp *http.Response
		resp, lastErr = client.Get(url)
		if lastErr == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("server never answered on %s: %v", url, lastErr)
	}

	stopped := make(chan error, 1)
	go func() {
		stopped <- a.Stop(context.Background())
	}()

	refused := false
	for range 50 {
		resp, err := client.Get(url)
		if err != nil {
			refused = true
			break
		}
		_ = resp.Body.Close()
		time.Sleep(20 * time.Millisecond)
	}
	if !refused {
		t.Fatal("the server still accepted requests while waiting for asynchronous ones")
	}

	select {
	case err = <-stopped:
		t.Fatalf("Stop returned with an asynchronous request still running: %v", err)
	default:
	}

	counter.count.Store(0)

	select {
	case err = <-stopped:
		if err != nil {
			t.Fatalf("expected no error from Stop, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return once the asynchronous requests completed")
	}
}

func TestApi_Stop_HonorsContextWhileWaitingForAsyncOnes(t *testing.T) {
	a := newTLSTestApi(t, "", "")
	counter := new(asyncCounterStub)
	counter.count.Store(1)
	a.asyncCounters = []AsynchronousCounter{counter}

	err := a.Start()
	if err != nil {
		t.Fatalf("expected no error from Start, got %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = a.Stop(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected a deadline error, got %v", err)
	}
}
