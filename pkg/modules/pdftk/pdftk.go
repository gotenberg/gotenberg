package pdftk

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
	gotenberg.MustRegisterModule(new(PdfTk))
}

// PdfTk abstracts the CLI tool PDFtk and implements the [gotenberg.PdfEngine]
// interface.
type PdfTk struct {
	binPath string
}

// Descriptor returns a [PdfTk]'s module descriptor.
func (engine *PdfTk) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "pdftk",
		New: func() gotenberg.Module { return new(PdfTk) },
	}
}

// Provision sets the module properties.
func (engine *PdfTk) Provision(ctx *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("PDFTK_BIN_PATH")
	if !ok {
		return errors.New("PDFTK_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine *PdfTk) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("PDFtk binary path does not exist: %w", err)
	}

	return nil
}

// Debug returns additional debug data.
func (engine *PdfTk) Debug() map[string]any {
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
		debug["version"] = "Unable to determine PDFtk version"
	}

	return debug
}

// Split splits a given PDF file.
func (engine *PdfTk) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	ctx, span := gotenberg.Tracer().Start(ctx, "pdftk.Split",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	var args []string
	outputPath := fmt.Sprintf("%s/%s", outputDirPath, filepath.Base(inputPath))

	switch mode.Mode {
	case gotenberg.SplitModePages:
		if !mode.Unify {
			err := fmt.Errorf("split PDFs using mode '%s' without unify with PDFtk: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		args = append(args, inputPath, "cat", mode.Span, "output", outputPath)
	default:
		err := fmt.Errorf("split PDFs using mode '%s' with PDFtk: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
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
		err = fmt.Errorf("split PDFs with PDFtk: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return []string{outputPath}, nil
}

// Merge combines multiple PDFs into a single PDF.
func (engine *PdfTk) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "pdftk.Merge",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	args := make([]string, 0, 3+len(inputPaths))
	args = append(args, inputPaths...)
	args = append(args, "cat", "output", outputPath)

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

	err = fmt.Errorf("merge PDFs with PDFtk: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Flatten is not available in this implementation.
func (engine *PdfTk) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.Flatten",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("flatten PDF with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Convert is not available in this implementation.
func (engine *PdfTk) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.Convert",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("convert PDF to '%+v' with PDFtk: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadMetadata is not available in this implementation.
func (engine *PdfTk) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.ReadMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("read PDF metadata with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// WriteMetadata is not available in this implementation.
func (engine *PdfTk) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.WriteMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("write PDF metadata with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// PageCount is not available in this implementation.
func (engine *PdfTk) PageCount(ctx context.Context, logger *slog.Logger, inputPath string) (int, error) {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.PageCount",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("page count with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return 0, err
}

// WriteBookmarks is not available in this implementation.
func (engine *PdfTk) WriteBookmarks(ctx context.Context, logger *slog.Logger, inputPath string, bookmarks []gotenberg.Bookmark) error {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.WriteBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("write PDF bookmarks with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadBookmarks is not available in this implementation.
func (engine *PdfTk) ReadBookmarks(ctx context.Context, logger *slog.Logger, inputPath string) ([]gotenberg.Bookmark, error) {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.ReadBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("read PDF bookmarks with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// Encrypt adds password protection to a PDF file using PDFtk.
func (engine *PdfTk) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "pdftk.Encrypt",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	if userPassword == "" {
		err := errors.New("user password cannot be empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if ownerPassword == userPassword || ownerPassword == "" {
		err := gotenberg.NewPdfEngineInvalidArgs("pdftk", "both 'userPassword' and 'ownerPassword' must be provided and different. Consider switching to another PDF engine if this behavior does not work with your workflow")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Create a temp output file in the same directory.
	tmpPath := inputPath + ".tmp"

	args := make([]string, 0, 8)
	args = append(args, inputPath)
	args = append(args, "output", tmpPath)
	args = append(args, "encrypt_128bit")
	args = append(args, "user_pw", userPassword)
	args = append(args, "owner_pw", ownerPassword)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("encrypt PDF with PDFtk: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err = os.Rename(tmpPath, inputPath)
	if err != nil {
		err = fmt.Errorf("rename temporary output file with input file: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// EmbedFiles is not available in this implementation.
func (engine *PdfTk) EmbedFiles(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "pdftk.EmbedFiles",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	err := fmt.Errorf("embed files with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Watermark applies a watermark (behind page content) to a PDF file using PDFtk.
// Only PDF source is supported.
//
//nolint:dupl
func (engine *PdfTk) Watermark(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "pdftk.Watermark",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	if stamp.Source != gotenberg.StampSourcePDF {
		err := fmt.Errorf("watermark PDF with PDFtk: %w", gotenberg.ErrPdfStampSourceNotSupported)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	tmpPath := inputPath + ".tmp"

	args := []string{inputPath, "background", stamp.Expression, "output", tmpPath}

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("watermark PDF with PDFtk: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err = os.Rename(tmpPath, inputPath)
	if err != nil {
		err = fmt.Errorf("rename temporary output file with input file: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Stamp applies a stamp (on top of page content) to a PDF file using PDFtk.
// Only PDF source is supported.
//
//nolint:dupl
func (engine *PdfTk) Stamp(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "pdftk.Stamp",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	if stamp.Source != gotenberg.StampSourcePDF {
		err := fmt.Errorf("stamp PDF with PDFtk: %w", gotenberg.ErrPdfStampSourceNotSupported)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	tmpPath := inputPath + ".tmp"

	args := []string{inputPath, "stamp", stamp.Expression, "output", tmpPath}

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("stamp PDF with PDFtk: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err = os.Rename(tmpPath, inputPath)
	if err != nil {
		err = fmt.Errorf("rename temporary output file with input file: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Rotate rotates all pages of a PDF file by the given angle using PDFtk.
// Page-specific rotation is not supported; if pages is non-empty,
// ErrPdfEngineMethodNotSupported is returned.
func (engine *PdfTk) Rotate(ctx context.Context, logger *slog.Logger, inputPath string, angle int, pages string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "pdftk.Rotate",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress("pdftk")),
	)
	defer span.End()

	if pages != "" {
		err := fmt.Errorf("rotate PDF with PDFtk (page-specific rotation): %w", gotenberg.ErrPdfEngineMethodNotSupported)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	var direction string
	switch angle {
	case 90:
		direction = "east"
	case 180:
		direction = "south"
	case 270:
		direction = "west"
	default:
		err := fmt.Errorf("rotate PDF with PDFtk: %w", gotenberg.ErrPdfRotateAngleNotSupported)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	tmpPath := inputPath + ".tmp"
	args := []string{inputPath, "cat", "1-end" + direction, "output", tmpPath}

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("rotate PDF with PDFtk: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err = os.Rename(tmpPath, inputPath)
	if err != nil {
		err = fmt.Errorf("rename temporary output file with input file: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PdfTk)(nil)
	_ gotenberg.Provisioner = (*PdfTk)(nil)
	_ gotenberg.Validator   = (*PdfTk)(nil)
	_ gotenberg.Debuggable  = (*PdfTk)(nil)
	_ gotenberg.PdfEngine   = (*PdfTk)(nil)
)
