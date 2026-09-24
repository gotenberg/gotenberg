package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	"github.com/labstack/echo/v4"
)

// TestRequestCanceled pins the client-abort discriminator: only a
// context.Canceled that stems from the request context counts, so a server
// timeout or an unrelated cancellation still surfaces as an internal failure.
// See https://github.com/gotenberg/gotenberg/issues/1627.
func TestRequestCanceled(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	timedOut, cancelTimeout := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelTimeout()

	for _, tc := range []struct {
		name   string
		reqCtx context.Context
		err    error
		want   bool
	}{
		{"client abort", canceled, context.Canceled, true},
		{"wrapped client abort", canceled, fmt.Errorf("convert to PDF: %w", context.Canceled), true},
		{"canceled error but live request", context.Background(), context.Canceled, false},
		{"canceled request but unrelated error", canceled, errors.New("boom"), false},
		{"server timeout is not a client abort", timedOut, context.DeadlineExceeded, false},
		{"no error", canceled, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(tc.reqCtx)
			c := echo.New().NewContext(req, httptest.NewRecorder())
			if got := requestCanceled(c, tc.err); got != tc.want {
				t.Fatalf("requestCanceled = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestHttpErrorHandler_ClientClosedRequest ensures a client abort is recorded
// as 499 rather than 500, and that a genuine failure keeps its status.
func TestHttpErrorHandler_ClientClosedRequest(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	for _, tc := range []struct {
		name       string
		reqCtx     context.Context
		err        error
		wantStatus int
	}{
		{"client abort", canceled, fmt.Errorf("convert to PDF: %w", context.Canceled), statusClientClosedRequest},
		{"internal failure", context.Background(), errors.New("boom"), http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(tc.reqCtx)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set("logger", slog.New(slog.DiscardHandler))

			httpErrorHandler()(tc.err, c)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

// TestOutputFilenameMiddleware pins the sanitizing of the
// "Gotenberg-Output-Filename" header. The value reaches archive entry names and
// a Content-Disposition header, so a path separator must never survive it.
// See https://github.com/gotenberg/gotenberg/issues/1227 and
// GHSA-hwc4-gmrw-5222.
func TestOutputFilenameMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		want   string
	}{
		{"no header", "", ""},
		{"plain filename", "foo", "foo"},
		{"POSIX path", "/tmp/foo", "foo"},
		{"POSIX traversal", "../../../etc/passwd", "passwd"},
		{"Windows traversal", `..\..\..\..\Windows\System32\evil`, "evil"},
		{"rooted Windows path", `C:\Windows\Temp\evil`, "evil"},
		{"mixed separators", `a/b\c`, "c"},
		{"trailing separator", "/tmp/", ""},
		{"bare dot dot", "..", ".."},
		{"control characters", "fo\x01o\x7f", "foo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := outputFilenameMiddleware()(func(c echo.Context) error { return nil })

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.header != "" {
				req.Header.Set("Gotenberg-Output-Filename", tc.header)
			}
			c := echo.New().NewContext(req, httptest.NewRecorder())

			err := handler(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, ok := c.Get("outputFilename").(string)
			if !ok {
				t.Fatal("outputFilename is not set as a string")
			}
			if got != tc.want {
				t.Errorf("outputFilename = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHardTimeoutMiddleware_MissingLoggerReturnsErrorInsteadOfPanicking(t *testing.T) {
	mw := hardTimeoutMiddleware(100 * time.Millisecond)
	handler := mw(func(c echo.Context) error { return nil })

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// c has no "logger" key, mimicking a pooled context whose store was
	// recycled under a concurrently running webhook goroutine. The
	// middleware must surface an error instead of panicking on the
	// unchecked type assertion the pre-fix code relied on.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("hardTimeoutMiddleware panicked: %v", r)
		}
	}()

	err := handler(c)
	if err == nil {
		t.Fatal("expected an error for missing logger, got nil")
	}
	if !strings.Contains(err.Error(), "logger") {
		t.Fatalf("error = %q, want a message mentioning logger", err)
	}
}

func TestOidcAuthMiddleware(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	const (
		keyID    = "test-key"
		audience = "gotenberg"
	)

	oidcServer := &oidctest.Server{
		PublicKeys: []oidctest.PublicKey{
			{PublicKey: privateKey.Public(), KeyID: keyID, Algorithm: oidc.RS256},
		},
	}
	srv := httptest.NewServer(oidcServer)
	defer srv.Close()
	oidcServer.SetIssuer(srv.URL)

	// Building through the module's own helper exercises the discovery path too.
	a := &Api{oidcIssuer: srv.URL, oidcAudience: audience}
	verifier, err := a.buildOidcVerifier()
	if err != nil {
		t.Fatalf("build verifier: %v", err)
	}

	claims := func(issuer, aud string, expiresIn time.Duration) string {
		now := time.Now()
		return fmt.Sprintf(`{"iss":%q,"aud":%q,"sub":"user","exp":%d,"iat":%d}`,
			issuer, aud, now.Add(expiresIn).Unix(), now.Unix())
	}
	sign := func(claims string) string {
		return oidctest.SignIDToken(privateKey, keyID, oidc.RS256, claims)
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}

	for _, tc := range []struct {
		scenario   string
		authHeader string
		wantStatus int
	}{
		{"valid token", "Bearer " + sign(claims(srv.URL, audience, time.Hour)), http.StatusOK},
		{"missing header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic Zm9vOmJhcg==", http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
		{"malformed token", "Bearer not-a-jwt", http.StatusUnauthorized},
		{"wrong issuer", "Bearer " + sign(claims("https://evil.example/", audience, time.Hour)), http.StatusUnauthorized},
		{"wrong audience", "Bearer " + sign(claims(srv.URL, "someone-else", time.Hour)), http.StatusUnauthorized},
		{"expired token", "Bearer " + sign(claims(srv.URL, audience, -time.Hour)), http.StatusUnauthorized},
		{"unknown signing key", "Bearer " + oidctest.SignIDToken(otherKey, "unknown", oidc.RS256, claims(srv.URL, audience, time.Hour)), http.StatusUnauthorized},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			c := echo.New().NewContext(req, httptest.NewRecorder())

			handler := oidcAuthMiddleware(verifier)(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			err := handler(c)

			if tc.wantStatus == http.StatusOK {
				if err != nil {
					t.Fatalf("expected the request to pass, got error: %v", err)
				}
				return
			}

			var httpErr *echo.HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("expected an *echo.HTTPError, got %T (%v)", err, err)
			}
			if httpErr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", httpErr.Code, tc.wantStatus)
			}
		})
	}
}
