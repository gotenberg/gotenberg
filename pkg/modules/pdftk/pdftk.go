package pdftk

import (
	"context"
	"errors"
	"fmt"
	"os"

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

// Provision sets the modules properties.
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
		return fmt.Errorf("PdfTk binary path does not exist: %w", err)
	}

	return nil
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

// Convert is not available in this implementation.
func (engine *PdfTk) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with PDFtk: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PdfTk)(nil)
	_ gotenberg.Provisioner = (*PdfTk)(nil)
	_ gotenberg.Validator   = (*PdfTk)(nil)
	_ gotenberg.PdfEngine   = (*PdfTk)(nil)
)
