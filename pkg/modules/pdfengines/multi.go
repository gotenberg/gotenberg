package pdfengines

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

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
		readBookmarksEngines:  readBookmarksEngines,
		writeBookmarksEngines: writeBookmarksEngines,
		watermarkEngines:      watermarkEngines,
		stampEngines:          stampEngines,
		rotateEngines:         rotateEngines,
	}
}

// Merge combines multiple PDF files into a single document using the first
// available engine that supports PDF merging.
//
//nolint:dupl
func (multi *multiPdfEngines) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Merge", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.mergeEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Merge(ctx, logger, inputPaths, outputPath)
		}(engine)

		select {
		case mergeErr := <-errChan:
			if mergeErr != nil {
				err = errors.Join(err, mergeErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("merge PDFs with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

type splitResult struct {
	outputPaths []string
	err         error
}

// Split divides the PDF into separate pages using the first available engine
// that supports PDF splitting.
func (multi *multiPdfEngines) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Split", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	var mu sync.Mutex // to safely append errors.

	for _, engine := range multi.splitEngines {
		resultChan := make(chan splitResult, 1)

		go func(engine gotenberg.PdfEngine) {
			outputPaths, err := engine.Split(ctx, logger, mode, inputPath, outputDirPath)
			resultChan <- splitResult{outputPaths: outputPaths, err: err}
		}(engine)

		select {
		case result := <-resultChan:
			if result.err != nil {
				mu.Lock()
				err = errors.Join(err, result.err)
				mu.Unlock()
			} else {
				span.SetStatus(codes.Ok, "")
				return result.outputPaths, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	err = fmt.Errorf("split PDF with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return nil, err
}

// Flatten merges existing annotation appearances with page content using the
// first available engine that supports flattening.
func (multi *multiPdfEngines) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Flatten", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.flattenEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Flatten(ctx, logger, inputPath)
		}(engine)

		select {
		case mergeErr := <-errChan:
			if mergeErr != nil {
				err = errors.Join(err, mergeErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("flatten PDF with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// Convert transforms the given PDF to a specific PDF format using the first
// available engine that supports PDF conversion.
func (multi *multiPdfEngines) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Convert", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.convertEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Convert(ctx, logger, formats, inputPath, outputPath)
		}(engine)

		select {
		case mergeErr := <-errChan:
			if mergeErr != nil {
				err = errors.Join(err, mergeErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("convert PDF to '%+v' with multi PDF engines: %w", formats, err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

type readMetadataResult struct {
	metadata map[string]any
	err      error
}

// ReadMetadata extracts metadata from a PDF file using the first available
// engine that supports metadata reading.
//
//nolint:dupl
func (multi *multiPdfEngines) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.ReadMetadata", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	var mu sync.Mutex // to safely append errors.

	for _, engine := range multi.readMetadataEngines {
		resultChan := make(chan readMetadataResult, 1)

		go func(engine gotenberg.PdfEngine) {
			metadata, err := engine.ReadMetadata(ctx, logger, inputPath)
			resultChan <- readMetadataResult{metadata: metadata, err: err}
		}(engine)

		select {
		case result := <-resultChan:
			if result.err != nil {
				mu.Lock()
				err = errors.Join(err, result.err)
				mu.Unlock()
			} else {
				span.SetStatus(codes.Ok, "")
				return result.metadata, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	err = fmt.Errorf("read PDF metadata with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return nil, err
}

// WriteMetadata embeds metadata into a PDF file using the first available
// engine that supports metadata writing.
func (multi *multiPdfEngines) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.WriteMetadata", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.writeMetadataEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.WriteMetadata(ctx, logger, metadata, inputPath)
		}(engine)

		select {
		case writeMetadataErr := <-errChan:
			if writeMetadataErr != nil {
				err = errors.Join(err, writeMetadataErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("write PDF metadata with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

type pageCountResult struct {
	pageCount int
	err       error
}

// PageCount returns the number of pages in a PDF file using the first available
// engine that supports metadata reading.
func (multi *multiPdfEngines) PageCount(ctx context.Context, logger *slog.Logger, inputPath string) (int, error) {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.PageCount", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	var mu sync.Mutex // to safely append errors.

	for _, engine := range multi.readMetadataEngines {
		resultChan := make(chan pageCountResult, 1)

		go func(engine gotenberg.PdfEngine) {
			pageCount, err := engine.PageCount(ctx, logger, inputPath)
			resultChan <- pageCountResult{pageCount: pageCount, err: err}
		}(engine)

		select {
		case result := <-resultChan:
			if result.err != nil {
				mu.Lock()
				err = errors.Join(err, result.err)
				mu.Unlock()
			} else {
				span.SetStatus(codes.Ok, "")
				return result.pageCount, nil
			}
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

	err = fmt.Errorf("page count with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return 0, err
}

type readBookmarksResult struct {
	bookmarks []gotenberg.Bookmark
	err       error
}

// ReadBookmarks reads bookmarks from a PDF file using the first available
// engine that supports bookmarks reading.
//
//nolint:dupl
func (multi *multiPdfEngines) ReadBookmarks(ctx context.Context, logger *slog.Logger, inputPath string) ([]gotenberg.Bookmark, error) {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.ReadBookmarks", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	var mu sync.Mutex // to safely append errors.

	for _, engine := range multi.readBookmarksEngines {
		resultChan := make(chan readBookmarksResult, 1)

		go func(engine gotenberg.PdfEngine) {
			bookmarks, err := engine.ReadBookmarks(ctx, logger, inputPath)
			resultChan <- readBookmarksResult{bookmarks: bookmarks, err: err}
		}(engine)

		select {
		case result := <-resultChan:
			if result.err != nil {
				mu.Lock()
				err = errors.Join(err, result.err)
				mu.Unlock()
			} else {
				span.SetStatus(codes.Ok, "")
				return result.bookmarks, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	err = fmt.Errorf("read PDF bookmarks with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return nil, err
}

// WriteBookmarks adds a document outline (bookmarks) to a PDF file using the
// first available engine that supports bookmarks writing.
func (multi *multiPdfEngines) WriteBookmarks(ctx context.Context, logger *slog.Logger, inputPath string, bookmarks []gotenberg.Bookmark) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.WriteBookmarks", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.writeBookmarksEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.WriteBookmarks(ctx, logger, inputPath, bookmarks)
		}(engine)

		select {
		case writeBookmarksErr := <-errChan:
			if writeBookmarksErr != nil {
				err = errors.Join(err, writeBookmarksErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("write PDF bookmarks with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// Encrypt adds password protection to a PDF file using the first available
// engine that supports password protection.
func (multi *multiPdfEngines) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Encrypt", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.passwordEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Encrypt(ctx, logger, inputPath, userPassword, ownerPassword)
		}(engine)

		select {
		case protectErr := <-errChan:
			if protectErr != nil {
				err = errors.Join(err, protectErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("encrypt PDF using multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// EmbedFiles embeds files into a PDF using the first available
// engine that supports file embedding.
//
//nolint:dupl
func (multi *multiPdfEngines) EmbedFiles(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.EmbedFiles", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.embedEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.EmbedFiles(ctx, logger, filePaths, inputPath)
		}(engine)

		select {
		case embedErr := <-errChan:
			if embedErr != nil {
				err = errors.Join(err, embedErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("embed files into PDF using multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// Watermark applies a watermark (behind page content) to a PDF file using the
// first available engine that supports watermarking.
//
//nolint:dupl
func (multi *multiPdfEngines) Watermark(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Watermark", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.watermarkEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Watermark(ctx, logger, inputPath, stamp)
		}(engine)

		select {
		case watermarkErr := <-errChan:
			if watermarkErr != nil {
				err = errors.Join(err, watermarkErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("watermark PDF with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// Stamp applies a stamp (on top of page content) to a PDF file using the
// first available engine that supports stamping.
//
//nolint:dupl
func (multi *multiPdfEngines) Stamp(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Stamp", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.stampEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Stamp(ctx, logger, inputPath, stamp)
		}(engine)

		select {
		case stampErr := <-errChan:
			if stampErr != nil {
				err = errors.Join(err, stampErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("stamp PDF with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// Rotate rotates pages of a PDF file using the first available engine that
// supports rotation.
func (multi *multiPdfEngines) Rotate(ctx context.Context, logger *slog.Logger, inputPath string, angle int, pages string) error {
	tracer := gotenberg.Tracer()
	ctx, span := tracer.Start(ctx, "pdfengines.Rotate", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.rotateEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Rotate(ctx, logger, inputPath, angle, pages)
		}(engine)

		select {
		case rotateErr := <-errChan:
			if rotateErr != nil {
				err = errors.Join(err, rotateErr)
			} else {
				span.SetStatus(codes.Ok, "")
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	err = fmt.Errorf("rotate PDF with multi PDF engines: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// Interface guards.
var (
	_ gotenberg.PdfEngine = (*multiPdfEngines)(nil)
)
