package pdfengines

import (
	"context"
	"fmt"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// multiPDFEngines implements the gotenberg.PDFEngine interface and gathers one
// or more gotenberg.PDFEngine. It provides a sort of fallback mechanism: if an
// engine's method returns an error, it calls the same method from another
// engine.
type multiPDFEngines struct {
	engines []gotenberg.PDFEngine
}

// newMultiPDFEngines returns a multiPDFEngines. Arguments' order determines the
// order of the engines called.
func newMultiPDFEngines(engines ...gotenberg.PDFEngine) *multiPDFEngines {
	return &multiPDFEngines{
		engines: engines,
	}
}

// Merge tries to merge the given PDFs into a unique PDF thanks to its
// children. If the context is done, it stops and returns an error.
func (multi multiPDFEngines) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.engines {
		go func(engine gotenberg.PDFEngine) {
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
func (multi multiPDFEngines) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	var err error
	errChan := make(chan error, 1)

	for _, engine := range multi.engines {
		go func(engine gotenberg.PDFEngine) {
			errChan <- engine.Convert(ctx, logger, format, inputPath, outputPath)
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

	return fmt.Errorf("convert PDF to '%s' with multi PDF engines: %w", format, err)
}

// Interface guards.
var (
	_ gotenberg.PDFEngine = (*multiPDFEngines)(nil)
)
