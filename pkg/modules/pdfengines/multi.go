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
	mergeEngines       []gotenberg.PdfEngine
	splitEngines       []gotenberg.PdfEngine
	convertEngines     []gotenberg.PdfEngine
	readMedataEngines  []gotenberg.PdfEngine
	writeMedataEngines []gotenberg.PdfEngine
}

func newMultiPdfEngines(
	mergeEngines,
	splitEngines,
	convertEngines,
	readMetadataEngines,
	writeMedataEngines []gotenberg.PdfEngine,
) *multiPdfEngines {
	return &multiPdfEngines{
		mergeEngines:       mergeEngines,
		splitEngines:       splitEngines,
		convertEngines:     convertEngines,
		readMedataEngines:  readMetadataEngines,
		writeMedataEngines: writeMedataEngines,
	}
}

// Merge tries to merge the given PDFs into a unique PDF thanks to its
// children. If the context is done, it stops and returns an error.
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

// Split tries to split at intervals a given PDF thanks to its children. If the
// context is done, it stops and returns an error.
func (multi *multiPdfEngines) Split(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	var err error
	var mu sync.Mutex // to safely append errors.

	resultChan := make(chan splitResult, len(multi.splitEngines))

	for _, engine := range multi.splitEngines {
		go func(engine gotenberg.PdfEngine) {
			outputPaths, err := engine.Split(ctx, logger, mode, inputPath, outputDirPath)
			resultChan <- splitResult{outputPaths: outputPaths, err: err}
		}(engine)
	}

	for range multi.splitEngines {
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

// Convert converts the given PDF to a specific PDF format. thanks to its
// children. If the context is done, it stops and returns an error.
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

func (multi *multiPdfEngines) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
	var err error
	var mu sync.Mutex // to safely append errors.

	resultChan := make(chan readMetadataResult, len(multi.readMedataEngines))

	for _, engine := range multi.readMedataEngines {
		go func(engine gotenberg.PdfEngine) {
			metadata, err := engine.ReadMetadata(ctx, logger, inputPath)
			resultChan <- readMetadataResult{metadata: metadata, err: err}
		}(engine)
	}

	for range multi.readMedataEngines {
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

func (multi *multiPdfEngines) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.writeMedataEngines {
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

// Interface guards.
var (
	_ gotenberg.PdfEngine = (*multiPdfEngines)(nil)
)
