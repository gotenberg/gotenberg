package pdfengine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func init() {
	gotenberg.MustRegisterModule(new(LibreOfficePdfEngine))
}

// LibreOfficePdfEngine interacts with the LibreOffice (Universal Network Objects) API
// and implements the [gotenberg.PdfEngine] interface.
type LibreOfficePdfEngine struct {
	unoApi api.Uno
}

// Descriptor returns a [LibreOfficePdfEngine]'s module descriptor.
func (engine *LibreOfficePdfEngine) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "libreoffice-pdfengine",
		New: func() gotenberg.Module { return new(LibreOfficePdfEngine) },
	}
}

// Provision sets the module properties.
func (engine *LibreOfficePdfEngine) Provision(ctx *gotenberg.Context) error {
	provider, err := ctx.Module(new(api.Provider))
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno provider: %w", err)
	}

	unoApi, err := provider.(api.Provider).LibreOffice()
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno: %w", err)
	}

	engine.unoApi = unoApi

	return nil
}

// Merge is not available in this implementation.
func (engine *LibreOfficePdfEngine) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.Merge", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("merge PDFs with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// Split is not available in this implementation.
func (engine *LibreOfficePdfEngine) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	_, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.Split", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("split PDF with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return nil, err
}

// Flatten is not available in this implementation.
func (engine *LibreOfficePdfEngine) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.Flatten", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("flatten PDF with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// Convert converts the given PDF to a specific PDF format. Currently, only the
// PDF/A-1b, PDF/A-2b, PDF/A-3b and PDF/UA formats are available. If another
// PDF format is requested, it returns a [gotenberg.ErrPdfFormatNotSupported]
// error.
func (engine *LibreOfficePdfEngine) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.Convert", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	opts := api.DefaultOptions()
	opts.PdfFormats = formats
	err := engine.unoApi.Pdf(ctx, logger, inputPath, outputPath, opts)

	if err == nil {
		return nil
	}

	if errors.Is(err, api.ErrInvalidPdfFormats) {
		err = fmt.Errorf("convert PDF to '%+v' with LibreOffice: %w", formats, gotenberg.ErrPdfFormatNotSupported)
	} else {
		err = fmt.Errorf("convert PDF to '%+v' with LibreOffice: %w", formats, err)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// ReadMetadata is not available in this implementation.
func (engine *LibreOfficePdfEngine) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	_, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.ReadMetadata", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("read PDF metadata with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return nil, err
}

// WriteMetadata is not available in this implementation.
func (engine *LibreOfficePdfEngine) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.WriteMetadata", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("write PDF metadata with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// Encrypt is not available in this implementation.
func (engine *LibreOfficePdfEngine) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	_, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.Encrypt", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("encrypt PDF using LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// AddAttachments is not available in this implementation.
func (engine *LibreOfficePdfEngine) AddAttachments(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "LibreOfficePdfEngine.AddAttachments", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	err := fmt.Errorf("add attachments with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// Interface guards.
var (
	_ gotenberg.Module      = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.Provisioner = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.PdfEngine   = (*LibreOfficePdfEngine)(nil)
)
