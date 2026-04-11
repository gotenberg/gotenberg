package exiftool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	"github.com/barasher/go-exiftool"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(ExifTool))
}

// safeKeyPattern matches legitimate ExifTool tag names: alphanumeric,
// hyphens, underscores, colons, and periods. Rejects control characters
// (especially \n) that would inject stdin arguments via go-exiftool's
// line-based protocol.
var safeKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9\-_.:]+$`)

// validateMetadataValue rejects metadata values containing characters that
// could inject ExifTool stdin arguments. go-exiftool writes each key/value
// pair as a single line via fmt.Fprintln(stdin, "-"+k+"="+str), so a
// newline in the value splits into a second stdin line that ExifTool
// interprets as an additional argument. An attacker who controls a
// metadata value but not the key could otherwise inject pseudo-tags like
// -FileName=, -Directory=, -SymLink=, or -HardLink= and trigger arbitrary
// filesystem side effects. The same applies to carriage returns and NUL.
//
// The returned error wraps [gotenberg.ErrPdfEngineMetadataValueNotSupported]
// so the API layer surfaces it as HTTP 400.
func validateMetadataValue(key, value string) error {
	if strings.ContainsAny(value, "\n\r\x00") {
		return fmt.Errorf("write PDF metadata with ExifTool: invalid metadata value for key %q (contains control character): %w", key, gotenberg.ErrPdfEngineMetadataValueNotSupported)
	}
	return nil
}

// systemTags lists ExifTool tags that reflect internal filesystem state
// rather than actual PDF metadata. These are stripped from both read and
// write operations.
var systemTags = []string{
	"FileName",            // Reflects UUID-based disk name, not original filename
	"Directory",           // Leaks internal temp path
	"FileSize",            // System attribute
	"FileModifyDate",      // System attribute
	"FileAccessDate",      // System attribute
	"FileInodeChangeDate", // System attribute
	"FilePermissions",     // System attribute
	"ExifToolVersion",     // Tool metadata
	"Error",               // Extraction error messages
	"Warning",             // Extraction warning messages
}

// writeOnlyDerivedTags lists ExifTool tags that are safe to return when
// reading metadata but should not be written back (writing them can break
// PDF/A compliance or cause side effects).
var writeOnlyDerivedTags = []string{
	"PageCount",         // Causes prism:pageCount injection
	"Linearized",        // Computed status; writing it may invalidate structure
	"PDFVersion",        // Header version; should not be manually forced via metadata
	"MIMEType",          // Read-only derived
	"FileType",          // Read-only derived
	"FileTypeExtension", // Read-only derived
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

// Debug returns additional debug data.
func (engine *ExifTool) Debug() map[string]any {
	debug := make(map[string]any)

	cmd := exec.Command(engine.binPath, "-ver") //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	output, err := cmd.Output()
	if err != nil {
		debug["version"] = err.Error()
		return debug
	}

	debug["version"] = strings.TrimSpace(string(output))
	return debug
}

// Merge is not available in this implementation.
func (engine *ExifTool) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Merge",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("merge PDFs with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Split is not available in this implementation.
func (engine *ExifTool) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Split",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("split PDF with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// Flatten is not available in this implementation.
func (engine *ExifTool) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Flatten",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("flatten PDF with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Convert is not available in this implementation.
func (engine *ExifTool) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Convert",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("convert PDF to '%+v' with ExifTool: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadMetadata extracts the metadata of a given PDF file.
func (engine *ExifTool) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.ReadMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	exifTool, err := exiftool.NewExiftool(exiftool.SetExiftoolBinaryPath(engine.binPath))
	if err != nil {
		err = fmt.Errorf("new ExifTool: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	defer func(exifTool *exiftool.Exiftool) {
		err := exifTool.Close()
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("close ExifTool: %v", err))
		}
	}(exifTool)

	fileMetadata := exifTool.ExtractMetadata(inputPath)
	if fileMetadata[0].Err != nil {
		err = fmt.Errorf("read metadata with ExitfTool: %w", fileMetadata[0].Err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Strip system tags that reflect internal filesystem state (e.g.,
	// UUID-based FileName, temp Directory) rather than actual PDF metadata.
	for _, tag := range systemTags {
		delete(fileMetadata[0].Fields, tag)
	}

	span.SetStatus(codes.Ok, "")
	return fileMetadata[0].Fields, nil
}

// WriteMetadata writes the metadata into a given PDF file.
func (engine *ExifTool) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.WriteMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	exifTool, err := exiftool.NewExiftool(exiftool.SetExiftoolBinaryPath(engine.binPath))
	if err != nil {
		err = fmt.Errorf("new ExifTool: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	defer func(exifTool *exiftool.Exiftool) {
		err := exifTool.Close()
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("close ExifTool: %v", err))
		}
	}(exifTool)

	fileMetadata := exifTool.ExtractMetadata(inputPath)
	if fileMetadata[0].Err != nil {
		err = fmt.Errorf("read metadata with ExitfTool: %w", fileMetadata[0].Err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Strip system and derived tags from the existing file metadata so
	// they are not written back (which can break PDF/A compliance or
	// cause side effects).
	for _, tag := range systemTags {
		delete(fileMetadata[0].Fields, tag)
	}
	for _, tag := range writeOnlyDerivedTags {
		delete(fileMetadata[0].Fields, tag)
	}

	// Filter user-supplied metadata to prevent ExifTool pseudo-tags from
	// triggering dangerous side effects like file renames, moves, or link
	// creation. Comparison is case-insensitive because ExifTool processes
	// tag names case-insensitively.
	// See https://exiftool.org/TagNames/Extra.html.
	dangerousTags := []string{
		"FileName",  // Writing this triggers a file rename in ExifTool
		"Directory", // Writing this triggers a file move in ExifTool
		"HardLink",  // Writing this creates a hard link in ExifTool
		"SymLink",   // Writing this creates a symbolic link in ExifTool
	}
	// Reject metadata keys containing characters that could inject ExifTool
	// stdin arguments. ExifTool uses a line-based stdin protocol; a newline
	// in a key splits into a separate argument, enabling flag injection
	// (e.g., -if with Perl eval). Only allow alphanumeric, hyphen, colon,
	// period, and underscore — sufficient for all legitimate tag names.
	for key := range metadata {
		if !safeKeyPattern.MatchString(key) {
			err = fmt.Errorf("write PDF metadata with ExifTool: invalid metadata key %q: %w", key, gotenberg.ErrPdfEngineMetadataValueNotSupported)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	for key := range metadata {
		for _, tag := range dangerousTags {
			if strings.EqualFold(key, tag) {
				delete(metadata, key)
			}
		}
	}

	for key, value := range metadata {
		switch val := value.(type) {
		case string:
			if err = validateMetadataValue(key, val); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
			fileMetadata[0].SetString(key, val)
		case []string:
			for _, s := range val {
				if err = validateMetadataValue(key, s); err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return err
				}
			}
			fileMetadata[0].SetStrings(key, val)
		case []any:
			// See https://github.com/gotenberg/gotenberg/issues/1048.
			strs := make([]string, len(val))
			for i, entry := range val {
				str, ok := entry.(string)
				if !ok {
					err = fmt.Errorf("write PDF metadata with ExifTool: %s %+v %s %w", key, val, reflect.TypeFor[[]any](), gotenberg.ErrPdfEngineMetadataValueNotSupported)
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return err
				}
				if err = validateMetadataValue(key, str); err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return err
				}
				strs[i] = str
			}
			fileMetadata[0].SetStrings(key, strs)
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
			err = fmt.Errorf("write PDF metadata with ExifTool: %s %+v %s %w", key, val, reflect.TypeOf(val), gotenberg.ErrPdfEngineMetadataValueNotSupported)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	exifTool.WriteMetadata(fileMetadata)
	if fileMetadata[0].Err != nil {
		err = fmt.Errorf("write PDF metadata with ExifTool: %w", fileMetadata[0].Err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// PageCount returns the number of pages in a PDF file using ExifTool.
func (engine *ExifTool) PageCount(ctx context.Context, logger *slog.Logger, inputPath string) (int, error) {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.PageCount",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	metadata, err := engine.ReadMetadata(ctx, logger, inputPath)
	if err != nil {
		err = fmt.Errorf("read metadata with ExifTool: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	pageCountValue, ok := metadata["PageCount"]
	if !ok {
		err = errors.New("PageCount not found in metadata")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	switch val := pageCountValue.(type) {
	case int:
		span.SetStatus(codes.Ok, "")
		return val, nil
	case int64:
		span.SetStatus(codes.Ok, "")
		return int(val), nil
	case float64:
		span.SetStatus(codes.Ok, "")
		return int(val), nil
	case string:
		var res int
		_, err := fmt.Sscanf(val, "%d", &res)
		if err != nil {
			err = fmt.Errorf("parse PageCount string '%s': %w", val, err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return 0, err
		}
		span.SetStatus(codes.Ok, "")
		return res, nil
	default:
		err = fmt.Errorf("unexpected PageCount type '%T'", pageCountValue)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
}

// WriteBookmarks is not available in this implementation.
func (engine *ExifTool) WriteBookmarks(ctx context.Context, logger *slog.Logger, inputPath string, bookmarks []gotenberg.Bookmark) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.WriteBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("write PDF bookmarks with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadBookmarks is not available in this implementation.
func (engine *ExifTool) ReadBookmarks(ctx context.Context, logger *slog.Logger, inputPath string) ([]gotenberg.Bookmark, error) {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.ReadBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("read PDF bookmarks with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// Encrypt is not available in this implementation.
func (engine *ExifTool) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Encrypt",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("encrypt PDF using ExifTool: %w", gotenberg.ErrPdfEncryptionNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// EmbedFiles is not available in this implementation.
func (engine *ExifTool) EmbedFiles(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.EmbedFiles",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("embed files with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Watermark is not available in this implementation.
func (engine *ExifTool) Watermark(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Watermark",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("watermark PDF with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Stamp is not available in this implementation.
func (engine *ExifTool) Stamp(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Stamp",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("stamp PDF with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Rotate is not available in this implementation.
func (engine *ExifTool) Rotate(ctx context.Context, logger *slog.Logger, inputPath string, angle int, pages string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.Rotate",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("rotate PDF with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Interface guards.
var (
	_ gotenberg.Module      = (*ExifTool)(nil)
	_ gotenberg.Provisioner = (*ExifTool)(nil)
	_ gotenberg.Validator   = (*ExifTool)(nil)
	_ gotenberg.Debuggable  = (*ExifTool)(nil)
	_ gotenberg.PdfEngine   = (*ExifTool)(nil)
)
