package pdfcpu

import (
	"context"
	"fmt"

	pdfcpuAPI "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	pdfcpuConfig "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(PdfCpu))
}

// PdfCpu abstracts the pdfcpu library and implements the [gotenberg.PdfEngine]
// interface.
type PdfCpu struct {
	conf *pdfcpuConfig.Configuration
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
	pdfcpuConfig.ConfigPath = "disable"
	pdfcpuLog.DisableLoggers()
	engine.conf = pdfcpuConfig.NewDefaultConfiguration()

	return nil
}

// Merge combines multiple PDFs into a single PDF.
func (engine *PdfCpu) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	err := pdfcpuAPI.MergeCreateFile(inputPaths, outputPath, false, engine.conf)
	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with PDFcpu: %w", err)
}

// Convert is not available in this implementation.
func (engine *PdfCpu) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with PDFcpu: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PdfCpu)(nil)
	_ gotenberg.Provisioner = (*PdfCpu)(nil)
	_ gotenberg.PdfEngine   = (*PdfCpu)(nil)
)
