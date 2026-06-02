package pdfengines

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

type multiPdfEngines struct {
	mergeEngines          []gotenberg.PdfEngine
	splitEngines          []gotenberg.PdfEngine
	flattenEngines        []gotenberg.PdfEngine
	convertEngines        []gotenberg.PdfEngine
	readMetadataEngines   []gotenberg.PdfEngine
	writeMetadataEngines  []gotenberg.PdfEngine
	passwordEngines       []gotenberg.PdfEngine
	embedEngines          []gotenberg.PdfEngine
	embedMetadataEngines  []gotenberg.PdfEngine
	readBookmarksEngines  []gotenberg.PdfEngine
	writeBookmarksEngines []gotenberg.PdfEngine
	watermarkEngines      []gotenberg.PdfEngine
	stampEngines          []gotenberg.PdfEngine
	rotateEngines         []gotenberg.PdfEngine
}

func newMultiPdfEngines(
	mergeEngines,
	splitEngines,
	flattenEngines,
	convertEngines,
	readMetadataEngines,
	writeMetadataEngines,
	passwordEngines,
	embedEngines,
	embedMetadataEngines,
	readBookmarksEngines,
	writeBookmarksEngines,
	watermarkEngines,
	stampEngines,
	rotateEngines []gotenberg.PdfEngine,
) *multiPdfEngines {
	return &multiPdfEngines{
		mergeEngines:          mergeEngines,
		splitEngines:          splitEngines,
		flattenEngines:        flattenEngines,
		convertEngines:        convertEngines,
		readMetadataEngines:   readMetadataEngines,
		writeMetadataEngines:  writeMetadataEngines,
		passwordEngines:       passwordEngines,
		embedEngines:          embedEngines,
		embedMetadataEngines:  embedMetadataEngines,
		readBookmarksEngines:  readBookmarksEngines,
		writeBookmarksEngines: writeBookmarksEngines,
		watermarkEngines:      watermarkEngines,
		stampEngines:          stampEngines,
		rotateEngines:         rotateEngines,
	}
}

// engineName returns the module ID of a PDF engine for telemetry, falling back
// to its type name when it does not expose a descriptor.
func engineName(engine gotenberg.PdfEngine) string {
	if module, ok := engine.(gotenberg.Module); ok {
		return module.Descriptor().ID
	}
	return fmt.Sprintf("%T", engine)
}

// runWithFallback runs op against each engine in order and returns the first
// success. It wraps the attempts in a pdfengines span, records the winning
// engine and attempt count, emits a pdf_engine.attempt_failed event for each
// failed engine, and joins all engine errors on total failure. A context
// cancellation marks the span as errored too. wrap applies the op-specific
// final error message.
func runWithFallback[T any](
	ctx context.Context,
	spanName string,
	engines []gotenberg.PdfEngine,
	op func(ctx context.Context, engine gotenberg.PdfEngine) (T, error),
	wrap func(err error) error,
) (T, error) {
	var zero T

	ctx, span := gotenberg.Tracer().Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	type attemptResult struct {
		value T
		err   error
	}

	var joined error
	for attempt, engine := range engines {
		resultChan := make(chan attemptResult, 1)

		go func(engine gotenberg.PdfEngine) {
			value, err := op(ctx, engine)
			resultChan <- attemptResult{value: value, err: err}
		}(engine)

		select {
		case result := <-resultChan:
			if result.err == nil {
				span.SetAttributes(
					attribute.String("gotenberg.pdf_engine.selected", engineName(engine)),
					attribute.Int("gotenberg.pdf_engine.attempts", attempt+1),
				)
				span.SetStatus(codes.Ok, "")
				return result.value, nil
			}

			joined = errors.Join(joined, result.err)
			span.AddEvent("pdf_engine.attempt_failed", trace.WithAttributes(
				attribute.String("engine", engineName(engine)),
			))
		case <-ctx.Done():
			err := ctx.Err()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return zero, err
		}
	}

	err := wrap(joined)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return zero, err
}

// runWithFallbackVoid adapts [runWithFallback] for operations that return no
// value beyond an error.
func runWithFallbackVoid(
	ctx context.Context,
	spanName string,
	engines []gotenberg.PdfEngine,
	op func(ctx context.Context, engine gotenberg.PdfEngine) error,
	wrap func(err error) error,
) error {
	_, err := runWithFallback(ctx, spanName, engines,
		func(ctx context.Context, engine gotenberg.PdfEngine) (struct{}, error) {
			return struct{}{}, op(ctx, engine)
		},
		wrap,
	)
	return err
}

// Merge combines multiple PDF files into a single document using the first
// available engine that supports PDF merging.
func (multi *multiPdfEngines) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	return runWithFallbackVoid(ctx, "pdfengines.Merge", multi.mergeEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.Merge(ctx, logger, inputPaths, outputPath)
		},
		func(err error) error { return fmt.Errorf("merge PDFs with multi PDF engines: %w", err) },
	)
}

// Split divides the PDF into separate pages using the first available engine
// that supports PDF splitting.
func (multi *multiPdfEngines) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	return runWithFallback(ctx, "pdfengines.Split", multi.splitEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) ([]string, error) {
			return engine.Split(ctx, logger, mode, inputPath, outputDirPath)
		},
		func(err error) error { return fmt.Errorf("split PDF with multi PDF engines: %w", err) },
	)
}

// Flatten merges existing annotation appearances with page content using the
// first available engine that supports flattening.
func (multi *multiPdfEngines) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	return runWithFallbackVoid(ctx, "pdfengines.Flatten", multi.flattenEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.Flatten(ctx, logger, inputPath)
		},
		func(err error) error { return fmt.Errorf("flatten PDF with multi PDF engines: %w", err) },
	)
}

// Convert transforms the given PDF to a specific PDF format using the first
// available engine that supports PDF conversion.
func (multi *multiPdfEngines) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return runWithFallbackVoid(ctx, "pdfengines.Convert", multi.convertEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.Convert(ctx, logger, formats, inputPath, outputPath)
		},
		func(err error) error {
			return fmt.Errorf("convert PDF to '%+v' with multi PDF engines: %w", formats, err)
		},
	)
}

// ReadMetadata extracts metadata from a PDF file using the first available
// engine that supports metadata reading.
func (multi *multiPdfEngines) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	return runWithFallback(ctx, "pdfengines.ReadMetadata", multi.readMetadataEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) (map[string]any, error) {
			return engine.ReadMetadata(ctx, logger, inputPath)
		},
		func(err error) error { return fmt.Errorf("read PDF metadata with multi PDF engines: %w", err) },
	)
}

// WriteMetadata embeds metadata into a PDF file using the first available
// engine that supports metadata writing.
func (multi *multiPdfEngines) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	return runWithFallbackVoid(ctx, "pdfengines.WriteMetadata", multi.writeMetadataEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.WriteMetadata(ctx, logger, metadata, inputPath)
		},
		func(err error) error { return fmt.Errorf("write PDF metadata with multi PDF engines: %w", err) },
	)
}

// PageCount returns the number of pages in a PDF file using the first available
// engine that supports metadata reading.
func (multi *multiPdfEngines) PageCount(ctx context.Context, logger *slog.Logger, inputPath string) (int, error) {
	return runWithFallback(ctx, "pdfengines.PageCount", multi.readMetadataEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) (int, error) {
			return engine.PageCount(ctx, logger, inputPath)
		},
		func(err error) error { return fmt.Errorf("page count with multi PDF engines: %w", err) },
	)
}

// ReadBookmarks reads bookmarks from a PDF file using the first available
// engine that supports bookmarks reading.
func (multi *multiPdfEngines) ReadBookmarks(ctx context.Context, logger *slog.Logger, inputPath string) ([]gotenberg.Bookmark, error) {
	return runWithFallback(ctx, "pdfengines.ReadBookmarks", multi.readBookmarksEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) ([]gotenberg.Bookmark, error) {
			return engine.ReadBookmarks(ctx, logger, inputPath)
		},
		func(err error) error { return fmt.Errorf("read PDF bookmarks with multi PDF engines: %w", err) },
	)
}

// WriteBookmarks adds a document outline (bookmarks) to a PDF file using the
// first available engine that supports bookmarks writing.
func (multi *multiPdfEngines) WriteBookmarks(ctx context.Context, logger *slog.Logger, inputPath string, bookmarks []gotenberg.Bookmark) error {
	return runWithFallbackVoid(ctx, "pdfengines.WriteBookmarks", multi.writeBookmarksEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.WriteBookmarks(ctx, logger, inputPath, bookmarks)
		},
		func(err error) error { return fmt.Errorf("write PDF bookmarks with multi PDF engines: %w", err) },
	)
}

// Encrypt adds password protection to a PDF file using the first available
// engine that supports password protection.
func (multi *multiPdfEngines) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	return runWithFallbackVoid(ctx, "pdfengines.Encrypt", multi.passwordEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.Encrypt(ctx, logger, inputPath, userPassword, ownerPassword)
		},
		func(err error) error { return fmt.Errorf("encrypt PDF using multi PDF engines: %w", err) },
	)
}

// EmbedFiles embeds files into a PDF using the first available engine that
// supports file embedding.
func (multi *multiPdfEngines) EmbedFiles(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	return runWithFallbackVoid(ctx, "pdfengines.EmbedFiles", multi.embedEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.EmbedFiles(ctx, logger, filePaths, inputPath)
		},
		func(err error) error { return fmt.Errorf("embed files into PDF using multi PDF engines: %w", err) },
	)
}

// Watermark applies a watermark (behind page content) to a PDF file using the
// first available engine that supports watermarking.
func (multi *multiPdfEngines) Watermark(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	return runWithFallbackVoid(ctx, "pdfengines.Watermark", multi.watermarkEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.Watermark(ctx, logger, inputPath, stamp)
		},
		func(err error) error { return fmt.Errorf("watermark PDF with multi PDF engines: %w", err) },
	)
}

// Stamp applies a stamp (on top of page content) to a PDF file using the first
// available engine that supports stamping.
func (multi *multiPdfEngines) Stamp(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	return runWithFallbackVoid(ctx, "pdfengines.Stamp", multi.stampEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.Stamp(ctx, logger, inputPath, stamp)
		},
		func(err error) error { return fmt.Errorf("stamp PDF with multi PDF engines: %w", err) },
	)
}

// Rotate rotates pages of a PDF file using the first available engine that
// supports rotation.
func (multi *multiPdfEngines) Rotate(ctx context.Context, logger *slog.Logger, inputPath string, angle int, pages string) error {
	return runWithFallbackVoid(ctx, "pdfengines.Rotate", multi.rotateEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.Rotate(ctx, logger, inputPath, angle, pages)
		},
		func(err error) error { return fmt.Errorf("rotate PDF with multi PDF engines: %w", err) },
	)
}

// EmbedFilesMetadata sets metadata on embedded files using the first available
// engine that supports it.
func (multi *multiPdfEngines) EmbedFilesMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]map[string]string, inputPath string) error {
	return runWithFallbackVoid(ctx, "pdfengines.EmbedFilesMetadata", multi.embedMetadataEngines,
		func(ctx context.Context, engine gotenberg.PdfEngine) error {
			return engine.EmbedFilesMetadata(ctx, logger, metadata, inputPath)
		},
		func(err error) error { return fmt.Errorf("set embeds metadata using multi PDF engines: %w", err) },
	)
}

// Interface guards.
var (
	_ gotenberg.PdfEngine = (*multiPdfEngines)(nil)
)
