package qpdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(QPdf))
}

// QPdf abstracts the CLI tool QPDF and implements the [gotenberg.PdfEngine]
// interface.
type QPdf struct {
	binPath    string
	globalArgs []string
}

// Descriptor returns a [QPdf]'s module descriptor.
func (engine *QPdf) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "qpdf",
		New: func() gotenberg.Module { return new(QPdf) },
	}
}

// Provision sets the module properties.
func (engine *QPdf) Provision(ctx *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("QPDF_BIN_PATH")
	if !ok {
		return errors.New("QPDF_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath
	// Warnings should not cause errors.
	engine.globalArgs = []string{"--warning-exit-0"}

	return nil
}

// Validate validates the module properties.
func (engine *QPdf) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("QPDF binary path does not exist: %w", err)
	}

	return nil
}

// Debug returns additional debug data.
func (engine *QPdf) Debug() map[string]any {
	debug := make(map[string]any)

	cmd := exec.Command(engine.binPath, "--version") //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	output, err := cmd.Output()
	if err != nil {
		debug["version"] = err.Error()
		return debug
	}

	lines := bytes.SplitN(output, []byte("\n"), 2)
	if len(lines) > 0 {
		debug["version"] = string(lines[0])
	} else {
		debug["version"] = "Unable to determine QPDF version"
	}

	return debug
}

// Split splits a given PDF file.
func (engine *QPdf) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Split",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	var args []string
	outputPath := fmt.Sprintf("%s/%s", outputDirPath, filepath.Base(inputPath))

	switch mode.Mode {
	case gotenberg.SplitModePages:
		if !mode.Unify {
			err := fmt.Errorf("split PDFs using mode '%s' without unify with QPDF: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		args = append(args, inputPath)
		args = append(args, engine.globalArgs...)
		args = append(args, "--pages", ".", mode.Span)
		args = append(args, "--", outputPath)
	default:
		err := fmt.Errorf("split PDFs using mode '%s' with QPDF: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("split PDFs with QPDF: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return []string{outputPath}, nil
}

// Merge combines multiple PDFs into a single PDF.
func (engine *QPdf) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Merge",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	args := make([]string, 0, 4+len(engine.globalArgs)+len(inputPaths))
	args = append(args, "--empty")
	args = append(args, engine.globalArgs...)
	args = append(args, "--pages")
	args = append(args, inputPaths...)
	args = append(args, "--", outputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err == nil {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	err = fmt.Errorf("merge PDFs with QPDF: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Flatten merges annotation appearances with page content, deleting the
// original annotations.
func (engine *QPdf) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Flatten",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	args := make([]string, 0, 4+len(engine.globalArgs))
	args = append(args, inputPath)
	args = append(args, "--generate-appearances")
	args = append(args, "--flatten-annotations=all")
	args = append(args, "--replace-input")
	args = append(args, engine.globalArgs...)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err == nil {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	err = fmt.Errorf("flatten PDFs with QPDF: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Convert is not available in this implementation.
func (engine *QPdf) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Convert",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("convert PDF to '%+v' with QPDF: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadMetadata is not available in this implementation.
func (engine *QPdf) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.ReadMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("read PDF metadata with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// WriteMetadata is not available in this implementation.
func (engine *QPdf) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.WriteMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("write PDF metadata with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// PageCount is not available in this implementation.
func (engine *QPdf) PageCount(ctx context.Context, logger *slog.Logger, inputPath string) (int, error) {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.PageCount",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("page count with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return 0, err
}

// WriteBookmarks is not available in this implementation.
func (engine *QPdf) WriteBookmarks(ctx context.Context, logger *slog.Logger, inputPath string, bookmarks []gotenberg.Bookmark) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.WriteBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("write PDF bookmarks with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadBookmarks is not available in this implementation.
func (engine *QPdf) ReadBookmarks(ctx context.Context, logger *slog.Logger, inputPath string) ([]gotenberg.Bookmark, error) {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.ReadBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("read PDF bookmarks with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// Encrypt adds password protection to a PDF file using QPDF.
func (engine *QPdf) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Encrypt",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	if userPassword == "" {
		err := errors.New("user password cannot be empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if ownerPassword == "" {
		ownerPassword = userPassword
	}

	args := make([]string, 0, 7+len(engine.globalArgs))
	args = append(args, inputPath)
	args = append(args, engine.globalArgs...)
	args = append(args, "--replace-input")
	args = append(args, "--encrypt", userPassword, ownerPassword, "256", "--")

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("encrypt PDF with QPDF: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// EmbedFiles is not available in this implementation.
func (engine *QPdf) EmbedFiles(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.EmbedFiles",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("embed files with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Watermark is not available in this implementation.
func (engine *QPdf) Watermark(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Watermark",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("watermark PDF with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Stamp is not available in this implementation.
func (engine *QPdf) Stamp(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Stamp",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("stamp PDF with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Rotate is not available in this implementation.
func (engine *QPdf) Rotate(ctx context.Context, logger *slog.Logger, inputPath string, angle int, pages string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Rotate",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("rotate PDF with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

var (
	_ gotenberg.Module      = (*QPdf)(nil)
	_ gotenberg.Provisioner = (*QPdf)(nil)
	_ gotenberg.Validator   = (*QPdf)(nil)
	_ gotenberg.Debuggable  = (*QPdf)(nil)
	_ gotenberg.PdfEngine   = (*QPdf)(nil)
)
