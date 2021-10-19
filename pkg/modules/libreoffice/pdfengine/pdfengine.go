package pdfengine

import (
	"context"
	"fmt"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/unoconv"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(UnoconvPDFEngine{})
}

// UnoconvPDFEngine abstracts the CLI tool unoconv and implements the
// gotenberg.PDFEngine interface.
type UnoconvPDFEngine struct {
	unoconv unoconv.API
}

// Descriptor returns a UnoconvPDFEngine's module descriptor.
func (UnoconvPDFEngine) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "unoconv-pdfengine",
		New: func() gotenberg.Module { return new(UnoconvPDFEngine) },
	}
}

// Provision sets the module properties.
func (engine *UnoconvPDFEngine) Provision(ctx *gotenberg.Context) error {
	provider, err := ctx.Module(new(unoconv.Provider))
	if err != nil {
		return fmt.Errorf("get unoconv provider: %w", err)
	}

	uno, err := provider.(unoconv.Provider).Unoconv()
	if err != nil {
		return fmt.Errorf("get unoconv API: %w", err)
	}

	engine.unoconv = uno

	return nil
}

// Merge is not available for this PDF engine.
func (engine UnoconvPDFEngine) Merge(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
	return fmt.Errorf("merge PDFs with unoconv: %w", gotenberg.ErrPDFEngineMethodNotAvailable)
}

// Convert converts the given PDF to a specific PDF format. Currently, only the
// PDF/A-1 format is available. If another PDF format is requested, it returns
// a gotenberg.ErrPDFFormatNotAvailable error.
func (engine UnoconvPDFEngine) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	if format != gotenberg.FormatPDFA1a {
		return fmt.Errorf("convert PDF to '%s' with unoconv: %w", format, gotenberg.ErrPDFFormatNotAvailable)
	}

	err := engine.unoconv.PDF(ctx, logger, inputPath, outputPath, unoconv.Options{
		PDFArchive: true,
	})

	if err == nil {
		return nil
	}

	return fmt.Errorf("convert PDF to '%s' with unoconv: %w", format, err)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*UnoconvPDFEngine)(nil)
	_ gotenberg.Provisioner = (*UnoconvPDFEngine)(nil)
	_ gotenberg.PDFEngine   = (*UnoconvPDFEngine)(nil)
)
