package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
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

func TestNewContext_RemovesMultipartTemporaryFiles(t *testing.T) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "input.odt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	_, err = part.Write(bytes.Repeat([]byte("x"), 1024))
	if err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	err = req.ParseMultipartForm(1)
	if err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}
	defer func() {
		_ = req.MultipartForm.RemoveAll()
	}()

	upload, err := req.MultipartForm.File["files"][0].Open()
	if err != nil {
		t.Fatalf("open disk-backed multipart file: %v", err)
	}
	temporaryFile, ok := upload.(*os.File)
	if !ok {
		_ = upload.Close()
		t.Fatal("multipart upload is not disk-backed")
	}
	temporaryPath := temporaryFile.Name()
	err = temporaryFile.Close()
	if err != nil {
		t.Fatalf("close disk-backed multipart file: %v", err)
	}

	echoCtx := echo.New().NewContext(req, httptest.NewRecorder())
	logger := slog.New(slog.DiscardHandler)
	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
	downloadFromCfg := downloadFromConfig{disable: true}

	_, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromCfg)
	if err != nil {
		t.Fatalf("newContext returned error: %v", err)
	}
	defer cancel()

	_, err = os.Stat(temporaryPath)
	if !os.IsNotExist(err) {
		t.Fatalf("multipart temporary file still exists: %s", temporaryPath)
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

// An oversized downloadFrom array must be rejected at the trust boundary with
// a 400, before any download goroutine is spawned.
// https://github.com/gotenberg/gotenberg/security/advisories/GHSA-6vqw-2jgm-4x88
func TestNewContext_DownloadFromMaxEntries(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Disposition", `attachment; filename="download.txt"`)
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer server.Close()

	dls := make([]downloadFrom, 3)
	for i := range dls {
		dls[i] = downloadFrom{Url: fmt.Sprintf("%s/file?i=%d", server.URL, i)}
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
	downloadFromCfg := downloadFromConfig{maxEntries: 2}

	_, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromCfg)
	if cancel != nil {
		defer cancel()
	}
	if err == nil {
		t.Fatal("newContext returned no error, want a 400 for too many entries")
	}

	var httpErr HttpError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error %v is not an HttpError", err)
	}
	if status, _ := httpErr.HttpError(); status != http.StatusBadRequest {
		t.Fatalf("HTTP status = %d, want %d", status, http.StatusBadRequest)
	}
	if got := hits.Load(); got != 0 {
		t.Fatalf("server hits = %d, want 0 (rejected before any download)", got)
	}
}

// The number of in-flight downloadFrom fetches must never exceed the
// configured concurrency limit, regardless of array length.
// https://github.com/gotenberg/gotenberg/security/advisories/GHSA-6vqw-2jgm-4x88
func TestNewContext_DownloadFromMaxConcurrency(t *testing.T) {
	const (
		downloads      = 8
		maxConcurrency = 2
	)

	var current, peak atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlight := current.Add(1)
		for {
			observed := peak.Load()
			if inFlight <= observed || peak.CompareAndSwap(observed, inFlight) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		current.Add(-1)

		filename := fmt.Sprintf("download-%s.txt", r.URL.Query().Get("i"))
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer server.Close()

	dls := make([]downloadFrom, downloads)
	for i := range dls {
		dls[i] = downloadFrom{Url: fmt.Sprintf("%s/file?i=%d", server.URL, i)}
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
	downloadFromCfg := downloadFromConfig{maxConcurrency: maxConcurrency}

	ctx, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromCfg)
	if err != nil {
		t.Fatalf("newContext returned error: %v", err)
	}
	defer cancel()

	if got := len(ctx.files); got != downloads {
		t.Fatalf("downloaded files = %d, want %d", got, downloads)
	}
	if got := peak.Load(); got > maxConcurrency {
		t.Fatalf("peak concurrency = %d, want <= %d", got, maxConcurrency)
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

func TestContext_FileCount(t *testing.T) {
	ctx := &Context{}
	if got := ctx.FileCount(); got != 0 {
		t.Errorf("expected 0 files, got %d", got)
	}

	ctx.files = map[string]string{
		"index.html":  "/work/index.html",
		"header.html": "/work/header.html",
		"styles.css":  "/work/styles.css",
	}
	if got := ctx.FileCount(); got != 3 {
		t.Errorf("expected 3 files, got %d", got)
	}
}
