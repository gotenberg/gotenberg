package gotenberg

import (
	"context"
	"errors"
	"fmt"

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

	// ErrPdfEncryptionNotSupported is returned when encryption
	// is not supported by the PDF engine.
	ErrPdfEncryptionNotSupported = errors.New("encryption not supported")

	// ErrPdfStampSourceNotSupported is returned when a stamp source type
	// is not supported by the PDF engine.
	ErrPdfStampSourceNotSupported = errors.New("stamp source not supported")

	// ErrPdfRotateAngleNotSupported is returned when the rotation angle is
	// not supported.
	ErrPdfRotateAngleNotSupported = errors.New("rotation angle not supported")
)

// PdfEngineInvalidArgsError represents an error returned by a PDF engine when
// invalid arguments are provided. It includes the name of the engine and a
// detailed message describing the issue.
type PdfEngineInvalidArgsError struct {
	engine string
	msg    string
}

// Error implements the error interface.
func (e *PdfEngineInvalidArgsError) Error() string {
	return fmt.Sprintf("%s: %s", e.engine, e.msg)
}

// NewPdfEngineInvalidArgs creates a new PdfEngineInvalidArgsError with the
// given engine name and message.
func NewPdfEngineInvalidArgs(engine, msg string) error {
	return &PdfEngineInvalidArgsError{engine, msg}
}

const (
	// StampSourceText represents a text-based stamp source.
	StampSourceText string = "text"

	// StampSourceImage represents an image-based stamp source.
	StampSourceImage string = "image"

	// StampSourcePDF represents a PDF-based stamp source.
	StampSourcePDF string = "pdf"
)

// Stamp gathers the data required to apply a watermark or stamp to a PDF.
type Stamp struct {
	// Source is one of "text", "image", or "pdf".
	Source string

	// Expression is the text content (for text source) or file path (for
	// image/pdf source).
	Expression string

	// Pages is the optional page range to apply the stamp to.
	Pages string

	// Options holds engine-specific styling options.
	Options map[string]string
}

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

// Bookmark represents a node in the PDF document's outline
// (table of contents).
type Bookmark struct {
	Title    string     `json:"title"`
	Page     int        `json:"page"`
	Children []Bookmark `json:"children,omitempty"`
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
	// forms as well as forms share a relationship with annotations. Note that
	// this operation is irreversible.
	Flatten(ctx context.Context, logger *zap.Logger, inputPath string) error

	// Convert transforms a given PDF to the specified formats defined in
	// PdfFormats. If no format, it does nothing.
	Convert(ctx context.Context, logger *zap.Logger, formats PdfFormats, inputPath, outputPath string) error

	// ReadMetadata extracts the metadata of a given PDF file.
	ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]any, error)

	// PageCount returns the number of pages in a PDF file.
	PageCount(ctx context.Context, logger *zap.Logger, inputPath string) (int, error)

	// WriteMetadata writes the metadata into a given PDF file.
	WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]any, inputPath string) error

	// ReadBookmarks reads the document outline (bookmarks) of a PDF file.
	ReadBookmarks(ctx context.Context, logger *zap.Logger, inputPath string) ([]Bookmark, error)

	// WriteBookmarks adds a document outline (bookmarks) to a PDF file.
	// The bookmarks parameter represents the hierarchical tree of the outline.
	WriteBookmarks(ctx context.Context, logger *zap.Logger, inputPath string, bookmarks []Bookmark) error

	// Encrypt adds password protection to a PDF file.
	// The userPassword is required to open the document.
	// The ownerPassword provides full access to the document.
	// If the ownerPassword is empty, it defaults to the userPassword.
	Encrypt(ctx context.Context, logger *zap.Logger, inputPath, userPassword, ownerPassword string) error

	// EmbedFiles embeds files into a PDF. All files are embedded as file attachments
	// without modifying the main PDF content.
	// TODO: attachments instead? Rename the route?
	EmbedFiles(ctx context.Context, logger *zap.Logger, filePaths []string, inputPath string) error

	// Watermark applies a watermark (behind page content) to a PDF file.
	Watermark(ctx context.Context, logger *zap.Logger, inputPath string, stamp Stamp) error

	// Stamp applies a stamp (on top of page content) to a PDF file.
	Stamp(ctx context.Context, logger *zap.Logger, inputPath string, stamp Stamp) error

	// Rotate rotates pages of a PDF file by the given angle (90, 180, 270).
	// If pages is empty, all pages are rotated.
	Rotate(ctx context.Context, logger *zap.Logger, inputPath string, angle int, pages string) error
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
