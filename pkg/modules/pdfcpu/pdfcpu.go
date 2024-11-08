package pdfcpu

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"

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

// Merge combines multiple PDFs into a single PDF.
func (engine *PdfCpu) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var args []string
	args = append(args, "merge", outputPath)
	args = append(args, inputPaths...)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with pdfcpu: %w", err)
}

// Convert is not available in this implementation.
func (engine *PdfCpu) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with pdfcpu: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

// ReadMetadata is not available in this implementation.
func (engine *PdfCpu) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("read PDF metadata with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// WriteMetadata is not available in this implementation.
func (engine *PdfCpu) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	return fmt.Errorf("write PDF metadata with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PdfCpu)(nil)
	_ gotenberg.Provisioner = (*PdfCpu)(nil)
	_ gotenberg.Validator   = (*PdfCpu)(nil)
	_ gotenberg.PdfEngine   = (*PdfCpu)(nil)
)
