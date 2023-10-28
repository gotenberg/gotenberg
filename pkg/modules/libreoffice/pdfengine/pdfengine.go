package pdfengine

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/api"
)

func init() {
	gotenberg.MustRegisterModule(new(LibreOfficePdfEngine))
}

// LibreOfficePdfEngine interacts with the LibreOffice (Universal Network Objects) API
// and implements the [gotenberg.PDFEngine] interface.
type LibreOfficePdfEngine struct {
	unoAPI api.Uno
}

// Descriptor returns a [LibreOfficePdfEngine]'s module descriptor.
func (engine *LibreOfficePdfEngine) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "libreoffice-pdfengine",
		New: func() gotenberg.Module { return new(LibreOfficePdfEngine) },
	}
}

// Provision sets the module properties.
func (engine *LibreOfficePdfEngine) Provision(ctx *gotenberg.Context) error {
	provider, err := ctx.Module(new(api.Provider))
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno provider: %w", err)
	}

	unoAPI, err := provider.(api.Provider).LibreOffice()
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno: %w", err)
	}

	engine.unoAPI = unoAPI

	return nil
}

// Merge is not available for this PDF engine.
func (engine *LibreOfficePdfEngine) Merge(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
	return fmt.Errorf("merge PDFs with LibreOffice: %w", gotenberg.ErrPDFEngineMethodNotAvailable)
}

// Convert converts the given PDF to a specific PDF format. Currently, only the
// PDF/A-1a, PDF/A-2b and PDF/A-3b formats are available. If another PDF format
// is requested, it returns a [gotenberg.ErrPDFFormatNotAvailable] error.
func (engine *LibreOfficePdfEngine) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	err := engine.unoAPI.Pdf(ctx, logger, inputPath, outputPath, api.Options{
		PdfFormat: format,
	})

	if err == nil {
		return nil
	}

	if errors.Is(err, api.ErrInvalidPdfFormat) {
		return fmt.Errorf("convert PDF to '%s' with LibreOffice: %w", format, gotenberg.ErrPDFFormatNotAvailable)
	}

	return fmt.Errorf("convert PDF to '%s' with unoconv: %w", format, err)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.Provisioner = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.PDFEngine   = (*LibreOfficePdfEngine)(nil)
)
