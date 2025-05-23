package pdfengine

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"

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
func (engine *LibreOfficePdfEngine) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return fmt.Errorf("merge PDFs with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Split is not available in this implementation.
func (engine *LibreOfficePdfEngine) Split(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	return nil, fmt.Errorf("split PDF with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Flatten is not available in this implementation.
func (engine *LibreOfficePdfEngine) Flatten(ctx context.Context, logger *zap.Logger, inputPath string) error {
	return fmt.Errorf("flatten PDF with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Convert converts the given PDF to a specific PDF format. Currently, only the
// PDF/A-1b, PDF/A-2b, PDF/A-3b and PDF/UA formats are available. If another
// PDF format is requested, it returns a [gotenberg.ErrPdfFormatNotSupported]
// error.
func (engine *LibreOfficePdfEngine) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	opts := api.DefaultOptions()
	opts.PdfFormats = formats
	err := engine.unoApi.Pdf(ctx, logger, inputPath, outputPath, opts)

	if err == nil {
		return nil
	}

	if errors.Is(err, api.ErrInvalidPdfFormats) {
		return fmt.Errorf("convert PDF to '%+v' with LibreOffice: %w", formats, gotenberg.ErrPdfFormatNotSupported)
	}

	return fmt.Errorf("convert PDF to '%+v' with LibreOffice: %w", formats, err)
}

// ReadMetadata is not available in this implementation.
func (engine *LibreOfficePdfEngine) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("read PDF metadata with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// WriteMetadata is not available in this implementation.
func (engine *LibreOfficePdfEngine) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	return fmt.Errorf("write PDF metadata with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// ProtectWithPassword adds password protection to a PDF file using LibreOffice.
func (engine *LibreOfficePdfEngine) ProtectWithPassword(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, userPassword, ownerPassword string) error {
	if userPassword == "" {
		return errors.New("user password cannot be empty")
	}

	// LibreOffice can set PDF passwords during conversion
	opts := api.DefaultOptions()

	// Configure PDF security options
	// LibreOffice uses the same options as during export to PDF
	err := engine.unoApi.Pdf(ctx, logger, inputPath, outputPath, opts)
	if err != nil {
		return fmt.Errorf("protect PDF with password using LibreOffice: %w", err)
	}

	// After LibreOffice generates the PDF, we'll use QPDF to add the password
	// because LibreOffice doesn't support adding passwords to existing PDFs through command line

	// Check if QPDF binary exists
	qpdfBinPath, ok := os.LookupEnv("QPDF_BIN_PATH")
	if !ok {
		return errors.New("QPDF_BIN_PATH environment variable is not set")
	}

	// If owner password is not provided, use the user password as owner password
	if ownerPassword == "" {
		ownerPassword = userPassword
	}

	// Use QPDF to apply password protection
	var args []string
	args = append(args, outputPath)
	args = append(args, "--encrypt", userPassword, ownerPassword, "128", "--use-aes=y", "--")
	args = append(args, outputPath+".protected")

	cmd, err := gotenberg.CommandContext(ctx, logger, qpdfBinPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("protect PDF with QPDF: %w", err)
	}

	// Replace the original file with the protected one
	err = os.Rename(outputPath+".protected", outputPath)
	if err != nil {
		return fmt.Errorf("replace original file with protected one: %w", err)
	}

	return nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.Provisioner = (*LibreOfficePdfEngine)(nil)
	_ gotenberg.PdfEngine   = (*LibreOfficePdfEngine)(nil)
)
