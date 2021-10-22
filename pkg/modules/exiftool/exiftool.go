package exiftool

import (
	"context"
	"errors"
	"fmt"
	exiftoolLib "github.com/barasher/go-exiftool"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
	"os"
)

func init() {
	gotenberg.MustRegisterModule(Exiftool{})
}

// MetadataValueTypeError is constructed when metadata value types cannot be processed.
// The underlying library used in this implementation supports the writing of a limited
// number of metadata value types.
//
// For example, according to https://exiftool.org/TagNames/PDF.html#Reference metadata can be a boolean,
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

// Exiftool is a module which provides an API to interact with exiftool
// via the go-exiftool library.
type Exiftool struct {
	binPath  string
	exiftool *exiftoolLib.Exiftool
}

// FileMetadata is a wrapper to identify which file the metadata came from.
type FileMetadata struct {
	path     string
	metadata map[string]interface{}
}

// API is an abstraction on top of go-exiftool.
//
// See https://pkg.go.dev/github.com/barasher/go-exiftool
type API interface {
	ReadMetadata(ctx context.Context, logger *zap.Logger, paths []string) (*[]FileMetadata, error)
	WriteMetadata(ctx context.Context, logger *zap.Logger, paths []string, newMetadata *map[string]interface{}) error
}

// Provider is a module interface which exposes a method for creating an API
// for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(exiftool.Provider))
//		exiftool, _ := provider.(exiftool.Provider).Exiftool()
//	}
type Provider interface {
	Exiftool() (API, error)
}

// Descriptor returns Exiftool's module descriptor.
func (Exiftool) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "exif",
		New: func() gotenberg.Module { return new(Exiftool) },
	}
}

// Provision sets the module properties. It returns an error if
// - the environment variable EXIF_BIN_PATH is not set
// - there is an error creating an instance of exiftool.Exiftool
func (mod *Exiftool) Provision(_ *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("EXIF_BIN_PATH")
	if !ok {
		return errors.New("EXIF_BIN_PATH environment variable is not set")
	}
	mod.binPath = binPath

	tool, err := exiftoolLib.NewExiftool(exiftoolLib.SetExiftoolBinaryPath(mod.binPath))
	if err != nil {
		return fmt.Errorf("unable to create NewExiftool: %w", err)
	}
	mod.exiftool = tool

	return nil
}

// Validate validates the module properties. This involves
// - verifying that the exiftool binary path is present and valid
// - verifying the existence of an instance of the underlying go-exiftool library
func (mod Exiftool) Validate() error {
	_, err := os.Stat(mod.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("exif binary path does not exist: %w", err)
	}

	if mod.exiftool == nil {
		return fmt.Errorf("pointer to Exif library does not exist: %w", err)
	}

	return nil
}

// Exiftool returns an API for interacting with exiftool.
func (mod Exiftool) Exiftool() (API, error) {
	return mod, nil
}

// ReadMetadata reads document metadata from the files located by the paths input parameter.
func (mod Exiftool) ReadMetadata(ctx context.Context, logger *zap.Logger, paths []string) (*[]FileMetadata, error) {

	extractedMetadata, err := extractMetadata(paths, mod.exiftool, logger)
	if err != nil {
		return nil, err
	}

	// Converts library struct to an array of FileMetadata structs.
	// This abstraction makes it easier to replace the underlying exiftool API.
	outputFileMetadata := make([]FileMetadata, len(paths))
	for i, metadata := range *extractedMetadata {
		outputFileMetadata[i] = FileMetadata{
			path:     metadata.File,
			metadata: metadata.Fields,
		}
	}

	return &outputFileMetadata, nil
}

// extractMetadata extracts metadata from files specified by the paths input parameter.
// It returns a pointer to an instance of the go-exiftool's exiftool.FileMetadata.
func extractMetadata(paths []string, exiftool *exiftoolLib.Exiftool, logger *zap.Logger) (*[]exiftoolLib.FileMetadata, error) {
	extractedMetadata := exiftool.ExtractMetadata(paths...)
	for _, metadata := range extractedMetadata {
		if metadata.Err != nil {
			return nil, fmt.Errorf("error reading metadata from %s: %w", metadata.File, metadata.Err)
		}
		logger.Debug(fmt.Sprintf("extracted metadata %s from %s", metadata, metadata.File))
	}

	return &extractedMetadata, nil
}

// WriteMetadata appends metadata specified by the pointer input parameter (newMetadata)
// to files located by the paths input parameter.
func (mod Exiftool) WriteMetadata(ctx context.Context, logger *zap.Logger, paths []string, newMetadata *map[string]interface{}) error {
	fileMetadata, err := extractMetadata(paths, mod.exiftool, logger)
	if err != nil {
		return err
	}

	// Metadata values can only be specific value types.
	// An error is returned if metadata with an invalid type is requested.
	metadataValueErrors := MetadataValueTypeError{
		Entries: make(map[string]interface{}),
	}
	for _, metadata := range *fileMetadata {
		for key, value := range *newMetadata {
			switch value.(type) {
			case string:
				metadata.SetString(key, value.(string))
			case int:
				metadata.SetInt(key, int64(value.(int)))
			case int64:
				metadata.SetInt(key, value.(int64))
			case float32:
				metadata.SetFloat(key, float64(value.(float32)))
			case float64:
				metadata.SetFloat(key, value.(float64))
			case []string:
				metadata.SetStrings(key, value.([]string))
			// TODO: support more complex cases, e.g. arrays and nested objects (limitations in underlying library)
			default:
				metadataValueErrors.Entries[key] = value
			}
		}
		logger.Debug(fmt.Sprintf("writing metadata %s to %s", metadata.Fields, metadata.File))
	}

	if len(metadataValueErrors.Entries) > 0 {
		return &metadataValueErrors
	}

	mod.exiftool.WriteMetadata(*fileMetadata)

	return nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*Exiftool)(nil)
	_ gotenberg.Provisioner = (*Exiftool)(nil)
	_ gotenberg.Validator   = (*Exiftool)(nil)
	_ API                   = (*Exiftool)(nil)
	_ Provider              = (*Exiftool)(nil)
)
