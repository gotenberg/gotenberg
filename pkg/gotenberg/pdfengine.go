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

	// ErrPdfSplitModeNotSupported is returned when the Split method of the
	// PdfEngine interface does not support a requested PDF split mode.
	ErrPdfSplitModeNotSupported = errors.New("split mode not supported")

	// ErrPdfFormatNotSupported is returned when the Convert method of the
	// PdfEngine interface does not support a requested PDF format conversion.
	ErrPdfFormatNotSupported = errors.New("PDF format not supported")

	// ErrPdfEngineMetadataValueNotSupported is returned when a metadata value
	// is not supported.
	ErrPdfEngineMetadataValueNotSupported = errors.New("metadata value not supported")
)

const (
	// SplitModeIntervals represents a mode where a PDF is split at specific
	// intervals.
	SplitModeIntervals string = "intervals"

	// SplitModePages represents a mode where a PDF is split at specific page
	// ranges.
	SplitModePages string = "pages"
)

// SplitMode gathers the data required to split a PDF into multiple parts.
type SplitMode struct {
	// Mode is either "intervals" or "pages".
	Mode string

	// Span is either the intervals or the page ranges to extract, depending on
	// the selected mode.
	Span string

	// Unify specifies whether to put extracted pages into a single file or as
	// many files as there are page ranges. Only works with "pages" mode.
	Unify bool
}

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
// can use various tools like PDFtk, or implement functionality directly in
// Go.
//
//nolint:dupl
type PdfEngine interface {
	// Merge combines multiple PDFs into a single PDF. The resulting page order
	// is determined by the order of files provided in inputPaths.
	Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error

	// Split splits a given PDF file.
	Split(ctx context.Context, logger *zap.Logger, mode SplitMode, inputPath, outputDirPath string) ([]string, error)

	// Flatten merges existing annotation appearances with page content,
	// effectively deleting the original annotations. This process can flatten
	// forms as well, as forms share a relationship with annotations. Note that
	// this operation is irreversible.
	Flatten(ctx context.Context, logger *zap.Logger, inputPath string) error

	// Convert transforms a given PDF to the specified formats defined in
	// PdfFormats. If no format, it does nothing.
	Convert(ctx context.Context, logger *zap.Logger, formats PdfFormats, inputPath, outputPath string) error

	// ReadMetadata extracts the metadata of a given PDF file.
	ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error)

	// WriteMetadata writes the metadata into a given PDF file.
	WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error
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
