package gotenberg

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

var (
	// ErrPDFEngineMethodNotAvailable happens if a PDFEngine method is not
	// available in the implementation.
	ErrPDFEngineMethodNotAvailable = errors.New("method not available")

	// ErrPDFFormatNotAvailable happens if a PDFEngine Convert's method does
	// not handle a specific format.
	ErrPDFFormatNotAvailable = errors.New("PDF format not available")
)

const (
	FormatPDFA1a string = "PDF/A-1a"
	FormatPDFA1b string = "PDF/A-1b"
	FormatPDFA2a string = "PDF/A-2a"
	FormatPDFA2b string = "PDF/A-2b"
	FormatPDFA2u string = "PDF/A-2u"
	FormatPDFA3a string = "PDF/A-3a"
	FormatPDFA3b string = "PDF/A-3b"
	FormatPDFA3u string = "PDF/A-3u"
)

// PDFEngine is a module interface which exposes methods for manipulating one
// or more PDFs. Implementations may abstract powerful tools like PDFtk, or
// fulfill those methods contracts in Golang directly.
type PDFEngine interface {
	// Merge merges the given PDFs into a unique PDF. The pages' order reflects
	// order of the given files.
	Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error

	// Convert converts the given PDF to a specific PDF format.
	Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error
}

// PDFEngineProvider is a module interface which exposes a method for creating a
// PDFEngine for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _  := ctx.Module(new(gotenberg.PDFEngineProvider))
//		pdfengines, _ := provider.(gotenberg.PDFEngineProvider).PDFEngine()
//	}
type PDFEngineProvider interface {
	PDFEngine() (PDFEngine, error)
}
