package gotenberg

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

var (
	// ErrPdfEngineMethodNotSupported is returned when a specific method of the
	// PdfEngine interface is not supported by its current implementation.
	ErrPdfEngineMethodNotSupported = errors.New("method not supported")

	// ErrPdfFormatNotSupported is returned when the Convert method of the
	// PdfEngine interface does not support a requested PDF format conversion.
	ErrPdfFormatNotSupported = errors.New("PDF format not supported")
)

const (
	// PdfA1a represents the PDF/A-1a format.
	PdfA1a string = "PDF/A-1a"

	// PdfA1b represents the PDF/A-1b format.
	PdfA1b string = "PDF/A-1b"

	// PdfA2a represents the PDF/A-2a format.
	PdfA2a string = "PDF/A-2a"

	// PdfA2b represents the PDF/A-2b format.
	PdfA2b string = "PDF/A-2b"

	// PdfA2u represents the PDF/A-2u format.
	PdfA2u string = "PDF/A-2u"

	// PdfA3a represents the PDF/A-3a format.
	PdfA3a string = "PDF/A-3a"

	// PdfA3b represents the PDF/A-3b format.
	PdfA3b string = "PDF/A-3b"

	// PdfA3u represents the PDF/A-3u format.
	PdfA3u string = "PDF/A-3u"
)

// PdfFormats specifies the target formats for a PDF conversion.
type PdfFormats struct {
	// PdfA denotes the PDF/A standard format (e.g., PDF/A-1a).
	PdfA string

	// PdfUa indicates whether the PDF should comply
	// with the PDF/UA (Universal Accessibility) standard.
	PdfUa bool
}

// PdfEngine provides an interface for operations on PDFs. Implementations
// can utilize various tools like PDFtk, or implement functionality directly in
// Go.
type PdfEngine interface {
	// Merge combines multiple PDFs into a single PDF. The resulting page order
	// is determined by the order of files provided in inputPaths.
	Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error

	// Convert transforms a given PDF to the specified formats defined in
	// PdfFormats. If no format, it does nothing.
	Convert(ctx context.Context, logger *zap.Logger, formats PdfFormats, inputPath, outputPath string) error
}

// PdfEngineProvider offers an interface to instantiate a [PdfEngine].
// This is used to decouple the creation of a [PdfEngine] from its consumers.
//
// Example:
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.PdfEngineProvider))
//		engine, _ := provider.(gotenberg.PdfEngineProvider).PdfEngine()
//	}
type PdfEngineProvider interface {
	// PdfEngine returns an instance of the [PdfEngine] interface for PDF operations.
	PdfEngine() (PdfEngine, error)
}
