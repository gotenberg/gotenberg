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
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
