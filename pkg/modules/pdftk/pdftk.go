package pdftk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"go.uber.org/zap"

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
func (engine *PdfTk) Debug() map[string]interface{} {
	debug := make(map[string]interface{})

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
func (engine *PdfTk) Split(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	var args []string
	outputPath := fmt.Sprintf("%s/%s", outputDirPath, filepath.Base(inputPath))

	switch mode.Mode {
	case gotenberg.SplitModePages:
		if !mode.Unify {
			return nil, fmt.Errorf("split PDFs using mode '%s' without unify with PDFtk: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
		}
		args = append(args, inputPath, "cat", mode.Span, "output", outputPath)
	default:
		return nil, fmt.Errorf("split PDFs using mode '%s' with PDFtk: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
	}

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return nil, fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return nil, fmt.Errorf("split PDFs with PDFtk: %w", err)
	}

	return []string{outputPath}, nil
}

// Merge combines multiple PDFs into a single PDF.
func (engine *PdfTk) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var args []string
	args = append(args, inputPaths...)
	args = append(args, "cat", "output", outputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with PDFtk: %w", err)
}

// Flatten is not available in this implementation.
func (engine *PdfTk) Flatten(ctx context.Context, logger *zap.Logger, inputPath string) error {
	return fmt.Errorf("flatten PDF with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Convert is not available in this implementation.
func (engine *PdfTk) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with PDFtk: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

// ReadMetadata is not available in this implementation.
func (engine *PdfTk) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("read PDF metadata with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// WriteMetadata is not available in this implementation.
func (engine *PdfTk) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	return fmt.Errorf("write PDF metadata with PDFtk: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PdfTk)(nil)
	_ gotenberg.Provisioner = (*PdfTk)(nil)
	_ gotenberg.Validator   = (*PdfTk)(nil)
	_ gotenberg.Debuggable  = (*PdfTk)(nil)
	_ gotenberg.PdfEngine   = (*PdfTk)(nil)
)
