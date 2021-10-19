package pdfcpu

import (
	"context"
	"fmt"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	pdfcpuAPI "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	pdfcpuConfig "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(PDFcpu{})
}

// PDFcpu is a module which wraps the https://github.com/pdfcpu/pdfcpu library
// and implements the gotenberg.PDFEngine interface.
type PDFcpu struct {
	conf *pdfcpuConfig.Configuration
}

// Descriptor returns a PDFcpu's module descriptor.
func (PDFcpu) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "pdfcpu",
		New: func() gotenberg.Module { return new(PDFcpu) },
	}
}

// Provision sets the engine properties.
func (engine *PDFcpu) Provision(_ *gotenberg.Context) error {
	pdfcpuConfig.ConfigPath = "disable"
	pdfcpuLog.DisableLoggers()

	engine.conf = pdfcpuConfig.NewDefaultConfiguration()

	return nil
}

// Merge merges the given PDFs into a unique PDF.
func (engine PDFcpu) Merge(_ context.Context, _ *zap.Logger, inputPaths []string, outputPath string) error {
	err := pdfcpuAPI.MergeCreateFile(inputPaths, outputPath, engine.conf)
	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with PDFcpu: %w", err)
}

// Convert is not available for this PDF engine.
func (engine PDFcpu) Convert(_ context.Context, _ *zap.Logger, format, _, _ string) error {
	return fmt.Errorf("convert PDF to '%s' with PDFcpu: %w", format, gotenberg.ErrPDFEngineMethodNotAvailable)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PDFcpu)(nil)
	_ gotenberg.Provisioner = (*PDFcpu)(nil)
	_ gotenberg.PDFEngine   = (*PDFcpu)(nil)
)
