package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// Propagate HTTP request context cancellation to processing modules to save resources
// https://github.com/gotenberg/gotenberg/issues/1455
func TestNewContext_Cancellation(t *testing.T) {
	e := echo.New()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	err := writer.Close()
	if err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	// Create a request with a cancellable context.
	reqCtx, cancelReq := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/", body).WithContext(reqCtx)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	logger := slog.New(slog.DiscardHandler)
	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
	timeout := time.Duration(10) * time.Second
	downloadFromCfg := downloadFromConfig{
		disable: true,
	}

	ctx, cancel, err := newContext(c, logger, fs, timeout, 0, downloadFromCfg)
	if err != nil {
		t.Fatalf("expected no error from newContext, got: %v", err)
	}
	defer cancel()

	// Verify initial state: context SHOULD NOT be done yet.
	select {
	case <-ctx.Done():
		t.Fatal("context should not be done immediately")
	default:
	}

	// Simulate Client Disconnect
	cancelReq()

	// Verify Propagation
	select {
	case <-ctx.Done():
		// Success! The context was cancelled.
		if ctx.Err() != context.Canceled {
			t.Errorf("expected context error to be 'context.Canceled', got: %v", ctx.Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected context to be cancelled after request context cancellation, but it timed out")
	}
}

// Concurrent downloadFrom entries must not race on the shared maps
// (ctx.files, ctx.diskToOriginal, ctx.filesByField). Run under -race
// to catch the data race; without -race a sufficient number of entries
// still surfaces "fatal error: concurrent map writes".
func TestNewContext_DownloadFromConcurrentMapWrites(t *testing.T) {
	const downloads = 64

	var ready sync.WaitGroup
	ready.Add(downloads)
	release := make(chan struct{})
	var releaseOnce sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready.Done()
		go func() {
			ready.Wait()
			releaseOnce.Do(func() { close(release) })
		}()
		<-release

		filename := fmt.Sprintf("download-%s.txt", r.URL.Query().Get("i"))
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer server.Close()

	dls := make([]downloadFrom, downloads)
	for i := range dls {
		dls[i] = downloadFrom{
			Url:   fmt.Sprintf("%s/file?i=%d", server.URL, i),
			Field: "embedded",
		}
	}

	payload, err := json.Marshal(dls)
	if err != nil {
		t.Fatalf("marshal downloadFrom payload: %v", err)
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	err = writer.WriteField("downloadFrom", string(payload))
	if err != nil {
		t.Fatalf("write downloadFrom field: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	echoCtx := echo.New().NewContext(req, httptest.NewRecorder())
	logger := slog.New(slog.DiscardHandler)
	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
	downloadFromCfg := downloadFromConfig{
		maxRetry: 0,
	}

	ctx, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromCfg)
	if err != nil {
		t.Fatalf("newContext returned error: %v", err)
	}
	defer cancel()

	if got := len(ctx.files); got != downloads {
		t.Fatalf("downloaded files = %d, want %d", got, downloads)
	}
	if got := len(ctx.diskToOriginal); got != downloads {
		t.Fatalf("diskToOriginal entries = %d, want %d", got, downloads)
	}
	if got := len(ctx.filesByField[EmbedsFormField]); got != downloads {
		t.Fatalf("filesByField[%q] entries = %d, want %d", EmbedsFormField, got, downloads)
	}
}

func TestSanitizeFilename(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		input    string
		expect   string
	}{
		{
			scenario: "plain filename is unchanged",
			input:    "report.pdf",
			expect:   "report.pdf",
		},
		{
			scenario: "POSIX traversal is stripped",
			input:    "../../etc/passwd",
			expect:   "passwd",
		},
		{
			scenario: "Windows traversal with backslashes is stripped",
			input:    `..\..\..\..\Windows\System32\evil.pdf`,
			expect:   "evil.pdf",
		},
		{
			scenario: "mixed separators take the last segment",
			input:    `foo/bar\baz.pdf`,
			expect:   "baz.pdf",
		},
		{
			scenario: "control characters are dropped",
			input:    "evil\x00\x07\x1f\x7f.pdf",
			expect:   "evil.pdf",
		},
		{
			scenario: "NFC normalization collapses decomposed sequences",
			// "e" + combining acute accent -> precomposed "é".
			input:  "café.pdf",
			expect: "café.pdf",
		},
		{
			scenario: "trailing backslash yields empty name",
			input:    `foo\`,
			expect:   "",
		},
		{
			scenario: "empty input yields empty name",
			input:    "",
			expect:   "",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			got := sanitizeFilename(tc.input)
			if got != tc.expect {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}
