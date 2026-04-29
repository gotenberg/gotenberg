package exiftool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(ExifTool))
}

// safeKeyPattern matches legitimate ExifTool tag names: alphanumeric,
// hyphens, underscores, colons, and periods. The first character may not
// be a hyphen, otherwise exiftool would treat the argv entry as a flag
// rather than a tag assignment. Control characters are implicitly
// rejected because the class is ASCII-only.
var safeKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_.:][a-zA-Z0-9\-_.:]*$`)

// validateMetadataValue rejects metadata values containing NUL, newline,
// or carriage return. NUL terminates C strings and is rejected by
// [exec.Cmd] anyway; newlines and carriage returns are rejected as
// defense in depth against exiftool parsing quirks, even though argv
// invocation is not susceptible to stdin-protocol injection the way
// the previous go-exiftool backend was. The returned error wraps
// [gotenberg.ErrPdfEngineMetadataValueNotSupported] so the API layer
// surfaces it as HTTP 400.
func validateMetadataValue(key, value string) error {
	if strings.ContainsAny(value, "\n\r\x00") {
		return fmt.Errorf("write PDF metadata with ExifTool: invalid metadata value for key %q (contains control character): %w", key, gotenberg.ErrPdfEngineMetadataValueNotSupported)
	}
	return nil
}

// systemTags lists ExifTool tags that reflect internal filesystem state
// or tool identity rather than actual PDF metadata. Stripped from read
// output before returning to the caller.
var systemTags = []string{
	"SourceFile",          // Full path exiftool -j always emits first
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

// dangerousTags lists ExifTool pseudo-tags that trigger filesystem side
// effects (file rename, move, link creation, permission change). Writes
// containing any of these keys are silently dropped before the argv is
// handed to exiftool. The comparison strips group prefixes (e.g.
// "System:FileName" collapses to "FileName") because exiftool treats
// the prefixed and bare forms identically.
//
// See https://exiftool.org/TagNames/Extra.html.
var dangerousTags = []string{
	"FileName",        // Writing this triggers a file rename in ExifTool
	"Directory",       // Writing this triggers a file move in ExifTool
	"HardLink",        // Writing this creates a hard link in ExifTool
	"SymLink",         // Writing this creates a symbolic link in ExifTool
	"FilePermissions", // Writing this changes the file's permissions
}

// isDangerousTag reports whether key matches one of the [dangerousTags]
// after case-insensitive comparison with any group prefix stripped.
func isDangerousTag(key string) bool {
	bare := key
	if i := strings.LastIndex(key, ":"); i >= 0 {
		bare = key[i+1:]
	}
	for _, tag := range dangerousTags {
		if strings.EqualFold(bare, tag) {
			return true
		}
	}
	return false
}

// buildExifToolWriteArgs builds the variadic argv tail for
//
//	exiftool -overwrite_original <args> <path>
//
// from a user-supplied metadata map. Dangerous pseudo-tags are silently
// dropped. Invalid keys (empty, leading dash, control characters) and
// values containing NUL or newlines return an error wrapping
// [gotenberg.ErrPdfEngineMetadataValueNotSupported] so the API layer
// replies with HTTP 400. Supported value kinds: string, []string,
// []any of strings, bool, int, int64, float32, float64.
func buildExifToolWriteArgs(metadata map[string]any) ([]string, error) {
	var args []string
	for key, value := range metadata {
		if isDangerousTag(key) {
			continue
		}
		if !safeKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("write PDF metadata with ExifTool: invalid metadata key %q: %w", key, gotenberg.ErrPdfEngineMetadataValueNotSupported)
		}

		switch val := value.(type) {
		case string:
			if err := validateMetadataValue(key, val); err != nil {
				return nil, err
			}
			args = append(args, fmt.Sprintf("-%s=%s", key, val))
		case []string:
			for _, s := range val {
				if err := validateMetadataValue(key, s); err != nil {
					return nil, err
				}
				args = append(args, fmt.Sprintf("-%s=%s", key, s))
			}
		case []any:
			// See https://github.com/gotenberg/gotenberg/issues/1048.
			for _, entry := range val {
				s, ok := entry.(string)
				if !ok {
					return nil, fmt.Errorf("write PDF metadata with ExifTool: unsupported element type %T in []any for key %q: %w", entry, key, gotenberg.ErrPdfEngineMetadataValueNotSupported)
				}
				if err := validateMetadataValue(key, s); err != nil {
					return nil, err
				}
				args = append(args, fmt.Sprintf("-%s=%s", key, s))
			}
		case bool:
			args = append(args, fmt.Sprintf("-%s=%t", key, val))
		case int:
			args = append(args, fmt.Sprintf("-%s=%d", key, val))
		case int64:
			args = append(args, fmt.Sprintf("-%s=%d", key, val))
		case float32:
			args = append(args, fmt.Sprintf("-%s=%g", key, val))
		case float64:
			args = append(args, fmt.Sprintf("-%s=%g", key, val))
		default:
			return nil, fmt.Errorf("write PDF metadata with ExifTool: unsupported type %T for key %q: %w", value, key, gotenberg.ErrPdfEngineMetadataValueNotSupported)
		}
	}
	return args, nil
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

// ReadMetadata extracts the metadata of a given PDF file by invoking
// the exiftool binary with "-j" (JSON output) and parsing the result.
func (engine *ExifTool) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.ReadMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	cmd := exec.CommandContext(ctx, engine.binPath, "-j", inputPath) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	output, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("read metadata with ExifTool: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var files []map[string]any
	err = json.Unmarshal(output, &files)
	if err != nil {
		err = fmt.Errorf("parse ExifTool JSON output: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if len(files) == 0 {
		err = errors.New("ExifTool returned no file entries")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	metadata := files[0]

	// ExifTool records extraction errors as an "Error" key on the file
	// entry rather than via a non-zero exit code. Surface that back as a
	// Go error before stripping so callers see the real cause.
	if msg, ok := metadata["Error"].(string); ok && msg != "" {
		err = fmt.Errorf("read metadata with ExifTool: %s", msg)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for _, tag := range systemTags {
		delete(metadata, tag)
	}

	span.SetStatus(codes.Ok, "")
	return metadata, nil
}

// WriteMetadata writes the metadata into a given PDF file by invoking
// the exiftool binary with "-overwrite_original -TAG=VALUE ... path".
// ExifTool preserves tags that are not mentioned in the argv, so the
// write is a merge rather than a rewrite.
func (engine *ExifTool) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "exiftool.WriteMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	extraArgs, err := buildExifToolWriteArgs(metadata)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if len(extraArgs) == 0 {
		// Nothing to write after filtering. Treat as success so the
		// caller can move on without a dedicated zero-tag branch.
		span.SetStatus(codes.Ok, "")
		return nil
	}

	args := append([]string{"-overwrite_original"}, extraArgs...)
	args = append(args, inputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create ExifTool command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	exitCode, err := cmd.Exec()
	if err != nil {
		err = fmt.Errorf("write PDF metadata with ExifTool (exit %d): %w", exitCode, err)
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

// EmbedFilesMetadata is not available in this implementation.
func (engine *ExifTool) EmbedFilesMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]map[string]string, inputPath string) error {
	return fmt.Errorf("set embeds metadata with ExifTool: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Interface guards.
var (
	_ gotenberg.Module      = (*ExifTool)(nil)
	_ gotenberg.Provisioner = (*ExifTool)(nil)
	_ gotenberg.Validator   = (*ExifTool)(nil)
	_ gotenberg.Debuggable  = (*ExifTool)(nil)
	_ gotenberg.PdfEngine   = (*ExifTool)(nil)
)
