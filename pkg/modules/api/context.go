package api

import (
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/labstack/echo/v4"
	"github.com/mholt/archiver/v3"
	"go.uber.org/zap"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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
	dirPath string
	values  map[string][]string
	files   map[string]string

	outputPaths []string

	cancelled bool
	logger    *zap.Logger
	echoCtx   echo.Context
	context.Context
}

// newContext returns a Context by parsing a "multipart/form-data" request.
func newContext(echoCtx echo.Context, logger *zap.Logger, timeout time.Duration) (*Context, context.CancelFunc, error) {
	processCtx, processCancel := context.WithTimeout(context.Background(), timeout)

	ctx := &Context{
		outputPaths: make([]string, 0),
		cancelled:   false,
		logger:      logger,
		echoCtx:     echoCtx,
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

			ctx.logger.Debug(fmt.Sprintf("'%s' removed", ctx.dirPath))
			ctx.cancelled = true
		}
	}()

	form, err := echoCtx.MultipartForm()
	if err != nil {

		if errors.Is(err, http.ErrNotMultipart) {
			return nil, cancel, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHTTPError(http.StatusUnsupportedMediaType, "Invalid 'Content-Type' header value: want 'multipart/form-data'"),
			)
		}

		if errors.Is(err, http.ErrMissingBoundary) {
			return nil, cancel, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHTTPError(http.StatusUnsupportedMediaType, "Invalid 'Content-Type' header value: no boundary"),
			)
		}

		if strings.Contains(err.Error(), io.EOF.Error()) {
			return nil, cancel, WrapError(
				fmt.Errorf("get multipart form: %w", err),
				NewSentinelHTTPError(http.StatusBadRequest, "Malformed body: it does not match the 'Content-Type' header boundaries"),
			)
		}

		return nil, cancel, fmt.Errorf("get multipart form: %w", err)
	}

	dirPath, err := gotenberg.MkdirAll()
	if err != nil {
		return nil, cancel, fmt.Errorf("create working directory: %w", err)
	}

	ctx.dirPath = dirPath
	ctx.values = form.Value
	ctx.files = make(map[string]string)

	copyToDisk := func(fh *multipart.FileHeader) error {
		// Avoid directory traversal and normalize filename.
		// See https://github.com/gotenberg/gotenberg/issues/104.
		t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

		filename, _, err := transform.String(t, filepath.Base(fh.Filename))
		if err != nil {
			return fmt.Errorf("transform filename: %w", err)
		}

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

		_, err = io.Copy(out, in)
		if err != nil {
			return fmt.Errorf("copy multipart file to local file: %w", err)
		}

		ctx.files[filename] = path

		return nil
	}

	for _, files := range form.File {
		for _, fh := range files {
			err = copyToDisk(fh)

			if err != nil {
				return ctx, cancel, fmt.Errorf("copy to disk: %w", err)
			}
		}
	}

	ctx.Log().Debug(fmt.Sprintf("form data values: %+v", ctx.values))
	ctx.Log().Debug(fmt.Sprintf("form data files: %+v", ctx.files))

	return ctx, cancel, err
}

// Request returns the http.Request.
func (ctx Context) Request() *http.Request {
	return ctx.echoCtx.Request()
}

// FormData return a FormData.
func (ctx Context) FormData() *FormData {
	return &FormData{
		values: ctx.values,
		files:  ctx.files,
		errors: nil,
	}
}

// GeneratePath generates a path within the context's working directory. It
// does not create a file.
func (ctx Context) GeneratePath(extension string) string {
	return fmt.Sprintf("%s/%s%s", ctx.dirPath, uuid.New(), extension)
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

// Log returns the context zap.Logger.
func (ctx Context) Log() *zap.Logger {
	return ctx.logger
}

// BuildOutputFile builds the output file according to the output paths
// registered in the context. If many output paths, an archive is created.
func (ctx Context) BuildOutputFile() (string, error) {
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

	z := archiver.Zip{
		CompressionLevel:       flate.DefaultCompression,
		MkdirAll:               true,
		SelectiveCompression:   true,
		ContinueOnError:        false,
		OverwriteExisting:      false,
		ImplicitTopLevelFolder: false,
	}

	archivePath := ctx.GeneratePath(".zip")

	err := z.Archive(ctx.outputPaths, archivePath)
	if err != nil {
		return "", fmt.Errorf("archive output files: %w", err)
	}

	ctx.logger.Debug(fmt.Sprintf("archive '%s' created", archivePath))

	return archivePath, nil
}

// OutputFilename returns the filename based on the given output path or the
// "Gotenberg-Output-Filename" header's value.
func (ctx Context) OutputFilename(outputPath string) string {
	filename := ctx.echoCtx.Request().Header.Get("Gotenberg-Output-Filename")

	if filename == "" {
		return filepath.Base(outputPath)
	}

	return fmt.Sprintf("%s%s", filename, filepath.Ext(outputPath))
}
