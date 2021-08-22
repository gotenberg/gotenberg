package pdftk

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(PDFtk{})
}

// PDFtk abstracts the CLI tool PDFtk and implements the gotenberg.PDFEngine
// interface.
type PDFtk struct {
	binPath string
}

// Descriptor returns a PDFtk's module descriptor.
func (engine PDFtk) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "pdftk",
		New: func() gotenberg.Module { return new(PDFtk) },
	}
}

// Provision sets the modules properties. It returns an error if the
// environment variable PDFTK_BIN_PATH is not set.
func (engine *PDFtk) Provision(_ *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("PDFTK_BIN_PATH")
	if !ok {
		return errors.New("PDFTK_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine PDFtk) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("PDFtk binary path does not exist: %w", err)
	}

	return nil
}

// Merge merges the given PDFs into a unique PDF.
func (engine PDFtk) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var args []string
	args = append(args, inputPaths...)
	args = append(args, "cat", "output", outputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	err = cmd.Exec()
	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with PDFtk: %w", err)
}

// Convert is not available for this PDF engine.
func (engine PDFtk) Convert(_ context.Context, _ *zap.Logger, format, _, _ string) error {
	return fmt.Errorf("convert PDF to '%s' with PDFtk: %w", format, gotenberg.ErrPDFEngineMethodNotAvailable)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PDFtk)(nil)
	_ gotenberg.Provisioner = (*PDFtk)(nil)
	_ gotenberg.Validator   = (*PDFtk)(nil)
	_ gotenberg.PDFEngine   = (*PDFtk)(nil)
)
