package pdfengine

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func init() {
	gotenberg.MustRegisterModule(new(LibreOfficePdfEngine))
}

// LibreOfficePdfEngine interacts with the LibreOffice (Universal Network Objects) API
// and implements the [gotenberg.PdfEngine] interface.
type LibreOfficePdfEngine struct {
	libreoffice api.Uno
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

	libreoffice, err := provider.(api.Provider).LibreOffice()
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno: %w", err)
	}

	engine.libreoffice = libreoffice

	return nil
}

// Merge is not available in this implementation.
func (engine *LibreOfficePdfEngine) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return fmt.Errorf("merge PDFs with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Convert converts the given PDF to a specific PDF format. Currently, only the
// PDF/A-1b, PDF/A-2b, PDF/A-3b and PDF/UA formats are available. If another
// PDF format is requested, it returns a [gotenberg.ErrPdfFormatNotSupported]
// error.
func (engine *LibreOfficePdfEngine) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	opts := api.DefaultOptions()
	opts.PdfFormats = formats
	err := engine.libreoffice.Pdf(ctx, logger, inputPath, outputPath, opts)

	if err == nil {
		return nil
	}

	if errors.Is(err, api.ErrInvalidPdfFormats) {
		return fmt.Errorf("convert PDF to '%+v' with LibreOffice: %w", formats, gotenberg.ErrPdfFormatNotSupported)
	}

	return fmt.Errorf("convert PDF to '%+v' with LibreOffice: %w", formats, err)
}

// Optimize is not available in this implementation.
func (engine *LibreOfficePdfEngine) Optimize(ctx context.Context, logger *zap.Logger, options gotenberg.OptimizeOptions, inputPath, outputPath string) error {
	return fmt.Errorf("optimize PDF with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// ReadMetadata is not available in this implementation.
func (engine *LibreOfficePdfEngine) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("read PDF metadata with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// WriteMetadata is not available in this implementation.
func (engine *LibreOfficePdfEngine) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	return fmt.Errorf("write PDF metadata with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.Provisioner = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.PdfEngine   = (*LibreOfficePdfEngine)(nil)
)
