package exiftool

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/barasher/go-exiftool"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(ExifTool))
}

// ExifTool abstracts the CLI tool ExifTool and implements the
// [gotenberg.PdfEngine] interface.
type ExifTool struct {
	binPath string
}

// Descriptor returns [ExifTool]'s module descriptor.
func (engine *ExifTool) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "exiftool",
		New: func() gotenberg.Module { return new(ExifTool) },
	}
}

// Provision sets the module properties.
func (engine *ExifTool) Provision(ctx *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("EXIFTOOL_BIN_PATH")
	if !ok {
		return errors.New("EXIFTOOL_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine *ExifTool) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("ExifTool binary path does not exist: %w", err)
	}

	return nil
}

// Merge is not available in this implementation.
func (engine *ExifTool) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return fmt.Errorf("merge PDFs with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Convert is not available in this implementation.
func (engine *ExifTool) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with ExifTool: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

// ReadMetadata extracts the metadata of a given PDF file.
func (engine *ExifTool) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
	exifTool, err := exiftool.NewExiftool(exiftool.SetExiftoolBinaryPath(engine.binPath))
	if err != nil {
		return nil, fmt.Errorf("new ExifTool: %w", err)
	}

	defer func(exifTool *exiftool.Exiftool) {
		err := exifTool.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("close ExifTool: %v", err))
		}
	}(exifTool)

	fileMetadata := exifTool.ExtractMetadata(inputPath)
	if fileMetadata[0].Err != nil {
		return nil, fmt.Errorf("read metadata with ExitfTool: %w", fileMetadata[0].Err)
	}

	return fileMetadata[0].Fields, nil
}

// WriteMetadata writes the metadata into a given PDF file.
func (engine *ExifTool) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
	exifTool, err := exiftool.NewExiftool(exiftool.SetExiftoolBinaryPath(engine.binPath))
	if err != nil {
		return fmt.Errorf("new ExifTool: %w", err)
	}

	defer func(exifTool *exiftool.Exiftool) {
		err := exifTool.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("close ExifTool: %v", err))
		}
	}(exifTool)

	fileMetadata := exifTool.ExtractMetadata(inputPath)
	if fileMetadata[0].Err != nil {
		return fmt.Errorf("read metadata with ExitfTool: %w", fileMetadata[0].Err)
	}

	for key, value := range metadata {
		switch val := value.(type) {
		case string:
			fileMetadata[0].SetString(key, val)
		case []string:
			fileMetadata[0].SetStrings(key, val)
		case bool:
			fileMetadata[0].SetString(key, fmt.Sprintf("%t", val))
		case int:
			fileMetadata[0].SetInt(key, int64(val))
		case int64:
			fileMetadata[0].SetInt(key, val)
		case float32:
			fileMetadata[0].SetFloat(key, float64(val))
		case float64:
			fileMetadata[0].SetFloat(key, val)
		// TODO: support more complex cases, e.g., arrays and nested objects
		// 	(limitations in underlying library).
		default:
			return fmt.Errorf("write PDF metadata with ExifTool: %w", gotenberg.ErrPdfEngineMetadataValueNotSupported)
		}
	}

	exifTool.WriteMetadata(fileMetadata)
	if fileMetadata[0].Err != nil {
		return fmt.Errorf("write PDF metadata with ExifTool: %w", fileMetadata[0].Err)
	}

	return nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*ExifTool)(nil)
	_ gotenberg.Provisioner = (*ExifTool)(nil)
	_ gotenberg.Validator   = (*ExifTool)(nil)
	_ gotenberg.PdfEngine   = (*ExifTool)(nil)
)
