package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v5"
	"github.com/mholt/archives"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/unicode/norm"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

var (
	// ErrContextAlreadyClosed happens when the context has been canceled.
	ErrContextAlreadyClosed = errors.New("context already closed")

	// ErrOutOfBoundsOutputPath happens when an output path is not within
	// context's working directory. It enforces having all the files in the
	// same directory.
	ErrOutOfBoundsOutputPath = errors.New("output path is not within context's working directory")
)

// Context is the request context for a "multipart/form-data" request.
type Context struct {
	dirPath        string
	values         map[string][]string
	files          map[string]string
	diskToOriginal map[string]string
	outputPaths    []string
	cancelled      bool

	// fileOrder records the order files were received in, keyed by disk path.
	// It breaks ties when two uploads share an original filename, so that
	// de-duplicated files keep their upload order instead of being ordered by
	// the suffix uniqueFilename added.
	fileOrder map[string]int

	// fileBase maps a disk path to the original filename as received, before
	// de-duplication. Sorting on it keeps a de-duplicated file next to its
	// twin rather than wherever its numbered name would land.
	fileBase map[string]string

	// outputFilename is the sanitized Gotenberg-Output-Filename header,
	// snapshotted while the [echo.Context] is still live. Echo returns that
	// context to a pool as soon as the handler returns, and an asynchronous
	// conversion outlives it, so reading the header from the pooled store later
	// yields whichever request happens to own it by then.
	outputFilename string

	logger     *slog.Logger
	echoCtx    *echo.Context
	mkdirAll   gotenberg.MkdirAll
	pathRename gotenberg.PathRename
	context.Context
}

type trackingReader struct {
	R            io.Reader
	AddReadBytes func(n int64) error
}

func (t *trackingReader) Read(p []byte) (int, error) {
	n, err := t.R.Read(p)
	if n > 0 {
		errAddRead := t.AddReadBytes(int64(n))
		if errAddRead != nil {
			return n, fmt.Errorf("add read bytes: %w", errAddRead)
		}
	}
	if err != nil {
		// It's a common practice in Go to return io.EOF unwrapped to signal
		// the end of a data stream. Wrapping it can lead to unexpected
		// behavior in standard library functions.
		return n, err
	}
	return n, nil
}

// errTooManyDownloadFromEntries is returned by [decodeDownloadFrom] when the
// array holds more entries than the configured maximum.
var errTooManyDownloadFromEntries = errors.New("too many downloadFrom entries")

// decodeDownloadFrom decodes the downloadFrom form field, refusing to
// accumulate more than maxEntries. A maxEntries of 0 means no limit.
//
// It decodes element by element rather than calling [json.Unmarshal] on the
// whole value. A compact array such as "[{},{},{}]" costs three bytes per
// entry on the wire and expands to roughly seventy times that once
// unmarshalled, so counting the entries afterwards is too late to bound the
// allocation. Streaming keeps the cost proportional to maxEntries no matter
// how long the array is.
func decodeDownloadFrom(raw string, maxEntries int) ([]downloadFrom, error) {
	dec := json.NewDecoder(strings.NewReader(raw))

	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return nil, fmt.Errorf("expected a JSON array, got '%v'", token)
	}

	var dls []downloadFrom
	for dec.More() {
		if maxEntries > 0 && len(dls) >= maxEntries {
			return nil, errTooManyDownloadFromEntries
		}

		var dl downloadFrom
		err = dec.Decode(&dl)
		if err != nil {
			return nil, err
		}

		dls = append(dls, dl)
	}

	return dls, nil
}

type downloadFrom struct {
	// Url is the URL to download a file from.
	Url string `json:"url"`

	// ExtraHttpHeaders are the HTTP headers to send alongside.
	ExtraHttpHeaders map[string]string `json:"extraHttpHeaders"`
}

// bodyBudget counts the bytes a request reads against --api-body-limit (env
// API_BODY_LIMIT): form fields (keys, values, files) and files downloaded
// remotely ("download from" feature). A limit of 0 means no limit.
type bodyBudget struct {
	limit int64
	read  atomic.Int64
}

// add counts n more bytes, and returns an error once over the limit.
func (b *bodyBudget) add(n int64) error {
	total := b.read.Add(n)
	if b.limit != 0 && total > b.limit {
		return b.exceeded()
	}
	return nil
}

// exceeded returns the error of a request over the limit.
func (b *bodyBudget) exceeded() error {
	return WrapError(
		fmt.Errorf("body limit reached (> %d)", b.limit),
		NewSentinelHttpError(http.StatusRequestEntityTooLarge, "The request body exceeds the configured size limit. Increase it with --api-body-limit, or send a smaller request."),
	)
}

// newContext returns a [Context] by parsing a "multipart/form-data" request.
func newContext(echoCtx *echo.Context, logger *slog.Logger, fs *gotenberg.FileSystem, timeout time.Duration, bodyLimit int64, downloadFromCfg downloadFromConfig) (*Context, context.CancelFunc, error) {
	processCtx, processCancel := context.WithTimeout(echoCtx.Request().Context(), timeout)
	budget := &bodyBudget{limit: bodyLimit}

	// Snapshot now, while echoCtx still belongs to this request.
	outputFilename, _ := echoCtx.Get("outputFilename").(string)

	ctx := &Context{
		outputPaths:    make([]string, 0),
		cancelled:      false,
		outputFilename: outputFilename,
		logger:         logger,
		echoCtx:        echoCtx,
		mkdirAll:       new(gotenberg.OsMkdirAll),
		pathRename:     new(gotenberg.OsPathRename),
		Context:        processCtx,
	}

	// A custom cancel function which removes the context's working directory
	// when called.
	cancel := func() context.CancelFunc {
		return func() {
			if ctx.cancelled {
				return
			}

			processCancel()

			if ctx.dirPath == "" {
				return
			}

			err := os.RemoveAll(ctx.dirPath)
			if err != nil {
				ctx.logger.ErrorContext(context.Background(), fmt.Sprintf("remove context's working directory: %s", err))

				return
			}

			ctx.logger.DebugContext(context.Background(), fmt.Sprintf("'%s' context's working directory removed", ctx.dirPath))
			ctx.cancelled = true
		}
	}()

	form, err := parseMultipartForm(echoCtx, budget)
	if err != nil {
		return nil, cancel, err
	}
	defer func() {
		err := form.RemoveAll()
		if err != nil {
			logger.ErrorContext(context.Background(), fmt.Sprintf("remove multipart temporary files: %s", err))
		}
	}()

	dirPath, err := fs.MkdirAll()
	if err != nil {
		return nil, cancel, fmt.Errorf("create working directory: %w", err)
	}

	ctx.dirPath = dirPath
	ctx.values = form.Value
	ctx.files = make(map[string]string)
	ctx.diskToOriginal = make(map[string]string)

	// First, try to download files listed in the "downloadFrom" form field, if
	// any.
	raw, ok := ctx.values["downloadFrom"]
	if !downloadFromCfg.disable && ok {
		err = ctx.downloadFrom(raw[0], downloadFromCfg, budget)
		if err != nil {
			return ctx, cancel, err
		}
	}

	// Then, copy the form files, if any.
	for _, files := range form.File {
		for _, fh := range files {
			err = ctx.copyToDisk(fh, budget)
			if err != nil {
				return ctx, cancel, fmt.Errorf("copy to disk: %w", err)
			}
		}
	}

	ctx.linkOriginalFilenames()

	ctx.Log().DebugContext(ctx, fmt.Sprintf("form fields: %+v", ctx.values))
	ctx.Log().DebugContext(ctx, fmt.Sprintf("form files: %+v", ctx.files))
	ctx.Log().DebugContext(ctx, fmt.Sprintf("total bytes: %d", budget.read.Load()))

	return ctx, cancel, nil
}

// parseMultipartForm parses the request body, and counts the form values
// against budget.
func parseMultipartForm(echoCtx *echo.Context, budget *bodyBudget) (*multipart.Form, error) {
	// The multipart parser reads the whole request and spools file parts to
	// disk before the budget sees a single byte, so the limit must also bound
	// the raw body itself. Content-Length allows an early rejection, but it
	// may be absent or lie, hence the reader.
	if budget.limit > 0 {
		req := echoCtx.Request()
		if req.ContentLength > budget.limit {
			return nil, fmt.Errorf("check content length: %w", budget.exceeded())
		}
		req.Body = http.MaxBytesReader(echoCtx.Response(), req.Body, budget.limit)
	}

	form, err := echoCtx.MultipartForm()
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, fmt.Errorf("get multipart form: %w", budget.exceeded())
		}

		if errors.Is(err, http.ErrNotMultipart) {
			return nil, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHttpError(http.StatusUnsupportedMediaType, "Invalid 'Content-Type' header value: want 'multipart/form-data'"),
			)
		}

		if errors.Is(err, http.ErrMissingBoundary) {
			return nil, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHttpError(http.StatusUnsupportedMediaType, "Invalid 'Content-Type' header value: no boundary"),
			)
		}

		if strings.Contains(err.Error(), io.EOF.Error()) {
			return nil, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHttpError(http.StatusBadRequest, "Malformed body: it does not match the 'Content-Type' header boundaries"),
			)
		}

		return nil, fmt.Errorf("get multipart form: %w", err)
	}

	var formValuesSize int64
	for key, valArray := range form.Value {
		formValuesSize += int64(len(key))
		for _, val := range valArray {
			formValuesSize += int64(len(val))
		}
	}

	err = budget.add(formValuesSize)
	if err != nil {
		removeErr := form.RemoveAll()
		if removeErr != nil {
			return nil, errors.Join(fmt.Errorf("add read bytes: %w", err), fmt.Errorf("remove multipart temporary files: %w", removeErr))
		}

		return nil, fmt.Errorf("add read bytes: %w", err)
	}

	return form, nil
}

// downloadFrom downloads the files the "downloadFrom" form field lists into
// the working directory.
func (ctx *Context) downloadFrom(raw string, cfg downloadFromConfig, budget *bodyBudget) error {
	dls, err := decodeDownloadFrom(raw, cfg.maxEntries)
	if errors.Is(err, errTooManyDownloadFromEntries) {
		return WrapError(
			fmt.Errorf("decode downloadFrom: %w", err),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'downloadFrom' form field value: too many entries, the maximum is %d", cfg.maxEntries)),
		)
	}
	if err != nil {
		return WrapError(
			fmt.Errorf("unmarshal json: %w", err),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'downloadFrom' form field value: %s", err)),
		)
	}

	// Each goroutine writes to its own results slot. This goroutine merges
	// into ctx.files and ctx.diskToOriginal after eg.Wait() to avoid
	// concurrent map writes.
	type downloadFromResult struct {
		filename, path string
	}
	results := make([]downloadFromResult, len(dls))

	// The group context cancels the remaining downloads as soon as one
	// fails: the request fails anyway.
	eg, groupCtx := errgroup.WithContext(ctx)
	// Bound the number of in-flight downloads. Each entry allocates a
	// retryable client, an outbound transport, a span, and logger state, so
	// an unbounded array would otherwise exhaust process memory. A value of 0
	// keeps the fan-out unbounded.
	if cfg.maxConcurrency > 0 {
		eg.SetLimit(cfg.maxConcurrency)
	}
	for i, dl := range dls {
		eg.Go(func() error {
			filename, path, err := ctx.downloadOne(groupCtx, i, dl, cfg, budget)
			if err != nil {
				return err
			}

			results[i] = downloadFromResult{filename: filename, path: path}

			return nil
		})
	}

	err = eg.Wait()
	if err != nil {
		return err
	}

	for _, r := range results {
		filename := ctx.uniqueFilename(r.filename)
		ctx.files[filename] = r.path
		ctx.diskToOriginal[r.path] = filename
		ctx.trackFileOrder(r.path, r.filename)
	}

	return nil
}

// downloadOne downloads the i-th "downloadFrom" entry into the working
// directory, and returns its sanitized filename and its path on disk.
func (ctx *Context) downloadOne(groupCtx context.Context, i int, dl downloadFrom, cfg downloadFromConfig, budget *bodyBudget) (filename, path string, err error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		// Should not happen, as context is created with a timeout.
		return "", "", errors.New("context has no deadline")
	}

	if strings.TrimSpace(dl.Url) == "" {
		return "", "", WrapError(
			errors.New("empty download from URL"),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'downloadFrom' form field entry %d: URL must be set", i)),
		)
	}

	ipOpts := []gotenberg.DecideOption{
		gotenberg.WithDenyPrivateIPs(cfg.denyPrivateIPs),
		gotenberg.WithDenyPublicIPs(cfg.denyPublicIPs),
	}
	err = gotenberg.FilterOutboundURL(groupCtx, dl.Url, cfg.allowList, cfg.denyList, deadline, ipOpts...)
	if err != nil {
		return "", "", fmt.Errorf("filter URL: %w", err)
	}

	dlCtx, dlSpan := gotenberg.Tracer().Start(groupCtx, "GET Download From",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(dl.Url)),
	)
	defer func() {
		if err != nil {
			dlSpan.RecordError(err)
			dlSpan.SetStatus(codes.Error, err.Error())
		} else {
			dlSpan.SetStatus(codes.Ok, "")
		}
		dlSpan.End()
	}()

	ctx.logger.DebugContext(dlCtx, fmt.Sprintf("download file from '%s'", dl.Url))

	// The request must carry dlCtx: retryablehttp.NewRequest builds on
	// context.Background(), and its wait between attempts is a select on the
	// request context, so a contextless request cannot be interrupted by
	// --api-timeout (env API_TIMEOUT) or by the caller going away.
	req, err := retryablehttp.NewRequestWithContext(dlCtx, http.MethodGet, dl.Url, nil)
	if err != nil {
		return "", "", fmt.Errorf("create request to '%s': %w", dl.Url, err)
	}

	req.Header.Set("User-Agent", "Gotenberg")
	for key, value := range dl.ExtraHttpHeaders {
		req.Header.Set(key, value)
	}

	// Inject OTEL trace context into outbound request.
	otel.GetTextMapPropagator().Inject(dlCtx, propagation.HeaderCarrier(req.Header))

	// Propagate correlation ID header.
	if correlationIdHeader, ok := ctx.echoCtx.Get("correlationIdHeader").(string); ok {
		if correlationId, ok := ctx.echoCtx.Get("correlationId").(string); ok {
			req.Header.Set(correlationIdHeader, correlationId)
		}
	}

	// Entries are serialized by the concurrency limit, so a late one can start
	// after the deadline has already passed. Fail closed rather than derive a
	// non-positive timeout, which [http.Client] reads as no deadline at all.
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return "", "", fmt.Errorf("download file from '%s': %w", dl.Url, context.DeadlineExceeded)
	}

	client := &retryablehttp.Client{
		HTTPClient:   gotenberg.NewOutboundHttpClient(remaining, cfg.allowList, cfg.denyList, cfg.enableEnvironmentProxy, ipOpts...),
		RetryMax:     cfg.maxRetry,
		RetryWaitMin: time.Duration(1) * time.Second,
		RetryWaitMax: remaining,
		Logger:       gotenberg.NewLeveledLogger(ctx.logger),
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		// Not DefaultBackoff: it hands a hostile origin control of the wait
		// via Retry-After.
		Backoff: gotenberg.ClampedBackoff,
	}

	resp, err := client.Do(req)
	if err != nil {
		// A redirect target is filtered inside the client, so the policy
		// verdict surfaces here rather than from the pre-flight above. Keep it
		// out of the response: the first hop answers a filtered URL with a
		// generic 403, and a later hop must not describe the allow-list, the
		// deny-list or the IP policy instead.
		if errors.Is(err, gotenberg.ErrFiltered) {
			return "", "", fmt.Errorf("download file from '%s': %w", dl.Url, err)
		}

		return "", "", WrapError(
			fmt.Errorf("download file from '%s': %w", dl.Url, err),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Unable to download file from '%s': %s", dl.Url, err)),
		)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			ctx.logger.ErrorContext(ctx, fmt.Sprintf("close response body from '%s': %s", dl.Url, err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", "", WrapError(
			fmt.Errorf("download file from '%s': got status: '%s'", dl.Url, resp.Status),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Unable to download file from '%s': got status: '%s'", dl.Url, resp.Status)),
		)
	}

	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition == "" {
		return "", "", WrapError(
			fmt.Errorf("no 'Content-Disposition' header from '%s'", dl.Url),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("No 'Content-Disposition' header from '%s'", dl.Url)),
		)
	}

	// FIXME: the implementation of this method might not be complete, as it
	//  fails to parse an empty mediatype.
	//  See: https://github.com/golang/go/issues/69551.
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return "", "", WrapError(
			fmt.Errorf("parse 'Content-Disposition' header '%s' from '%s': %w", contentDisposition, dl.Url, err),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'Content-Disposition' header '%s' from '%s': %s", contentDisposition, dl.Url, err)),
		)
	}

	filename, ok = params["filename"]
	if !ok {
		return "", "", WrapError(
			fmt.Errorf("get filename from 'Content-Disposition' header '%s' from '%s'", contentDisposition, dl.Url),
			NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'Content-Disposition' header '%s' from '%s': no filename", contentDisposition, dl.Url)),
		)
	}

	filename = sanitizeFilename(filename)

	path, err = ctx.writeToDisk(filename, resp.Body, budget)
	if err != nil {
		return "", "", fmt.Errorf("write downloaded file from '%s': %w", dl.Url, err)
	}

	return filename, path, nil
}

// copyToDisk copies an uploaded file into the working directory.
func (ctx *Context) copyToDisk(fh *multipart.FileHeader, budget *bodyBudget) error {
	in, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open multipart file: %w", err)
	}
	defer func() {
		err := in.Close()
		if err != nil {
			ctx.logger.ErrorContext(context.Background(), fmt.Sprintf("close file header: %s", err))
		}
	}()

	filename := sanitizeFilename(fh.Filename)

	path, err := ctx.writeToDisk(filename, in, budget)
	if err != nil {
		return fmt.Errorf("copy multipart file to local file: %w", err)
	}

	unique := ctx.uniqueFilename(filename)
	ctx.files[unique] = path
	ctx.diskToOriginal[path] = unique
	ctx.trackFileOrder(path, filename)

	return nil
}

// writeToDisk writes the content of r into a new file of the working
// directory, counting it against budget, and returns its path. filename must
// be sanitized already.
func (ctx *Context) writeToDisk(filename string, r io.Reader, budget *bodyBudget) (string, error) {
	// Use a UUID-based name on disk to avoid filesystem NAME_MAX limits with
	// long filenames.
	// See: https://github.com/gotenberg/gotenberg/issues/1500.
	path := fmt.Sprintf("%s/%s", ctx.dirPath, uuid.New().String()+safeExt(filename))

	out, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create local file: %w", err)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			ctx.logger.ErrorContext(context.Background(), fmt.Sprintf("close local file: %s", err))
		}
	}()

	_, err = io.Copy(out, &trackingReader{R: r, AddReadBytes: budget.add})
	if err != nil {
		return "", fmt.Errorf("copy to local file: %w", err)
	}

	return path, nil
}

// linkOriginalFilenames creates symlinks from the original filenames to the
// UUID-based disk names, so that relative asset references (e.g., <img
// src="image.png">) resolve when Chromium navigates to a file:// URL.
//
// It is best-effort: a symlink may fail for a filename exceeding the
// filesystem NAME_MAX limit, the reason UUIDs were introduced in the first
// place, and must not decide the outcome of the request.
func (ctx *Context) linkOriginalFilenames() {
	for originalName, diskPath := range ctx.files {
		symlinkPath := fmt.Sprintf("%s/%s", ctx.dirPath, originalName)
		if symlinkPath == diskPath {
			continue
		}

		err := os.Symlink(filepath.Base(diskPath), symlinkPath)
		if err != nil {
			ctx.logger.DebugContext(context.Background(), fmt.Sprintf("skip symlink for '%s': %s", originalName, err))
		}
	}
}

// Request returns the [http.Request].
func (ctx *Context) Request() *http.Request {
	return ctx.echoCtx.Request()
}

// FormData return a [FormData].
func (ctx *Context) FormData() *FormData {
	return &FormData{
		values:         ctx.values,
		files:          ctx.files,
		diskToOriginal: ctx.diskToOriginal,
		fileOrder:      ctx.fileOrder,
		fileBase:       ctx.fileBase,
		errors:         nil,
	}
}

// FileCount returns the number of files received in the request.
func (ctx *Context) FileCount() int {
	return len(ctx.files)
}

// OriginalFilename returns the original filename associated with a disk path.
// If no mapping exists, it falls back to [filepath.Base].
func (ctx *Context) OriginalFilename(diskPath string) string {
	if original, ok := ctx.diskToOriginal[diskPath]; ok {
		return original
	}
	return filepath.Base(diskPath)
}

// RegisterDiskPath associates a disk path with an original filename so that
// [Context.OriginalFilename] can resolve it later.
func (ctx *Context) RegisterDiskPath(diskPath, originalFilename string) {
	if ctx.diskToOriginal == nil {
		ctx.diskToOriginal = make(map[string]string)
	}

	ctx.diskToOriginal[diskPath] = originalFilename
}

// DirPath returns the path to the request's working directory.
func (ctx *Context) DirPath() string {
	return ctx.dirPath
}

// GeneratePath generates a path within the context's working directory.
// It generates a new UUID-based filename. It does not create a file.
func (ctx *Context) GeneratePath(extension string) string {
	return fmt.Sprintf("%s/%s%s", ctx.dirPath, uuid.New().String(), extension)
}

// GeneratePathFromFilename generates a path within the context's working
// directory. It uses a UUID-based name on disk to avoid filesystem NAME_MAX
// limits but registers the given filename so that [Context.OriginalFilename]
// can resolve it. It does not create a file.
func (ctx *Context) GeneratePathFromFilename(filename string) string {
	safeName := uuid.New().String() + safeExt(filename)
	path := fmt.Sprintf("%s/%s", ctx.dirPath, safeName)
	ctx.RegisterDiskPath(path, filename)
	return path
}

// CreateSubDirectory creates a subdirectory within the context's working
// directory.
func (ctx *Context) CreateSubDirectory(dirName string) (string, error) {
	path := fmt.Sprintf("%s/%s", ctx.dirPath, dirName)
	err := ctx.mkdirAll.MkdirAll(path, 0o755)
	if err != nil {
		return "", fmt.Errorf("create sub-directory %s: %w", path, err)
	}
	return path, nil
}

// Rename is just a wrapper around [os.Rename], as we need to mock this
// behavior in our tests.
func (ctx *Context) Rename(oldpath, newpath string) error {
	ctx.Log().DebugContext(ctx, fmt.Sprintf("rename %s to %s", oldpath, newpath))
	err := ctx.pathRename.Rename(oldpath, newpath)
	if err != nil {
		return fmt.Errorf("rename path: %w", err)
	}
	return nil
}

// AddOutputPaths adds the given paths. Those paths will be used later to build
// the output file.
func (ctx *Context) AddOutputPaths(paths ...string) error {
	if ctx.cancelled {
		return ErrContextAlreadyClosed
	}

	for _, path := range paths {
		if !strings.HasPrefix(path, ctx.dirPath) {
			return ErrOutOfBoundsOutputPath
		}

		ctx.outputPaths = append(ctx.outputPaths, path)
	}

	return nil
}

// Log returns the context [slog.Logger].
func (ctx *Context) Log() *slog.Logger {
	return ctx.logger
}

// BuildOutputFile builds the output file according to the output paths
// registered in the context. If many output paths, an archive is created.
func (ctx *Context) BuildOutputFile() (string, error) {
	if ctx.cancelled {
		return "", ErrContextAlreadyClosed
	}

	if len(ctx.outputPaths) == 0 {
		return "", errors.New("no output path")
	}

	if len(ctx.outputPaths) == 1 {
		ctx.logger.DebugContext(ctx, fmt.Sprintf("only one output file '%s', skip archive creation", ctx.outputPaths[0]))
		return ctx.outputPaths[0], nil
	}

	filesInfo, err := archives.FilesFromDisk(ctx.Context, nil, func() map[string]string {
		f := make(map[string]string)
		for _, outputPath := range ctx.outputPaths {
			f[outputPath] = ctx.OriginalFilename(outputPath)
		}
		return f
	}())
	if err != nil {
		return "", fmt.Errorf("create files info: %w", err)
	}

	archivePath := ctx.GeneratePath(".zip")
	out, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("create zip file: %w", err)
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			ctx.logger.ErrorContext(ctx, fmt.Sprintf("close zip file: %s", err))
		}
	}(out)

	err = archives.Zip{}.Archive(ctx.Context, out, filesInfo)
	if err != nil {
		return "", fmt.Errorf("archive output files: %w", err)
	}

	ctx.logger.DebugContext(ctx, fmt.Sprintf("archive '%s' created", archivePath))

	return archivePath, nil
}

// OutputFilename returns the filename based on the given output path or the
// "Gotenberg-Output-Filename" header's value.
func (ctx *Context) OutputFilename(outputPath string) string {
	if ctx.outputFilename == "" {
		return ctx.OriginalFilename(outputPath)
	}

	filename := ctx.outputFilename

	return fmt.Sprintf("%s%s", filename, filepath.Ext(outputPath))
}

// maxDiskExtLength bounds the extension copied onto a UUID-based disk name.
// The UUID stem is 36 characters, so a longer extension risks NAME_MAX, which
// is 255 on ext4 and overlayfs. The untruncated name is kept in
// [Context.diskToOriginal], which never reaches the filesystem.
const maxDiskExtLength = 32

// safeExt returns the extension to append to a UUID-based disk name. It drops
// an extension too long to be safe rather than let [os.Create] fail with
// ENAMETOOLONG, which surfaced to the caller as a bare 500.
func safeExt(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > maxDiskExtLength {
		return ""
	}

	return ext
}

// uniqueFilename returns filename, or a numbered variant of it when the
// request already carries a file by that name.
//
// Uploads are keyed by their sanitized original filename, so two files sharing
// one name used to collide: the second overwrote the first and only one
// reached the conversion, while both stayed on disk and counted against the
// body limit. Sanitizing strips directories, so "a/doc.pdf" and "b/doc.pdf"
// collide too.
func (ctx *Context) uniqueFilename(filename string) string {
	_, exists := ctx.files[filename]
	if !exists {
		return filename
	}

	ext := filepath.Ext(filename)
	stem := strings.TrimSuffix(filename, ext)

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		_, exists = ctx.files[candidate]
		if !exists {
			return candidate
		}
	}
}

// trackFileOrder records where a file arrived in the request and the filename
// it arrived under, so [FormData.paths] can order it the way the caller sent
// it.
func (ctx *Context) trackFileOrder(path, base string) {
	if ctx.fileOrder == nil {
		ctx.fileOrder = make(map[string]int)
		ctx.fileBase = make(map[string]string)
	}

	ctx.fileOrder[path] = len(ctx.fileOrder)
	ctx.fileBase[path] = base
}

// sanitizeFilename strips path separators (including backslashes, which
// [filepath.Base] ignores on Linux) and control characters from a
// caller-supplied filename, then NFC-normalizes the result. This prevents a
// Windows-side Zip Slip when an output zip is extracted by a permissive
// extractor that interprets '\' as a path separator.
func sanitizeFilename(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	return norm.NFC.String(name)
}
