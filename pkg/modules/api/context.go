package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/labstack/echo/v4"
	"github.com/mholt/archives"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/unicode/norm"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

var (
	// ErrContextAlreadyClosed happens when the context has been canceled.
	ErrContextAlreadyClosed = errors.New("context already closed")

	// ErrOutOfBoundsOutputPath happens when an output path is not within
	// context's working directory. It enforces having all the files in the
	// same directory.
	ErrOutOfBoundsOutputPath = errors.New("output path is not within context's working directory")
)

// Context is the request context for a "multipart/form-data" requests.
type Context struct {
	dirPath     string
	values      map[string][]string
	files       map[string]string
	outputPaths []string
	cancelled   bool

	logger     *zap.Logger
	echoCtx    echo.Context
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

type downloadFrom struct {
	// Url is the URL to download a file from.
	Url string `json:"url"`

	// ExtraHttpHeaders are the HTTP headers to send alongside.
	ExtraHttpHeaders map[string]string `json:"extraHttpHeaders"`
}

// newContext returns a [Context] by parsing a "multipart/form-data" request.
func newContext(echoCtx echo.Context, logger *zap.Logger, fs *gotenberg.FileSystem, timeout time.Duration, bodyLimit int64, downloadFromCfg downloadFromConfig, traceHeader, trace string) (*Context, context.CancelFunc, error) {
	processCtx, processCancel := context.WithTimeout(context.Background(), timeout)

	// We want to make sure the multipart/form-data does not exceed a given
	// limit. We consider: form fields (keys, values, files) and files
	// downloaded remotely ("download from" feature).
	var totalBytesRead atomic.Int64

	addReadBytes := func(n int64) error {
		newTotal := totalBytesRead.Add(n)
		if bodyLimit != 0 && newTotal > bodyLimit {
			return WrapError(
				fmt.Errorf("body limit reached (> %d)", bodyLimit),
				NewSentinelHttpError(http.StatusRequestEntityTooLarge, http.StatusText(http.StatusRequestEntityTooLarge)),
			)
		}
		return nil
	}

	ctx := &Context{
		outputPaths: make([]string, 0),
		cancelled:   false,
		logger:      logger,
		echoCtx:     echoCtx,
		mkdirAll:    new(gotenberg.OsMkdirAll),
		pathRename:  new(gotenberg.OsPathRename),
		Context:     processCtx,
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
				ctx.logger.Error(fmt.Sprintf("remove context's working directory: %s", err))

				return
			}

			ctx.logger.Debug(fmt.Sprintf("'%s' context's working directory removed", ctx.dirPath))
			ctx.cancelled = true
		}
	}()

	form, err := echoCtx.MultipartForm()
	if err != nil {
		if errors.Is(err, http.ErrNotMultipart) {
			return nil, cancel, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHttpError(http.StatusUnsupportedMediaType, "Invalid 'Content-Type' header value: want 'multipart/form-data'"),
			)
		}

		if errors.Is(err, http.ErrMissingBoundary) {
			return nil, cancel, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHttpError(http.StatusUnsupportedMediaType, "Invalid 'Content-Type' header value: no boundary"),
			)
		}

		if strings.Contains(err.Error(), io.EOF.Error()) {
			return nil, cancel, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHttpError(http.StatusBadRequest, "Malformed body: it does not match the 'Content-Type' header boundaries"),
			)
		}

		return nil, cancel, fmt.Errorf("get multipart form: %w", err)
	}

	// This will ensure we do not exceed the body limit.
	var formValuesSize int64
	for key, valArray := range form.Value {
		formValuesSize += int64(len(key))
		for _, val := range valArray {
			formValuesSize += int64(len(val))
		}
	}
	err = addReadBytes(formValuesSize)
	if err != nil {
		return nil, cancel, fmt.Errorf("add read bytes: %w", err)
	}

	dirPath, err := fs.MkdirAll()
	if err != nil {
		return nil, cancel, fmt.Errorf("create working directory: %w", err)
	}

	ctx.dirPath = dirPath
	ctx.values = form.Value
	ctx.files = make(map[string]string)

	// First, try to download files listed in the "downloadFrom" form field, if
	// any.
	raw, ok := ctx.values["downloadFrom"]
	if !downloadFromCfg.disable && ok {
		var dls []downloadFrom
		err = json.Unmarshal([]byte(raw[0]), &dls)
		if err != nil {
			return nil, cancel, WrapError(
				fmt.Errorf("unmarshal json: %w", err),
				NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'downloadFrom' form field value: %s", err)),
			)
		}

		eg, _ := errgroup.WithContext(ctx)
		for i, dl := range dls {
			eg.Go(func() error {
				deadline, ok := ctx.Deadline()
				if !ok {
					// Should not happen, as context is created with a timeout.
					return errors.New("context has no deadline")
				}

				if strings.TrimSpace(dl.Url) == "" {
					return WrapError(
						errors.New("empty download from URL"),
						NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'downloadFrom' form field entry %d: URL must be set", i)),
					)
				}

				err := gotenberg.FilterDeadline(downloadFromCfg.allowList, downloadFromCfg.denyList, dl.Url, deadline)
				if err != nil {
					return fmt.Errorf("filter URL: %w", err)
				}

				logger.Debug(fmt.Sprintf("download file from '%s'", dl.Url))

				req, err := retryablehttp.NewRequest(http.MethodGet, dl.Url, nil)
				if err != nil {
					return fmt.Errorf("create request to '%s': %w", dl.Url, err)
				}

				req.Header.Set("User-Agent", "Gotenberg")
				for key, value := range dl.ExtraHttpHeaders {
					req.Header.Set(key, value)
				}
				req.Header.Set(traceHeader, trace)

				client := &retryablehttp.Client{
					HTTPClient: &http.Client{
						Timeout: time.Until(deadline),
					},
					RetryMax:     downloadFromCfg.maxRetry,
					RetryWaitMin: time.Duration(1) * time.Second,
					RetryWaitMax: time.Until(deadline),
					Logger:       gotenberg.NewLeveledLogger(logger),
					CheckRetry:   retryablehttp.DefaultRetryPolicy,
					Backoff:      retryablehttp.DefaultBackoff,
				}

				resp, err := client.Do(req)
				if err != nil {
					return WrapError(
						fmt.Errorf("download file from to '%s': %w", dl.Url, err),
						NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Unable to download file from '%s': %s", dl.Url, err)),
					)
				}
				defer func() {
					err := resp.Body.Close()
					if err != nil {
						logger.Error(fmt.Sprintf("close response body from '%s': %s", dl.Url, err))
					}
				}()

				if resp.StatusCode != http.StatusOK {
					return WrapError(
						fmt.Errorf("download file from to '%s': got status: '%s'", dl.Url, resp.Status),
						NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Unable to download file from '%s': got status: '%s'", dl.Url, resp.Status)),
					)
				}

				contentDisposition := resp.Header.Get("Content-Disposition")
				if contentDisposition == "" {
					return WrapError(
						fmt.Errorf("no 'Content-Disposition' header from '%s'", dl.Url),
						NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("No 'Content-Disposition' header from '%s'", dl.Url)),
					)
				}

				// FIXME: the implementation of this method might not be
				//  complete, as it fails to parse an empty mediatype.
				//  See: https://github.com/golang/go/issues/69551.
				_, params, err := mime.ParseMediaType(contentDisposition)
				if err != nil {
					return WrapError(
						fmt.Errorf("parse 'Content-Disposition' header '%s' from '%s': %w", contentDisposition, dl.Url, err),
						NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'Content-Disposition' header '%s' from '%s': %s", contentDisposition, dl.Url, err)),
					)
				}

				filename, ok := params["filename"]
				if !ok {
					return WrapError(
						fmt.Errorf("get filename from 'Content-Disposition' header '%s' from '%s'", contentDisposition, dl.Url),
						NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid 'Content-Disposition' header '%s' from '%s': no filename", contentDisposition, dl.Url)),
					)
				}

				// Avoid directory traversal and make sure filename characters are
				// normalized.
				// See: https://github.com/gotenberg/gotenberg/issues/662.
				filename = norm.NFC.String(filepath.Base(filename))
				path := fmt.Sprintf("%s/%s", ctx.dirPath, filename)

				out, err := os.Create(path)
				if err != nil {
					return fmt.Errorf("create local file: %w", err)
				}
				defer func() {
					err := out.Close()
					if err != nil {
						logger.Error(fmt.Sprintf("close local file: %s", err))
					}
				}()

				// This will ensure we do not exceed the body limit.
				reader := &trackingReader{R: resp.Body, AddReadBytes: addReadBytes}

				_, err = io.Copy(out, reader)
				if err != nil {
					return fmt.Errorf("copy downloaded file from '%s' to local file: %w", dl.Url, err)
				}

				ctx.files[filename] = path

				return nil
			})
		}

		err = eg.Wait()
		if err != nil {
			return ctx, cancel, err
		}
	}

	copyToDisk := func(fh *multipart.FileHeader) error {
		in, err := fh.Open()
		if err != nil {
			return fmt.Errorf("open multipart file: %w", err)
		}

		defer func() {
			err := in.Close()
			if err != nil {
				logger.Error(fmt.Sprintf("close file header: %s", err))
			}
		}()

		// This will ensure we do not exceed the body limit.
		reader := &trackingReader{R: in, AddReadBytes: addReadBytes}

		// Avoid directory traversal and make sure filename characters are
		// normalized.
		// See: https://github.com/gotenberg/gotenberg/issues/662.
		filename := norm.NFC.String(filepath.Base(fh.Filename))
		path := fmt.Sprintf("%s/%s", ctx.dirPath, filename)

		out, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create local file: %w", err)
		}
		defer func() {
			err := out.Close()
			if err != nil {
				logger.Error(fmt.Sprintf("close local file: %s", err))
			}
		}()

		_, err = io.Copy(out, reader)
		if err != nil {
			return fmt.Errorf("copy multipart file to local file: %w", err)
		}

		ctx.files[filename] = path

		return nil
	}

	// Then, copy the form files, if any.
	for _, files := range form.File {
		for _, fh := range files {
			err = copyToDisk(fh)
			if err != nil {
				return ctx, cancel, fmt.Errorf("copy to disk: %w", err)
			}
		}
	}

	ctx.Log().Debug(fmt.Sprintf("form fields: %+v", ctx.values))
	ctx.Log().Debug(fmt.Sprintf("form files: %+v", ctx.files))
	ctx.Log().Debug(fmt.Sprintf("total bytes: %d", totalBytesRead.Load()))

	return ctx, cancel, err
}

// Request returns the [http.Request].
func (ctx *Context) Request() *http.Request {
	return ctx.echoCtx.Request()
}

// FormData return a [FormData].
func (ctx *Context) FormData() *FormData {
	return &FormData{
		values: ctx.values,
		files:  ctx.files,
		errors: nil,
	}
}

// GeneratePath generates a path within the context's working directory.
// It generates a new UUID-based filename. It does not create a file.
func (ctx *Context) GeneratePath(extension string) string {
	return fmt.Sprintf("%s/%s%s", ctx.dirPath, uuid.New().String(), extension)
}

// GeneratePathFromFilename generates a path within the context's working
// directory, using the given filename (with extension). It does not create
// a file.
func (ctx *Context) GeneratePathFromFilename(filename string) string {
	return fmt.Sprintf("%s/%s", ctx.dirPath, filename)
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
	ctx.Log().Debug(fmt.Sprintf("rename %s to %s", oldpath, newpath))
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

// Log returns the context [zap.Logger].
func (ctx *Context) Log() *zap.Logger {
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
		ctx.logger.Debug(fmt.Sprintf("only one output file '%s', skip archive creation", ctx.outputPaths[0]))
		return ctx.outputPaths[0], nil
	}

	filesInfo, err := archives.FilesFromDisk(ctx.Context, nil, func() map[string]string {
		f := make(map[string]string)
		for _, outputPath := range ctx.outputPaths {
			f[outputPath] = ""
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
			ctx.logger.Error(fmt.Sprintf("close zip file: %s", err))
		}
	}(out)

	err = archives.Zip{}.Archive(ctx.Context, out, filesInfo)
	if err != nil {
		return "", fmt.Errorf("archive output files: %w", err)
	}

	ctx.logger.Debug(fmt.Sprintf("archive '%s' created", archivePath))

	return archivePath, nil
}

// OutputFilename returns the filename based on the given output path or the
// "Gotenberg-Output-Filename" header's value.
func (ctx *Context) OutputFilename(outputPath string) string {
	filename := ctx.echoCtx.Get("outputFilename").(string)

	if filename == "" {
		return filepath.Base(outputPath)
	}

	return fmt.Sprintf("%s%s", filename, filepath.Ext(outputPath))
}
