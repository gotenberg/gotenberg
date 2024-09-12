package qpdf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	libeofficeapi "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func init() {
	gotenberg.MustRegisterModule(new(QPdf))
}

// QPdf abstracts the CLI tool QPDF and implements the [gotenberg.PdfEngine]
// interface.
type QPdf struct {
	binPath     string
	libreoffice libeofficeapi.Uno
}

// Descriptor returns a [QPdf]'s module descriptor.
func (engine *QPdf) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "qpdf",
		New: func() gotenberg.Module { return new(QPdf) },
	}
}

// Provision sets the modules properties.
func (engine *QPdf) Provision(ctx *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("QPDF_BIN_PATH")
	if !ok {
		return errors.New("QPDF_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	provider, err := ctx.Module(new(libeofficeapi.Provider))
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno provider: %w", err)
	}

	libreoffice, err := provider.(libeofficeapi.Provider).LibreOffice()
	if err != nil {
		return fmt.Errorf("get LibreOffice Uno: %w", err)
	}

	engine.libreoffice = libreoffice

	return nil
}

// Validate validates the module properties.
func (engine *QPdf) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("QPdf binary path does not exist: %w", err)
	}

	return nil
}

// Merge combines multiple PDFs into a single PDF.
func (engine *QPdf) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var args []string
	args = append(args, "--empty", "--pages")
	args = append(args, inputPaths...)
	args = append(args, "--", outputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with QPDF: %w", err)
}

// Convert is not available in this implementation.
func (engine *QPdf) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with QPDF: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

func (engine *QPdf) optimizeImages(ctx context.Context, logger *zap.Logger, quality, maxImageResolution int, inputPath, outputPath string) error {
	opts := libeofficeapi.DefaultOptions()
	if quality != 0 {
		opts.Quality = quality
	}
	if maxImageResolution != 0 {
		opts.ReduceImageResolution = true
		opts.MaxImageResolution = maxImageResolution
	}

	err := engine.libreoffice.Pdf(ctx, logger, inputPath, outputPath, opts)
	if err == nil {
		return nil
	}

	return fmt.Errorf("optimize images in PDF with LibreOffice: %w", err)
}

func (engine *QPdf) linearize(ctx context.Context, logger *zap.Logger, inputPath, outputPath string) error {
	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, "--linearize", inputPath, outputPath)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err == nil {
		return nil
	}

	return fmt.Errorf("linearize PDF with QPDF: %w", err)
}

func (engine *QPdf) compressStreams(ctx context.Context, logger *zap.Logger, inputPath string, outputPath string) error {
	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath,
		"--compress-streams=y",
		"--object-streams=generate",
		"--recompress-flate",
		"--compression-level=9",
		inputPath, outputPath)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err == nil {
		return nil
	}

	return fmt.Errorf("compress streams in PDF with QPDF: %w", err)
}

// Optimize optimizes (compresses) a given PDF.
func (engine *QPdf) Optimize(ctx context.Context, logger *zap.Logger, options gotenberg.OptimizeOptions, inputPath, outputPath string) error {
	basePath := filepath.Dir(inputPath)

	if !options.HasImagesOptimization() || options.SkipLo {
		logger.Debug("skip images optimization with LibreOffice")
	}

	if options.HasImagesOptimization() && !options.SkipLo {
		logger.Debug("optimize images with LibreOffice")
		var optimizedImagesOutputPath string
		if options.CompressStreams {
			optimizedImagesOutputPath = filepath.Join(basePath, fmt.Sprintf("%s.pdf", uuid.NewString()))
		} else {
			optimizedImagesOutputPath = outputPath
		}

		err := engine.optimizeImages(ctx, logger, options.ImageQuality, options.MaxImageResolution, inputPath, optimizedImagesOutputPath)
		if err != nil {
			return fmt.Errorf("optimize PDF with LibreOffice: %w", err)
		}

		if options.CompressStreams {
			inputPath = optimizedImagesOutputPath
		}
	}

	if !options.CompressStreams {
		logger.Debug("skip streams compression with QPDF")
	}

	if options.CompressStreams {
		logger.Debug("linearize with QPDF")
		linearizedOutputPath := filepath.Join(basePath, fmt.Sprintf("%s.pdf", uuid.NewString()))
		err := engine.linearize(ctx, logger, inputPath, linearizedOutputPath)
		if err != nil {
			return fmt.Errorf("optimize PDF with QPDF: %w", err)
		}

		logger.Debug("compress streams with QPDF")
		err = engine.compressStreams(ctx, logger, linearizedOutputPath, outputPath)
		if err != nil {
			return fmt.Errorf("optimize PDF with QPDF: %w", err)
		}
	}

	return nil
}

// ReadMetadata is not available in this implementation.
func (engine *QPdf) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("read PDF metadata with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// WriteMetadata is not available in this implementation.
func (engine *QPdf) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	return fmt.Errorf("write PDF metadata with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

var (
	_ gotenberg.Module      = (*QPdf)(nil)
	_ gotenberg.Provisioner = (*QPdf)(nil)
	_ gotenberg.Validator   = (*QPdf)(nil)
	_ gotenberg.PdfEngine   = (*QPdf)(nil)
)
