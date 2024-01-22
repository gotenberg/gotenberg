package pdfengines

import (
	"context"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

type multiPdfEngines struct {
	engines []gotenberg.PdfEngine
}

func newMultiPdfEngines(engines ...gotenberg.PdfEngine) *multiPdfEngines {
	return &multiPdfEngines{
		engines: engines,
	}
}

// Merge tries to merge the given PDFs into a unique PDF thanks to its
// children. If the context is done, it stops and returns an error.
func (multi *multiPdfEngines) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.engines {
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

// Convert converts the given PDF to a specific PDF format. thanks to its
// children. If the context is done, it stops and returns an error.
func (multi *multiPdfEngines) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.engines {
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

func (multi *multiPdfEngines) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPaths string, metadata map[string]interface{}) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.engines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.ReadMetadata(ctx, logger, inputPaths, metadata)
		}(engine)

		select {
		case readMetadataErr := <-errChan:
			errored := multierr.AppendInto(&err, readMetadataErr)
			if !errored {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("read PDF metadata with multi PDF engines: %w", err)
}

func (multi *multiPdfEngines) WriteMetadata(ctx context.Context, logger *zap.Logger, inputPaths string, newMetadata map[string]interface{}) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.engines {
		go func(engine gotenberg.PdfEngine) {
			errChan <- engine.WriteMetadata(ctx, logger, inputPaths, newMetadata)
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
