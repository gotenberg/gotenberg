package qpdf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(QPdf))
}

// QPdf abstracts the CLI tool QPDF and implements the [gotenberg.PdfEngine]
// interface.
type QPdf struct {
	binPath    string
	globalArgs []string
}

// Descriptor returns a [QPdf]'s module descriptor.
func (engine *QPdf) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "qpdf",
		New: func() gotenberg.Module { return new(QPdf) },
	}
}

// Provision sets the module properties.
func (engine *QPdf) Provision(ctx *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("QPDF_BIN_PATH")
	if !ok {
		return errors.New("QPDF_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath
	// Warnings should not cause errors.
	engine.globalArgs = []string{"--warning-exit-0"}

	return nil
}

// Validate validates the module properties.
func (engine *QPdf) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("QPDF binary path does not exist: %w", err)
	}

	return nil
}

// Debug returns additional debug data.
func (engine *QPdf) Debug() map[string]any {
	debug := make(map[string]any)

	cmd := exec.Command(engine.binPath, "--version") //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	output, err := cmd.Output()
	if err != nil {
		debug["version"] = err.Error()
		return debug
	}

	lines := bytes.SplitN(output, []byte("\n"), 2)
	if len(lines) > 0 {
		debug["version"] = string(lines[0])
	} else {
		debug["version"] = "Unable to determine QPDF version"
	}

	return debug
}

// Split splits a given PDF file.
func (engine *QPdf) Split(ctx context.Context, logger *slog.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Split",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	var args []string
	outputPath := fmt.Sprintf("%s/%s", outputDirPath, filepath.Base(inputPath))

	switch mode.Mode {
	case gotenberg.SplitModePages:
		if !mode.Unify {
			err := fmt.Errorf("split PDFs using mode '%s' without unify with QPDF: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		args = append(args, inputPath)
		args = append(args, engine.globalArgs...)
		args = append(args, "--pages", ".", mode.Span)
		args = append(args, "--", outputPath)
	default:
		err := fmt.Errorf("split PDFs using mode '%s' with QPDF: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("split PDFs with QPDF: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return []string{outputPath}, nil
}

// Merge combines multiple PDFs into a single PDF.
func (engine *QPdf) Merge(ctx context.Context, logger *slog.Logger, inputPaths []string, outputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Merge",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	args := make([]string, 0, 4+len(engine.globalArgs)+len(inputPaths))
	args = append(args, "--empty")
	args = append(args, engine.globalArgs...)
	args = append(args, "--pages")
	args = append(args, inputPaths...)
	args = append(args, "--", outputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err == nil {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	err = fmt.Errorf("merge PDFs with QPDF: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Flatten merges annotation appearances with page content, deleting the
// original annotations.
func (engine *QPdf) Flatten(ctx context.Context, logger *slog.Logger, inputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Flatten",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	args := make([]string, 0, 4+len(engine.globalArgs))
	args = append(args, inputPath)
	args = append(args, "--generate-appearances")
	args = append(args, "--flatten-annotations=all")
	args = append(args, "--replace-input")
	args = append(args, engine.globalArgs...)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err == nil {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	err = fmt.Errorf("flatten PDFs with QPDF: %w", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Convert is not available in this implementation.
func (engine *QPdf) Convert(ctx context.Context, logger *slog.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Convert",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("convert PDF to '%+v' with QPDF: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadMetadata is not available in this implementation.
func (engine *QPdf) ReadMetadata(ctx context.Context, logger *slog.Logger, inputPath string) (map[string]any, error) {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.ReadMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("read PDF metadata with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// WriteMetadata is not available in this implementation.
func (engine *QPdf) WriteMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]any, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.WriteMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("write PDF metadata with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// PageCount is not available in this implementation.
func (engine *QPdf) PageCount(ctx context.Context, logger *slog.Logger, inputPath string) (int, error) {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.PageCount",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("page count with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return 0, err
}

// WriteBookmarks is not available in this implementation.
func (engine *QPdf) WriteBookmarks(ctx context.Context, logger *slog.Logger, inputPath string, bookmarks []gotenberg.Bookmark) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.WriteBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("write PDF bookmarks with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// ReadBookmarks is not available in this implementation.
func (engine *QPdf) ReadBookmarks(ctx context.Context, logger *slog.Logger, inputPath string) ([]gotenberg.Bookmark, error) {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.ReadBookmarks",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("read PDF bookmarks with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return nil, err
}

// Encrypt adds password protection to a PDF file using QPDF.
func (engine *QPdf) Encrypt(ctx context.Context, logger *slog.Logger, inputPath, userPassword, ownerPassword string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Encrypt",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	if userPassword == "" {
		err := errors.New("user password cannot be empty")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if ownerPassword == "" {
		ownerPassword = userPassword
	}

	args := make([]string, 0, 7+len(engine.globalArgs))
	args = append(args, inputPath)
	args = append(args, engine.globalArgs...)
	args = append(args, "--replace-input")
	args = append(args, "--encrypt", userPassword, ownerPassword, "256", "--")

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		err = fmt.Errorf("create command: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	_, err = cmd.Exec()
	if err != nil {
		err = fmt.Errorf("encrypt PDF with QPDF: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// EmbedFiles is not available in this implementation.
func (engine *QPdf) EmbedFiles(ctx context.Context, logger *slog.Logger, filePaths []string, inputPath string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.EmbedFiles",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("embed files with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// EmbedFilesMetadata sets metadata on already-embedded files in a PDF using
// QPDF's JSON manipulation. It sets /AFRelationship on Filespec objects,
// /Subtype on EmbeddedFile streams, and ensures the Catalog /AF array
// references the Filespec objects.
func (engine *QPdf) EmbedFilesMetadata(ctx context.Context, logger *slog.Logger, metadata map[string]map[string]string, inputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.EmbedFilesMetadata",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	if len(metadata) == 0 {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	logger.DebugContext(ctx, fmt.Sprintf("setting embeds metadata on %s with QPDF", inputPath))

	args := append([]string{inputPath}, engine.globalArgs...)
	args = append(args, "--newline-before-endstream", "--json-output")

	output, err := engine.execCaptureOutput(ctx, args...)
	if err != nil {
		err = fmt.Errorf("get PDF JSON with QPDF: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	objects, err := parsePdfObjects(output)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	catalogRef, catalogValue, filespecRefs, updateObjects := patchFilespecMetadata(logger, objects, metadata)
	if len(filespecRefs) == 0 {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	patchCatalogAF(catalogRef, catalogValue, filespecRefs, updateObjects)

	err = engine.writeAndApplyUpdate(ctx, logger, inputPath, updateObjects)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// execCaptureOutput runs QPDF and returns its stdout. This uses
// exec.CommandContext directly because gotenberg.Cmd does not support
// capturing stdout (it only pipes to debug logs).
func (engine *QPdf) execCaptureOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, engine.binPath, args...) //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Output()
}

// parsePdfObjects parses QPDF JSON v2 output and returns the objects map.
func parsePdfObjects(output []byte) (map[string]json.RawMessage, error) {
	var pdfJSON struct {
		Qpdf []json.RawMessage `json:"qpdf"`
	}
	if err := json.Unmarshal(output, &pdfJSON); err != nil {
		return nil, fmt.Errorf("parse PDF JSON: %w", err)
	}
	if len(pdfJSON.Qpdf) < 2 {
		return nil, fmt.Errorf("unexpected QPDF JSON structure: expected at least 2 elements")
	}

	var objects map[string]json.RawMessage
	if err := json.Unmarshal(pdfJSON.Qpdf[1], &objects); err != nil {
		return nil, fmt.Errorf("parse QPDF objects: %w", err)
	}

	return objects, nil
}

// patchFilespecMetadata walks QPDF objects to find Filespecs matching the
// metadata keys. It sets /AFRelationship and /Subtype on matching objects
// and returns the catalog reference, catalog value, filespec references,
// and the update objects map.
func patchFilespecMetadata(logger *slog.Logger, objects map[string]json.RawMessage, metadata map[string]map[string]string) (string, map[string]any, []string, map[string]any) {
	updateObjects := make(map[string]any)
	var catalogRef string
	var catalogValue map[string]any
	var filespecRefs []string

	for ref, raw := range objects {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		valueRaw, hasValue := obj["value"]
		if !hasValue {
			continue
		}

		var value map[string]any
		if err := json.Unmarshal(valueRaw, &value); err != nil {
			continue
		}

		typeVal, _ := value["/Type"].(string)

		if typeVal == "/Catalog" {
			catalogRef = ref
			catalogValue = value
		}

		if typeVal == "/Filespec" {
			uf, _ := value["/UF"].(string)
			if uf == "" {
				uf, _ = value["/F"].(string)
			}

			cleanUf := stripQpdfStringPrefix(uf)

			meta, exists := metadata[cleanUf]
			if !exists {
				continue
			}

			if rel, ok := meta["relationship"]; ok {
				value["/AFRelationship"] = "/" + rel
			}

			if mimeType, ok := meta["mimeType"]; ok {
				if ef, ok := value["/EF"].(map[string]any); ok {
					efRef, _ := ef["/F"].(string)
					if efRef != "" {
						setStreamSubtype(logger, objects, updateObjects, efRef, mimeType)
					}
				}
			}

			filespecRefs = append(filespecRefs, ref)
			updateObjects[ref] = map[string]any{"value": value}
		}
	}

	return catalogRef, catalogValue, filespecRefs, updateObjects
}

// patchCatalogAF ensures the Catalog /AF array references all filespec objects.
func patchCatalogAF(catalogRef string, catalogValue map[string]any, filespecRefs []string, updateObjects map[string]any) {
	if catalogRef == "" || catalogValue == nil {
		return
	}

	afSet := make(map[string]bool)
	existingAF, _ := catalogValue["/AF"].([]any)
	for _, r := range existingAF {
		if s, ok := r.(string); ok {
			afSet[s] = true
		}
	}
	for _, ref := range filespecRefs {
		// Object references in values use "9 0 R" format,
		// not the "obj:9 0 R" key format.
		valRef := strings.TrimPrefix(ref, "obj:")
		if !afSet[valRef] {
			existingAF = append(existingAF, valRef)
		}
	}
	catalogValue["/AF"] = existingAF
	updateObjects[catalogRef] = map[string]any{"value": catalogValue}
}

// writeAndApplyUpdate marshals the update objects as QPDF JSON v2, writes
// them to a temp file, and applies the update via --update-from-json.
func (engine *QPdf) writeAndApplyUpdate(ctx context.Context, logger *slog.Logger, inputPath string, updateObjects map[string]any) error {
	updateJSON := map[string]any{
		"qpdf": []any{
			map[string]any{
				"jsonversion":                  2,
				"pushedinheritedpageresources": false,
				"calledgetallpages":            false,
				"maxobjectid":                  0,
			},
			updateObjects,
		},
	}

	jsonBytes, err := json.Marshal(updateJSON)
	if err != nil {
		return fmt.Errorf("marshal update JSON: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(inputPath), "qpdf-embeds-metadata-*.json")
	if err != nil {
		return fmt.Errorf("create temp file for update JSON: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(jsonBytes); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write update JSON: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	updateArgs := make([]string, 0, 5+len(engine.globalArgs))
	updateArgs = append(updateArgs, inputPath)
	updateArgs = append(updateArgs, engine.globalArgs...)
	updateArgs = append(updateArgs, "--newline-before-endstream")
	updateArgs = append(updateArgs, "--update-from-json="+tmpFile.Name())
	updateArgs = append(updateArgs, "--replace-input")

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, updateArgs...)
	if err != nil {
		return fmt.Errorf("create command for JSON update: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("update embeds metadata with QPDF: %w", err)
	}

	return nil
}

// setStreamSubtype finds a stream object by reference and sets the /Subtype
// key in its dict.
func setStreamSubtype(logger *slog.Logger, objects map[string]json.RawMessage, updateObjects map[string]any, ref, mimeType string) {
	objKey := ref
	if !strings.HasPrefix(objKey, "obj:") {
		objKey = "obj:" + objKey
	}
	raw, ok := objects[objKey]
	if !ok {
		logger.Warn(fmt.Sprintf("set stream subtype on %s: object not found", ref))
		return
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		logger.Warn(fmt.Sprintf("set stream subtype on %s: unmarshal object: %s", ref, err))
		return
	}

	streamRaw, ok := obj["stream"]
	if !ok {
		logger.Warn(fmt.Sprintf("set stream subtype on %s: no stream key", ref))
		return
	}

	var stream map[string]any
	if err := json.Unmarshal(streamRaw, &stream); err != nil {
		logger.Warn(fmt.Sprintf("set stream subtype on %s: unmarshal stream: %s", ref, err))
		return
	}

	dict, ok := stream["dict"].(map[string]any)
	if !ok {
		logger.Warn(fmt.Sprintf("set stream subtype on %s: stream dict is not a map", ref))
		return
	}

	// QPDF JSON uses literal name syntax; it handles PDF name
	// encoding internally when writing the binary PDF.
	dict["/Subtype"] = "/" + mimeType
	stream["dict"] = dict
	updateObjects[objKey] = map[string]any{"stream": stream}
}

// stripQpdfStringPrefix removes the type prefix that QPDF adds to JSON
// string values. Known prefixes: "u:" (Unicode), "b:" (binary), "e:" (encoded).
func stripQpdfStringPrefix(s string) string {
	for _, prefix := range []string{"u:", "b:", "e:"} {
		if strings.HasPrefix(s, prefix) {
			return s[len(prefix):]
		}
	}
	return s
}

// Watermark is not available in this implementation.
func (engine *QPdf) Watermark(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Watermark",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("watermark PDF with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Stamp is not available in this implementation.
func (engine *QPdf) Stamp(ctx context.Context, logger *slog.Logger, inputPath string, stamp gotenberg.Stamp) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Stamp",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("stamp PDF with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Rotate is not available in this implementation.
func (engine *QPdf) Rotate(ctx context.Context, logger *slog.Logger, inputPath string, angle int, pages string) error {
	_, span := gotenberg.Tracer().Start(ctx, "qpdf.Rotate",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := fmt.Errorf("rotate PDF with QPDF: %w", gotenberg.ErrPdfEngineMethodNotSupported)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

var (
	_ gotenberg.Module      = (*QPdf)(nil)
	_ gotenberg.Provisioner = (*QPdf)(nil)
	_ gotenberg.Validator   = (*QPdf)(nil)
	_ gotenberg.Debuggable  = (*QPdf)(nil)
	_ gotenberg.PdfEngine   = (*QPdf)(nil)
)
