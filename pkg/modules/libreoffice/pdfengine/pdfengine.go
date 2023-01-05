package pdfengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/uno"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(UNO{})
}

// UNO interacts with the UNO (Universal Network Objects) API and implements
// the gotenberg.PDFEngine interface.
type UNO struct {
	unoAPI uno.API
}

// Descriptor returns a UNO's module descriptor.
func (UNO) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "uno-pdfengine",
		New: func() gotenberg.Module { return new(UNO) },
	}
}

// Provision sets the module properties.
func (engine *UNO) Provision(ctx *gotenberg.Context) error {
	provider, err := ctx.Module(new(uno.Provider))
	if err != nil {
		return fmt.Errorf("get unoconv provider: %w", err)
	}

	unoAPI, err := provider.(uno.Provider).UNO()
	if err != nil {
		return fmt.Errorf("get unoconv API: %w", err)
	}

	engine.unoAPI = unoAPI

	return nil
}

// Merge is not available for this PDF engine.
func (engine UNO) Merge(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
	return fmt.Errorf("merge PDFs with unoconv: %w", gotenberg.ErrPDFEngineMethodNotAvailable)
}

// Convert converts the given PDF to a specific PDF format. Currently, only the
// PDF/A-1a, PDF/A-2b and PDF/A-3b formats are available. If another PDF format
// is requested, it returns a gotenberg.ErrPDFFormatNotAvailable error.
func (engine UNO) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	err := engine.unoAPI.Convert(ctx, logger, inputPath, outputPath, uno.Options{
		PDFformat: format,
	})

	if err == nil {
		return nil
	}

	if errors.Is(err, uno.ErrInvalidPDFformat) {
		return fmt.Errorf("convert PDF to '%s' with unoconv: %w", format, gotenberg.ErrPDFFormatNotAvailable)
	}

	return fmt.Errorf("convert PDF to '%s' with unoconv: %w", format, err)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*UNO)(nil)
	_ gotenberg.Provisioner = (*UNO)(nil)
	_ gotenberg.PDFEngine   = (*UNO)(nil)
)
