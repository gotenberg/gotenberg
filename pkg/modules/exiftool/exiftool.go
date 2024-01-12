package exiftool

import (
	"context"
	"errors"
	"fmt"
	"github.com/barasher/go-exiftool"
	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	"go.uber.org/zap"
	"net/http"
	"os"
)

func init() {
	gotenberg.MustRegisterModule(new(ExifTool))
}

// MetadataValueTypeError is constructed when metadata value types cannot be processed.
// The underlying library used in this implementation supports the writing of a limited
// number of metadata value types.
//
// For example, according to https://exiftool.org/TagNames/PDF.html metadata can be a boolean,
// i.e. "value format... may be string, date, integer, real, boolean or name".
// Furthermore, a native boolean type is also supported by JSON and Go. However, the
// underlying library does not currently support writing native Go boolean (bool) types.
// Therefore, an instance of this struct is created when a boolean metadata entry is supplied.
//
// The struct contains a key/value map corresponding to individual invalid metadata entries supplied by a consumer.
// This allows a helpful error message to be produced for API consumers.
// See API.WriteMetadata for more information on valid metadata value types.
type MetadataValueTypeError struct {
	Entries map[string]interface{}
}

// Error returns a helpful error message.
func (e *MetadataValueTypeError) Error() string {
	return fmt.Sprintf("invalid metadata value types supplied - identified by Entries: %s", e.Entries)
}

// GetKeys returns an array of keys with corresponding invalid value types,
func (e *MetadataValueTypeError) GetKeys() []string {
	keys := make([]string, len(e.Entries))
	i := 0
	for key := range e.Entries {
		keys[i] = key
		i++
	}
	return keys
}

// ExifTool abstracts the CLI tool ExifTool and implements the [gotenberg.PdfEngine] interface .
type ExifTool struct {
	binPath string
}

// ResponseError is to store the file name and error returned by exif tool
type ResponseError struct {
	fileName string
	err      error
}

// Descriptor returns ExifTool's module descriptor.
func (engine *ExifTool) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "exiftool",
		New: func() gotenberg.Module { return new(ExifTool) },
	}
}

// Provision sets the module properties. It returns an error if
// - the environment variable EXIFTOOL_BIN_PATH is not set
// - there is an error creating an instance of exiftool.ExifTool
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
	return fmt.Errorf("merge PDFs with LibreOffice: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Convert is not available in this implementation.
func (engine *ExifTool) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with PDFtk: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

// ReadMetadata reads the exif metadata of the given PDF files.
func (engine *ExifTool) ReadMetadata(ctx context.Context, logger *zap.Logger, paths []string, metadata []gotenberg.FileMetadata) error {
	logger.Debug(fmt.Sprintf("reading metadata of files: %s", paths))

	exifTool, err := exiftool.NewExiftool()
	if err != nil {
		fmt.Printf("Error intializing ExifTool: %v\n", err)
		return err
	}

	fileMetadataInfos := exifTool.ExtractMetadata(paths...)

	for idx, fileMetadataInfo := range fileMetadataInfos {
		if fileMetadataInfo.Err != nil {
			return fmt.Errorf("error reading metadata from %s: %w", fileMetadataInfo.File, fileMetadataInfo.Err)
		}
		// load the result into metadata passed
		metadata[idx].Path = fileMetadataInfo.File
		metadata[idx].Metadata = fileMetadataInfo.Fields
	}

	return exifTool.Close()
}

// WriteMetadata write the exif metadata to the given PDF files.
func (engine *ExifTool) WriteMetadata(ctx context.Context, logger *zap.Logger, paths []string, newMetadata map[string]interface{}) error {
	logger.Debug(fmt.Sprintf("writing new metadata %s to %s", newMetadata, paths))

	exifTool, err := exiftool.NewExiftool()
	if err != nil {
		fmt.Printf("Error intializing ExifTool: %v\n", err)
		return err
	}

	fileMetadataInfos := exifTool.ExtractMetadata(paths...)
	var extractMetadataErrors []ResponseError
	for _, fileMetadataInfo := range fileMetadataInfos {
		if fileMetadataInfo.Err != nil {
			extractMetadataErrors = append(extractMetadataErrors, ResponseError{fileName: fileMetadataInfo.File, err: fileMetadataInfo.Err})
		}
	}

	if len(extractMetadataErrors) > 0 {
		return fmt.Errorf("error reading metadata to following file(s): %+v", extractMetadataErrors)
	}

	// Metadata values can only be specific value types.
	// An error is returned if metadata with an invalid type is requested.
	metadataValueErrors := MetadataValueTypeError{
		Entries: make(map[string]interface{}),
	}

	//transform metadata
	for _, fileMetadataInfo := range fileMetadataInfos {
		for key, value := range newMetadata {
			switch value.(type) {
			case string:
				fileMetadataInfo.SetString(key, value.(string))
			case int:
				fileMetadataInfo.SetInt(key, int64(value.(int)))
			case int64:
				fileMetadataInfo.SetInt(key, value.(int64))
			case float32:
				fileMetadataInfo.SetFloat(key, float64(value.(float32)))
			case float64:
				fileMetadataInfo.SetFloat(key, value.(float64))
			case []string:
				fileMetadataInfo.SetStrings(key, value.([]string))
			// TODO: support more complex cases, e.g. arrays and nested objects (limitations in underlying library)
			default:
				metadataValueErrors.Entries[key] = value
			}
		}
		logger.Debug(fmt.Sprintf("writing metadata %s to %s", fileMetadataInfo.Fields, fileMetadataInfo.File))
	}

	if len(metadataValueErrors.Entries) > 0 {
		return api.WrapError(
			fmt.Errorf("write metadata: %w", err),
			api.NewSentinelHttpError(
				http.StatusBadRequest,
				fmt.Sprintf("Invalid metdata value types supplied by keys '%s'", metadataValueErrors.GetKeys())),
		)
	}

	exifTool.WriteMetadata(fileMetadataInfos)
	var writeMetadataErrors []ResponseError
	for _, fileMetadataInfo := range fileMetadataInfos {
		if fileMetadataInfo.Err != nil {
			writeMetadataErrors = append(writeMetadataErrors, ResponseError{fileName: fileMetadataInfo.File, err: fileMetadataInfo.Err})
		}
	}

	if len(writeMetadataErrors) > 0 {
		return fmt.Errorf("error writing metadata to following file(s): %+v", writeMetadataErrors)
	}

	return exifTool.Close()
}

// Interface guards.
var (
	_ gotenberg.Module      = (*ExifTool)(nil)
	_ gotenberg.Provisioner = (*ExifTool)(nil)
	_ gotenberg.Validator   = (*ExifTool)(nil)
	_ gotenberg.PdfEngine   = (*ExifTool)(nil)
)
