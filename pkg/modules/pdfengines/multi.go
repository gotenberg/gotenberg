package pdfengines

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

type multiPdfEngines struct {
	mergeEngines         []gotenberg.PdfEngine
	splitEngines         []gotenberg.PdfEngine
	flattenEngines       []gotenberg.PdfEngine
	convertEngines       []gotenberg.PdfEngine
	readMetadataEngines  []gotenberg.PdfEngine
	writeMetadataEngines []gotenberg.PdfEngine
	passwordEngines      []gotenberg.PdfEngine
	embedEngines         []gotenberg.PdfEngine
}

func newMultiPdfEngines(
	mergeEngines,
	splitEngines,
	flattenEngines,
	convertEngines,
	readMetadataEngines,
	writeMetadataEngines,
	passwordEngines,
	embedEngines []gotenberg.PdfEngine,
) *multiPdfEngines {
	return &multiPdfEngines{
		mergeEngines:         mergeEngines,
		splitEngines:         splitEngines,
		flattenEngines:       flattenEngines,
		convertEngines:       convertEngines,
		readMetadataEngines:  readMetadataEngines,
		writeMetadataEngines: writeMetadataEngines,
		passwordEngines:      passwordEngines,
		embedEngines:         embedEngines,
	}
}

// Merge combines multiple PDF files into a single document using the first
// available engine that supports PDF merging.
func (multi *multiPdfEngines) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.mergeEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Merge(ctx, logger, inputPaths, outputPath)
		}(engine)

		select {
		case mergeErr := <-errChan:
			errored := multierr.AppendInto(&err, mergeErr)
			if !errored {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("merge PDFs with multi PDF engines: %w", err)
}

type splitResult struct {
	outputPaths []string
	err         error
}

// Split divides the PDF into separate pages using the first available engine
// that supports PDF splitting.
func (multi *multiPdfEngines) Split(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
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
				err = multierr.Append(err, result.err)
				mu.Unlock()
			} else {
				return result.outputPaths, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("split PDF with multi PDF engines: %w", err)
}

// Flatten merges existing annotation appearances with page content using the
// first available engine that supports flattening.
func (multi *multiPdfEngines) Flatten(ctx context.Context, logger *zap.Logger, inputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.flattenEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Flatten(ctx, logger, inputPath)
		}(engine)

		select {
		case mergeErr := <-errChan:
			errored := multierr.AppendInto(&err, mergeErr)
			if !errored {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("flatten PDF with multi PDF engines: %w", err)
}

// Convert transforms the given PDF to a specific PDF format using the first
// available engine that supports PDF conversion.
func (multi *multiPdfEngines) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.convertEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Convert(ctx, logger, formats, inputPath, outputPath)
		}(engine)

		select {
		case mergeErr := <-errChan:
			errored := multierr.AppendInto(&err, mergeErr)
			if !errored {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("convert PDF to '%+v' with multi PDF engines: %w", formats, err)
}

type readMetadataResult struct {
	metadata map[string]interface{}
	err      error
}

// ReadMetadata extracts metadata from a PDF file using the first available
// engine that supports metadata reading.
func (multi *multiPdfEngines) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
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
				err = multierr.Append(err, result.err)
				mu.Unlock()
			} else {
				return result.metadata, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("read PDF metadata with multi PDF engines: %w", err)
}

// WriteMetadata embeds metadata into a PDF file using the first available
// engine that supports metadata writing.
func (multi *multiPdfEngines) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.writeMetadataEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.WriteMetadata(ctx, logger, metadata, inputPath)
		}(engine)

		select {
		case writeMetadataErr := <-errChan:
			errored := multierr.AppendInto(&err, writeMetadataErr)
			if !errored {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("write PDF metadata with multi PDF engines: %w", err)
}

// Encrypt adds password protection to a PDF file using the first available
// engine that supports password protection.
func (multi *multiPdfEngines) Encrypt(ctx context.Context, logger *zap.Logger, inputPath, userPassword, ownerPassword string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.passwordEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.Encrypt(ctx, logger, inputPath, userPassword, ownerPassword)
		}(engine)

		select {
		case protectErr := <-errChan:
			errored := multierr.AppendInto(&err, protectErr)
			if !errored {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("encrypt PDF using multi PDF engines: %w", err)
}

// EmbedFiles embeds files into a PDF using the first available
// engine that supports file embedding.
func (multi *multiPdfEngines) EmbedFiles(ctx context.Context, logger *zap.Logger, filePaths []string, inputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.embedEngines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.EmbedFiles(ctx, logger, filePaths, inputPath)
		}(engine)

		select {
		case embedErr := <-errChan:
			errored := multierr.AppendInto(&err, embedErr)
			if !errored {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("embed files into PDF using multi PDF engines: %w", err)
}

// Interface guards.
var (
	_ gotenberg.PdfEngine = (*multiPdfEngines)(nil)
)
