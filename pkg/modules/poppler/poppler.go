package poppler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"syscall"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Poppler))
}

// Poppler abstracts the CLI tool pdftoppm and rasterizes PDF files into images.
type Poppler struct {
	binPath string

	version     string
	versionOnce sync.Once
}

// ImageConvertOptions are the options for [Poppler.Rasterize].
type ImageConvertOptions struct {
	// Format is the output image format: "png", "jpeg", or "tiff".
	Format string

	// Dpi is the resolution, in dots per inch, used to rasterize each page.
	Dpi int

	// Quality is the JPEG quality, from 1 to 100. Ignored for other formats.
	Quality int

	// FirstPage is the first page to rasterize. Zero means the first page.
	FirstPage int

	// LastPage is the last page to rasterize. Zero means the last page.
	LastPage int
}

// Descriptor returns a [Poppler]'s module descriptor.
func (engine *Poppler) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "poppler",
		New: func() gotenberg.Module { return new(Poppler) },
	}
}

// Provision sets the module properties.
func (engine *Poppler) Provision(_ *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("PDFTOPPM_BIN_PATH")
	if !ok {
		return errors.New("PDFTOPPM_BIN_PATH environment variable is not set; set it to the absolute path of the pdftoppm binary")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine *Poppler) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("pdftoppm binary path does not exist: %w", err)
	}

	return nil
}

// Debug returns additional debug data.
func (engine *Poppler) Debug() map[string]any {
	return map[string]any{"version": engine.detectVersion()}
}

// Routes returns the routes provided by [Poppler].
func (engine *Poppler) Routes() ([]api.Route, error) {
	return []api.Route{
		convertImageRoute(engine),
	}, nil
}

// detectVersion resolves the pdftoppm version once, preferring the value
// captured at image build time so it never spawns pdftoppm at runtime. It
// falls back to running pdftoppm -v for local or non-Docker builds.
func (engine *Poppler) detectVersion() string {
	engine.versionOnce.Do(func() {
		if v, ok := gotenberg.BuildVersion("pdftoppm"); ok {
			engine.version = v
			return
		}

		cmd := exec.Command(engine.binPath, "-v") //nolint:gosec
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// pdftoppm writes its version banner to stderr.
		output, err := cmd.CombinedOutput()
		if err != nil {
			engine.version = err.Error()
			return
		}

		lines := bytes.SplitN(output, []byte("\n"), 2)
		if len(lines) > 0 {
			engine.version = string(bytes.TrimSpace(lines[0]))
			return
		}

		engine.version = "Unable to determine Poppler version"
	})

	return engine.version
}

// spanAttrs returns the client-span attributes for a pdftoppm invocation: the
// server address and the pdftoppm version, plus any extra attributes. The
// version rides on every span so a trace records which pdftoppm ran the
// operation.
func (engine *Poppler) spanAttrs(extra ...attribute.KeyValue) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 2+len(extra))
	attrs = append(attrs, semconv.ServerAddress(engine.binPath))
	if v := engine.detectVersion(); v != "" {
		attrs = append(attrs, attribute.String("gotenberg.poppler.version", v))
	}

	return append(attrs, extra...)
}

// formatFlag maps an [ImageConvertOptions.Format] to its pdftoppm flag.
func formatFlag(format string) string {
	switch format {
	case "jpeg":
		return "-jpeg"
	case "tiff":
		return "-tiff"
	default:
		return "-png"
	}
}

// Rasterize converts each page of the PDF at inputPath into an image, writing
// the images into outputDirPath. It returns the generated image paths in page
// order. The caller owns outputDirPath and is expected to provide an empty
// directory.
func (engine *Poppler) Rasterize(ctx context.Context, logger *slog.Logger, inputPath, outputDirPath string, opts ImageConvertOptions) ([]string, error) {
	ctx, span := gotenberg.Tracer().Start(ctx, "poppler.Rasterize",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(engine.spanAttrs()...),
	)
	defer span.End()

	args := make([]string, 0, 12)
	args = append(args, "-r", strconv.Itoa(opts.Dpi))
	args = append(args, formatFlag(opts.Format))
	if opts.Format == "jpeg" && opts.Quality > 0 {
		args = append(args, "-jpegopt", fmt.Sprintf("quality=%d", opts.Quality))
	}
	if opts.FirstPage > 0 {
		args = append(args, "-f", strconv.Itoa(opts.FirstPage))
	}
	if opts.LastPage > 0 {
		args = append(args, "-l", strconv.Itoa(opts.LastPage))
	}
	// pdftoppm writes <root>-<page>.<ext> for every page.
	args = append(args, inputPath, filepath.Join(outputDirPath, "page"))

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("rasterize PDF with pdftoppm: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	paths, err := imagePaths(outputDirPath)
	if err != nil {
		err = fmt.Errorf("collect generated images: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if len(paths) == 0 {
		err = errors.New("pdftoppm produced no images")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return paths, nil
}

// imagePaths lists the files generated by pdftoppm in dirPath, sorted in page
// order. pdftoppm zero-pads the page number, so a lexical sort matches the page
// order.
func imagePaths(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read output directory: %w", err)
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(dirPath, entry.Name()))
	}

	sort.Strings(paths)
	return paths, nil
}
