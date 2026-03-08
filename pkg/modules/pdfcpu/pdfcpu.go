package pdfcpu

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(PdfCpu))
}

// PdfCpu abstracts the CLI tool pdfcpu and implements the
// [gotenberg.PdfEngine] interface.
type PdfCpu struct {
	binPath string
}

// Descriptor returns a [PdfCpu]'s module descriptor.
func (engine *PdfCpu) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "pdfcpu",
		New: func() gotenberg.Module { return new(PdfCpu) },
	}
}

// Provision sets the engine properties.
func (engine *PdfCpu) Provision(ctx *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("PDFCPU_BIN_PATH")
	if !ok {
		return errors.New("PDFCPU_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine *PdfCpu) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("pdfcpu binary path does not exist: %w", err)
	}

	return nil
}

// Debug returns additional debug data.
func (engine *PdfCpu) Debug() map[string]any {
	debug := make(map[string]any)

	cmd := exec.Command(engine.binPath, "version") //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	output, err := cmd.Output()
	if err != nil {
		debug["version"] = err.Error()
		return debug
	}

	debug["version"] = "Unable to determine pdfcpu version"

	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if after, ok := strings.CutPrefix(line, "pdfcpu:"); ok {
			debug["version"] = strings.TrimSpace(after)
			break
		}
	}

	return debug
}

// Merge combines multiple PDFs into a single PDF.
func (engine *PdfCpu) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "PdfCpu.Merge", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	args := make([]string, 0, 2+len(inputPaths))
	args = append(args, "merge", outputPath)
	args = append(args, inputPaths...)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err == nil {
		return nil
	}

	err = fmt.Errorf("merge PDFs with pdfcpu: %w", err)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

// Split splits a given PDF file.
func (engine *PdfCpu) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	ctx, span := gotenberg.Tracer().Start(ctx, "PdfCpu.Split", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var args []string

	switch mode.Mode {
	case gotenberg.SplitModeIntervals:
		args = append(args, "split", "-mode", "span", inputPath, outputDirPath, mode.Span)
	case gotenberg.SplitModePages:
		if mode.Unify {
			outputPath := fmt.Sprintf("%s/%s", outputDirPath, filepath.Base(inputPath))
			args = append(args, "trim", "-pages", mode.Span, inputPath, outputPath)
			break
		}
		args = append(args, "extract", "-mode", "page", "-pages", mode.Span, inputPath, outputDirPath)
	default:
		err := fmt.Errorf("split PDFs using mode '%s' with pdfcpu: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
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
		err = fmt.Errorf("split PDFs with pdfcpu: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var outputPaths []string
	err = filepath.Walk(outputDirPath, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			return pathErr
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
			outputPaths = append(outputPaths, path)
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("walk directory to find resulting PDFs from split with pdfcpu: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	sort.Sort(digitSuffixSort(outputPaths))

	return outputPaths, nil
}

// Flatten is not available in this implementation.
func (engine *PdfCpu) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "PdfCpu.Flatten", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("flatten PDF with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// Convert is not available in this implementation.
func (engine *PdfCpu) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "PdfCpu.Convert", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("convert PDF to '%+v' with pdfcpu: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// ReadMetadata is not available in this implementation.
func (engine *PdfCpu) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	_, span := gotenberg.Tracer().Start(ctx, "PdfCpu.ReadMetadata", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("read PDF metadata with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return nil, err
}

// WriteMetadata is not available in this implementation.
func (engine *PdfCpu) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "PdfCpu.WriteMetadata", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("write PDF metadata with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// AddAttachments adds attachments into a PDF. All files are attached as
// file attachments without modifying the main PDF content.
func (engine *PdfCpu) AddAttachments(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "PdfCpu.AddAttachments", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	if len(filePaths) == 0 {
		return nil
	}

	logger.DebugContext(ctx, fmt.Sprintf("attaching %d file(s) to %s: %v", len(filePaths), inputPath, filePaths))

	args := make([]string, 0, 3+len(filePaths))
	args = append(args, "attachments", "add", inputPath)
	args = append(args, filePaths...)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command for attaching files: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("attach files with pdfcpu: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// Encrypt adds password protection to a PDF file using pdfcpu.
func (engine *PdfCpu) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "PdfCpu.Encrypt", trace.WithSpanKind(trace.SpanKindInternal))
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

	args := make([]string, 0, 11)
	args = append(args, "encrypt")
	args = append(args, "-mode", "aes")
	args = append(args, "-upw", userPassword)
	args = append(args, "-opw", ownerPassword)
	args = append(args, "-perm", "all")
	args = append(args, inputPath, inputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("encrypt PDF with pdfcpu: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PdfCpu)(nil)
	_ gotenberg.Provisioner = (*PdfCpu)(nil)
	_ gotenberg.Validator   = (*PdfCpu)(nil)
	_ gotenberg.Debuggable  = (*PdfCpu)(nil)
	_ gotenberg.PdfEngine   = (*PdfCpu)(nil)
)
