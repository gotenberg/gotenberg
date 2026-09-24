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
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
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

// A hostile origin must not choose how long Gotenberg waits.
// [retryablehttp.DefaultBackoff] returns a Retry-After header verbatim for 429
// and 503, and the wait between attempts is a select on the request context.
// Building the request without a context therefore pinned the goroutine, its
// connection, and its working directory for the attacker's chosen duration,
// well past --api-timeout (env API_TIMEOUT).
func TestNewContext_DownloadFromHostileRetryAfterIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	payload, err := json.Marshal([]downloadFrom{{Url: server.URL + "/file"}})
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

	const timeout = 500 * time.Millisecond

	start := time.Now()
	_, cancel, err := newContext(echoCtx, logger, fs, timeout, 0, downloadFromConfig{maxRetry: 2})
	elapsed := time.Since(start)
	if cancel != nil {
		defer cancel()
	}

	if err == nil {
		t.Fatal("expected newContext to fail against an origin that only answers 429")
	}
	// Generous: the deadline is 500ms and Retry-After asks for an hour. Any
	// value in seconds means the remote is still in control.
	if elapsed > 10*time.Second {
		t.Fatalf("newContext took %s with Retry-After 3600; --api-timeout must bound it", elapsed)
	}
}

// An entry that starts after the deadline has passed must fail closed. It used
// to derive a negative client timeout, which [http.Client] reads as no
// deadline at all, leaving the download unbounded.
func TestNewContext_DownloadFromExpiredBudgetFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	// Two entries, serialized by the concurrency limit, so the second one
	// starts once the first has burned the whole budget.
	payload, err := json.Marshal([]downloadFrom{
		{Url: server.URL + "/first"},
		{Url: server.URL + "/second"},
	})
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

	done := make(chan error, 1)
	go func() {
		_, cancel, err := newContext(echoCtx, logger, fs, 400*time.Millisecond, 0, downloadFromConfig{
			maxRetry:       0,
			maxConcurrency: 1,
		})
		if cancel != nil {
			cancel()
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected newContext to fail against a stalling origin")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("newContext never returned: an entry starting past the deadline built an unbounded client")
	}
}

func TestDecodeDownloadFrom(t *testing.T) {
	for _, tc := range []struct {
		scenario   string
		raw        string
		maxEntries int
		expectErr  error
		expectLen  int
	}{
		{"empty array", `[]`, 10, nil, 0},
		{"under the limit", `[{"url":"http://a"},{"url":"http://b"}]`, 10, nil, 2},
		{"exactly the limit", `[{"url":"http://a"},{"url":"http://b"}]`, 2, nil, 2},
		{"over the limit", `[{"url":"http://a"},{"url":"http://b"}]`, 1, errTooManyDownloadFromEntries, 0},
		{"no limit", `[{"url":"http://a"},{"url":"http://b"}]`, 0, nil, 2},
		{"not an array", `{"url":"http://a"}`, 10, nil, 0},
		{"malformed", `[{"url":`, 10, nil, 0},
		{"not json", `nope`, 10, nil, 0},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			dls, err := decodeDownloadFrom(tc.raw, tc.maxEntries)

			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("error = %v, want %v", err, tc.expectErr)
				}
				return
			}
			if tc.scenario == "not an array" || tc.scenario == "malformed" || tc.scenario == "not json" {
				if err == nil {
					t.Fatalf("expected an error for %q", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(dls) != tc.expectLen {
				t.Fatalf("decoded %d entries, want %d", len(dls), tc.expectLen)
			}
		})
	}
}

// A compact array costs three bytes per entry on the wire and expands by
// roughly seventy times once unmarshalled. Decoding must stop at the limit
// rather than materialize the whole array and count afterwards.
func TestDecodeDownloadFrom_StopsBeforeMaterializingTheArray(t *testing.T) {
	const entries = 2_000_000

	raw := "[" + strings.Repeat("{},", entries) + "{}]"

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	_, err := decodeDownloadFrom(raw, 1000)

	runtime.ReadMemStats(&after)

	if !errors.Is(err, errTooManyDownloadFromEntries) {
		t.Fatalf("error = %v, want errTooManyDownloadFromEntries", err)
	}

	// json.Unmarshal on the same input allocates hundreds of MiB. Bounded
	// decoding should stay in the low single-digit MiB, so this threshold is
	// deliberately loose and still fails loudly on a regression.
	allocated := after.TotalAlloc - before.TotalAlloc
	if allocated > 32<<20 {
		t.Fatalf("decoding allocated %d MiB for a %d-entry array, want the limit to bound it", allocated>>20, entries)
	}
	t.Logf("allocated %d KiB decoding a %d-entry array with a limit of 1000", allocated>>10, entries)
}

// An asynchronous conversion outlives the [echo.Context]. Echo returns that
// context to a sync.Pool as soon as the handler returns, and
// outputFilenameMiddleware runs in srv.Pre on every request, including
// /health, so a later request overwrites the store. Reading the output
// filename from it after the fact returned another caller's value.
func TestContext_OutputFilename_SurvivesEchoContextRecycling(t *testing.T) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	err := writer.Close()
	if err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	echoCtx := echo.New().NewContext(req, httptest.NewRecorder())
	// What outputFilenameMiddleware does for this request.
	echoCtx.Set("outputFilename", "victim")

	logger := slog.New(slog.DiscardHandler)
	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))

	ctx, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromConfig{disable: true})
	if err != nil {
		t.Fatalf("newContext returned error: %v", err)
	}
	defer cancel()

	// Echo recycles the context and another request claims the store.
	echoCtx.Set("outputFilename", "attacker-controlled")

	if got := ctx.OutputFilename("/tmp/out.pdf"); got != "victim.pdf" {
		t.Fatalf("OutputFilename = %q, want %q", got, "victim.pdf")
	}
}

// A recycled context has a nil store, so the previous unguarded type assertion
// could panic. The snapshot must tolerate an absent value.
func TestContext_OutputFilename_NoHeader(t *testing.T) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	err := writer.Close()
	if err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// No Set call at all: the store holds nothing for "outputFilename".
	echoCtx := echo.New().NewContext(req, httptest.NewRecorder())

	logger := slog.New(slog.DiscardHandler)
	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))

	ctx, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromConfig{disable: true})
	if err != nil {
		t.Fatalf("newContext returned error: %v", err)
	}
	defer cancel()

	if got := ctx.OutputFilename("/tmp/out.pdf"); got != "out.pdf" {
		t.Fatalf("OutputFilename = %q, want the original filename %q", got, "out.pdf")
	}
}

func TestSafeExt(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		filename string
		want     string
	}{
		{"ordinary extension", "report.pdf", ".pdf"},
		{"no extension", "report", ""},
		{"at the limit", "a." + strings.Repeat("x", maxDiskExtLength-1), "." + strings.Repeat("x", maxDiskExtLength-1)},
		{"over the limit is dropped", "a." + strings.Repeat("x", 300), ""},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			got := safeExt(tc.filename)
			if got != tc.want {
				t.Fatalf("safeExt(%q) = %q, want %q", tc.filename, got, tc.want)
			}
			// A UUID stem is 36 characters. The whole disk name must stay
			// under NAME_MAX.
			if len(got)+36 > 255 {
				t.Fatalf("disk name would be %d characters, over NAME_MAX", len(got)+36)
			}
		})
	}
}

// An upload whose extension exceeds NAME_MAX used to fail os.Create and return
// a bare 500. The extension is bounded, and the original name survives in
// diskToOriginal.
func TestNewContext_LongExtensionIsAccepted(t *testing.T) {
	filename := "invoice." + strings.Repeat("x", 300)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	_, err = part.Write([]byte("%PDF-1.4"))
	if err != nil {
		t.Fatalf("write multipart file: %v", err)
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

	ctx, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromConfig{disable: true})
	if cancel != nil {
		defer cancel()
	}
	if err != nil {
		t.Fatalf("newContext returned error for a long extension: %v", err)
	}
	if got := len(ctx.files); got != 1 {
		t.Fatalf("files = %d, want 1", got)
	}
}

// A filename that cannot become a symlink (too long, "..", "/") must not fail
// the request. The symlink loop is best-effort, but its error escaped through
// the shared err variable, and ctx.files iterates randomly, so byte-identical
// requests gave different HTTP outcomes.
func TestNewContext_UnsymlinkableFilenameStillSucceeds(t *testing.T) {
	for _, filename := range []string{
		strings.Repeat("a", 300) + ".txt",
		"..",
		"/",
	} {
		t.Run(filename[:min(len(filename), 12)], func(t *testing.T) {
			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("files", filename)
			if err != nil {
				t.Fatalf("create multipart file: %v", err)
			}
			_, err = part.Write([]byte("%PDF-1.4"))
			if err != nil {
				t.Fatalf("write multipart file: %v", err)
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

			_, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromConfig{disable: true})
			if cancel != nil {
				defer cancel()
			}
			if err != nil {
				t.Fatalf("newContext failed on a best-effort symlink for %q: %v", filename, err)
			}
		})
	}
}

// Two uploads sharing a filename must both reach the conversion. The second
// used to overwrite the first in ctx.files, so one file was silently dropped
// while both stayed on disk and counted against the body limit.
func TestNewContext_DuplicateFilenamesAreBothKept(t *testing.T) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for _, content := range []string{"FIRST", "SECOND"} {
		part, err := writer.CreateFormFile("files", "doc.pdf")
		if err != nil {
			t.Fatalf("create multipart file: %v", err)
		}
		_, err = part.Write([]byte(content))
		if err != nil {
			t.Fatalf("write multipart file: %v", err)
		}
	}
	err := writer.Close()
	if err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/forms/pdfengines/merge", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	echoCtx := echo.New().NewContext(req, httptest.NewRecorder())
	logger := slog.New(slog.DiscardHandler)
	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))

	ctx, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromConfig{disable: true})
	if err != nil {
		t.Fatalf("newContext returned error: %v", err)
	}
	defer cancel()

	if got := len(ctx.files); got != 2 {
		t.Fatalf("ctx.files = %d entries, want 2: a duplicate filename dropped a file", got)
	}
	if got := len(ctx.filesByField["files"]); got != 2 {
		t.Fatalf("filesByField[files] = %d entries, want 2", got)
	}

	// The two maps must agree, and both files must be distinct on disk.
	seen := make(map[string]struct{})
	for _, path := range ctx.files {
		if _, ok := ctx.diskToOriginal[path]; !ok {
			t.Fatalf("path %q has no diskToOriginal entry", path)
		}
		seen[path] = struct{}{}
	}
	if len(seen) != 2 {
		t.Fatalf("distinct disk paths = %d, want 2", len(seen))
	}
}

// A redirect target is filtered inside the HTTP client, so the policy verdict
// surfaces from client.Do rather than from the pre-flight check. It used to be
// interpolated into the response body, so a redirect described the allow-list,
// the deny-list or the IP policy where the first hop returns a generic 403.
func TestNewContext_DownloadFromRedirectVerdictStaysGeneric(t *testing.T) {
	private := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="secret.txt"`)
		_, _ = w.Write([]byte("internal"))
	}))
	defer private.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, private.URL+"/secret", http.StatusFound)
	}))
	defer redirector.Close()

	payload, err := json.Marshal([]downloadFrom{{Url: redirector.URL + "/start"}})
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

	// The first hop is allowed, the redirect target is denied by the deny-list.
	denyList := []*regexp2.Regexp{regexp2.MustCompile("^"+regexp.QuoteMeta(private.URL), 0)}

	_, cancel, err := newContext(echoCtx, logger, fs, 10*time.Second, 0, downloadFromConfig{
		denyList: denyList,
		maxRetry: 0,
	})
	if cancel != nil {
		defer cancel()
	}
	if err == nil {
		t.Fatal("expected the redirect to a denied host to fail")
	}

	status, message := ParseError(err)
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: a filtered redirect must answer like a filtered first hop", status, http.StatusForbidden)
	}
	if message != http.StatusText(http.StatusForbidden) {
		t.Fatalf("message = %q, want the generic %q", message, http.StatusText(http.StatusForbidden))
	}
	// The response must not name the policy, the pattern, or the blocked host.
	for _, leak := range []string{"denied list", "allowed list", "non-public", "expression", private.URL} {
		if strings.Contains(message, leak) {
			t.Fatalf("response message %q leaks %q", message, leak)
		}
	}
}
