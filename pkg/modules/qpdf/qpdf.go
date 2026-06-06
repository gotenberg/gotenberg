package qpdf

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// qpdfPermissionArgs maps PDF permissions to QPDF --encrypt restriction flags.
// It returns nil when every permission is allowed, matching QPDF's default
// (all actions permitted).
func qpdfPermissionArgs(p gotenberg.PdfPermissions) []string {
	if !p.Restricted() {
		return nil
	}

	yn := func(allowed bool) string {
		if allowed {
			return "y"
		}
		return "n"
	}

	print := "full"
	if !p.AllowPrinting {
		print = "none"
	}

	return []string{
		"--print=" + print,
		"--extract=" + yn(p.AllowCopying),
		"--modify-other=" + yn(p.AllowModifying),
		"--annotate=" + yn(p.AllowAnnotating),
		"--form=" + yn(p.AllowFillingForms),
		"--assemble=" + yn(p.AllowAssembling),
	}
}

func (engine *QPdf) Encrypt(ctx context.Context, logger *slog.Logger, inputPath string, opts gotenberg.EncryptOptions) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.Encrypt",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	ownerPassword := opts.OwnerPassword
	if ownerPassword == "" {
		ownerPassword = opts.UserPassword
	}

	// An empty user password is allowed: it produces an owner-only document.
	if opts.UserPassword == "" && ownerPassword == "" {
		err := errors.New("at least a user or owner password is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	args := make([]string, 0, 14+len(engine.globalArgs))
	args = append(args, inputPath)
	args = append(args, engine.globalArgs...)
	args = append(args, "--replace-input")
	args = append(args, "--encrypt", opts.UserPassword, ownerPassword, "256")
	args = append(args, qpdfPermissionArgs(opts.Permissions)...)
	args = append(args, "--")

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
// them to a temp file, and applies the update via --update-from-json. extraArgs
// are appended to the QPDF command (e.g., --json-stream-data=inline when the
// update replaces stream data).
func (engine *QPdf) writeAndApplyUpdate(ctx context.Context, logger *slog.Logger, inputPath string, updateObjects map[string]any, extraArgs ...string) error {
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

	updateArgs := make([]string, 0, 5+len(engine.globalArgs)+len(extraArgs))
	updateArgs = append(updateArgs, inputPath)
	updateArgs = append(updateArgs, engine.globalArgs...)
	updateArgs = append(updateArgs, extraArgs...)
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

// facturXNamespaceURI is the Factur-X/ZUGFeRD XMP namespace required by strict
// validators.
const facturXNamespaceURI = "urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#"

// InjectFacturXXMP injects Factur-X/ZUGFeRD XMP metadata into the document-level
// XMP packet (Catalog /Metadata stream) of a PDF/A-3 using QPDF's JSON
// manipulation. It reads the existing XMP packet, splices in the fx
// rdf:Description plus the PDF/A extension-schema declaration, and writes the
// stream back uncompressed so the document stays PDF/A-valid.
//
// It assumes the input already carries a Catalog /Metadata stream (always true
// for a LibreOffice PDF/A export). The injection is idempotent: a packet that
// already declares the fx namespace is left untouched.
func (engine *QPdf) InjectFacturXXMP(ctx context.Context, logger *slog.Logger, facturX gotenberg.FacturX, inputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.InjectFacturXXMP",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	err := validateFacturX(facturX)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	logger.DebugContext(ctx, fmt.Sprintf("injecting Factur-X XMP into %s with QPDF", inputPath))

	args := append([]string{inputPath}, engine.globalArgs...)
	args = append(args, "--newline-before-endstream", "--json-output", "--json-stream-data=inline")

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

	metaKey, metaDict, xmp, err := findMetadataStream(objects)
	if err != nil {
		err = fmt.Errorf("locate XMP metadata stream: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	newXmp, changed := injectFacturXIntoXMP(xmp, facturX)
	if !changed {
		logger.DebugContext(ctx, "Factur-X XMP already present, skipping injection")
		span.SetStatus(codes.Ok, "")
		return nil
	}

	// PDF/A requires the metadata stream to be uncompressed and unfiltered. We
	// provide the decoded XMP as the new stream data, so any existing filter
	// must be dropped and the length left for QPDF to recompute.
	delete(metaDict, "/Filter")
	delete(metaDict, "/DecodeParms")
	delete(metaDict, "/Length")

	updateObjects := map[string]any{
		metaKey: map[string]any{
			"stream": map[string]any{
				"dict": metaDict,
				"data": base64.StdEncoding.EncodeToString([]byte(newXmp)),
			},
		},
	}

	err = engine.writeAndApplyUpdate(ctx, logger, inputPath, updateObjects, "--json-stream-data=inline")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ReadPdfAConformance reads the PDF/A part and conformance from the
// document-level XMP packet (pdfaid:part and pdfaid:conformance) using QPDF's
// JSON output. It returns empty strings when the document carries no XMP
// metadata stream or no PDF/A identification.
func (engine *QPdf) ReadPdfAConformance(ctx context.Context, logger *slog.Logger, inputPath string) (string, string, error) {
	ctx, span := gotenberg.Tracer().Start(ctx, "qpdf.ReadPdfAConformance",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(engine.binPath)),
	)
	defer span.End()

	logger.DebugContext(ctx, fmt.Sprintf("reading PDF/A conformance from %s with QPDF", inputPath))

	args := append([]string{inputPath}, engine.globalArgs...)
	args = append(args, "--json-output", "--json-stream-data=inline")

	output, err := engine.execCaptureOutput(ctx, args...)
	if err != nil {
		err = fmt.Errorf("get PDF JSON with QPDF: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}

	objects, err := parsePdfObjects(output)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", "", err
	}

	_, _, xmp, err := findMetadataStream(objects)
	if err != nil {
		// No XMP metadata stream means no PDF/A identification.
		logger.DebugContext(ctx, fmt.Sprintf("no XMP metadata stream in %s: %s", inputPath, err))
		span.SetStatus(codes.Ok, "")
		return "", "", nil
	}

	part, conformance := parsePdfAId(xmp)
	span.SetStatus(codes.Ok, "")
	return part, conformance, nil
}

var (
	pdfaIdPartRe        = regexp.MustCompile(`pdfaid:part[\s>="']*([0-9]+)`)
	pdfaIdConformanceRe = regexp.MustCompile(`pdfaid:conformance[\s>="']*([A-Za-z]+)`)
)

// parsePdfAId extracts the PDF/A part and conformance from an XMP packet. It
// handles both the element (<pdfaid:part>3</pdfaid:part>) and attribute
// (pdfaid:part="3") serializations.
func parsePdfAId(xmp string) (part string, conformance string) {
	if m := pdfaIdPartRe.FindStringSubmatch(xmp); m != nil {
		part = m[1]
	}
	if m := pdfaIdConformanceRe.FindStringSubmatch(xmp); m != nil {
		conformance = m[1]
	}
	return part, conformance
}

// validateFacturX checks the Factur-X fields against the supported values.
func validateFacturX(facturX gotenberg.FacturX) error {
	switch facturX.ConformanceLevel {
	case gotenberg.FacturXConformanceMinimum,
		gotenberg.FacturXConformanceBasicWL,
		gotenberg.FacturXConformanceBasic,
		gotenberg.FacturXConformanceEN16931,
		gotenberg.FacturXConformanceExtended,
		gotenberg.FacturXConformanceXRechnung:
	default:
		return fmt.Errorf("conformance level '%s': %w", facturX.ConformanceLevel, gotenberg.ErrPdfFacturXValueNotSupported)
	}

	switch facturX.DocumentType {
	case gotenberg.FacturXDocumentTypeInvoice,
		gotenberg.FacturXDocumentTypeOrder,
		gotenberg.FacturXDocumentTypeOrderResponse,
		gotenberg.FacturXDocumentTypeOrderChange:
	default:
		return fmt.Errorf("document type '%s': %w", facturX.DocumentType, gotenberg.ErrPdfFacturXValueNotSupported)
	}

	if facturX.DocumentFileName == "" {
		return fmt.Errorf("document file name is empty: %w", gotenberg.ErrPdfFacturXValueNotSupported)
	}

	if facturX.Version == "" {
		return fmt.Errorf("version is empty: %w", gotenberg.ErrPdfFacturXValueNotSupported)
	}

	return nil
}

// findMetadataStream locates the document-level XMP metadata stream referenced
// by the Catalog /Metadata entry. It returns the object key (e.g. "obj:4 0 R"),
// the stream dict, and the decoded XMP packet.
func findMetadataStream(objects map[string]json.RawMessage) (string, map[string]any, string, error) {
	var metadataRef string
	for _, raw := range objects {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		valueRaw, ok := obj["value"]
		if !ok {
			continue
		}

		var value map[string]any
		if err := json.Unmarshal(valueRaw, &value); err != nil {
			continue
		}

		if typeVal, _ := value["/Type"].(string); typeVal == "/Catalog" {
			metadataRef, _ = value["/Metadata"].(string)
			break
		}
	}

	if metadataRef == "" {
		return "", nil, "", errors.New("no /Metadata reference in the catalog")
	}

	// References in values use the "4 0 R" form; object keys use "obj:4 0 R".
	objKey := "obj:" + metadataRef
	raw, ok := objects[objKey]
	if !ok {
		return "", nil, "", fmt.Errorf("metadata object '%s' not found", objKey)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", nil, "", fmt.Errorf("unmarshal metadata object: %w", err)
	}

	streamRaw, ok := obj["stream"]
	if !ok {
		return "", nil, "", errors.New("metadata object is not a stream")
	}

	var stream struct {
		Dict map[string]any `json:"dict"`
		Data string         `json:"data"`
	}
	if err := json.Unmarshal(streamRaw, &stream); err != nil {
		return "", nil, "", fmt.Errorf("unmarshal metadata stream: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(stream.Data)
	if err != nil {
		return "", nil, "", fmt.Errorf("decode metadata stream data: %w", err)
	}

	dict := stream.Dict
	if dict == nil {
		dict = make(map[string]any)
	}

	return objKey, dict, string(decoded), nil
}

// injectFacturXIntoXMP splices the fx rdf:Description and the PDF/A
// extension-schema declaration into an XMP packet. It returns the new packet and
// whether a change was made (false when the fx namespace is already present).
func injectFacturXIntoXMP(xmp string, facturX gotenberg.FacturX) (string, bool) {
	if strings.Contains(xmp, facturXNamespaceURI) {
		return xmp, false
	}

	anchor := strings.LastIndex(xmp, "</rdf:RDF>")
	if anchor == -1 {
		return xmp, false
	}

	insert := facturXDescription(facturX)

	if strings.Contains(xmp, "pdfaExtension:schemas") {
		// An extension-schema bag already exists (e.g. emitted by another tool):
		// splice the fx Description, then append the fx schema entry into the bag.
		spliced := xmp[:anchor] + insert + xmp[anchor:]
		return injectSchemaIntoExistingBag(spliced), true
	}

	// No extension-schema bag yet (the LibreOffice PDF/A case): create the whole
	// container alongside the fx Description.
	insert += facturXExtensionSchema()
	return xmp[:anchor] + insert + xmp[anchor:], true
}

// injectSchemaIntoExistingBag appends the fx schema entry into an existing
// pdfaExtension:schemas bag.
func injectSchemaIntoExistingBag(xmp string) string {
	mi := strings.Index(xmp, "pdfaExtension:schemas")
	if mi == -1 {
		return xmp
	}

	bag := strings.Index(xmp[mi:], "<rdf:Bag")
	if bag == -1 {
		return xmp
	}

	gt := strings.Index(xmp[mi+bag:], ">")
	if gt == -1 {
		return xmp
	}

	pos := mi + bag + gt + 1
	return xmp[:pos] + "\n" + facturXSchemaLi() + xmp[pos:]
}

// facturXDescription builds the fx rdf:Description carrying the runtime values.
func facturXDescription(facturX gotenberg.FacturX) string {
	return fmt.Sprintf(`  <rdf:Description rdf:about="" xmlns:fx="%s">
   <fx:DocumentType>%s</fx:DocumentType>
   <fx:DocumentFileName>%s</fx:DocumentFileName>
   <fx:Version>%s</fx:Version>
   <fx:ConformanceLevel>%s</fx:ConformanceLevel>
  </rdf:Description>
`,
		facturXNamespaceURI,
		xmlEscape(facturX.DocumentType),
		xmlEscape(facturX.DocumentFileName),
		xmlEscape(facturX.Version),
		xmlEscape(facturX.ConformanceLevel),
	)
}

// facturXExtensionSchema builds the rdf:Description that declares the PDF/A
// extension schema for the fx namespace, including the namespace declarations.
func facturXExtensionSchema() string {
	return fmt.Sprintf(`  <rdf:Description rdf:about="" xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/" xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#" xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#">
   <pdfaExtension:schemas>
    <rdf:Bag>
%s
    </rdf:Bag>
   </pdfaExtension:schemas>
  </rdf:Description>
`, facturXSchemaLi())
}

// facturXSchemaLi builds the rdf:li describing the fx schema and its four
// properties. These are fixed schema definitions, not runtime invoice values.
func facturXSchemaLi() string {
	return fmt.Sprintf(`     <rdf:li rdf:parseType="Resource">
      <pdfaSchema:schema>Factur-X PDFA Extension Schema</pdfaSchema:schema>
      <pdfaSchema:namespaceURI>%s</pdfaSchema:namespaceURI>
      <pdfaSchema:prefix>fx</pdfaSchema:prefix>
      <pdfaSchema:property>
       <rdf:Seq>
        <rdf:li rdf:parseType="Resource">
         <pdfaProperty:name>DocumentFileName</pdfaProperty:name>
         <pdfaProperty:valueType>Text</pdfaProperty:valueType>
         <pdfaProperty:category>external</pdfaProperty:category>
         <pdfaProperty:description>name of the embedded XML invoice file</pdfaProperty:description>
        </rdf:li>
        <rdf:li rdf:parseType="Resource">
         <pdfaProperty:name>DocumentType</pdfaProperty:name>
         <pdfaProperty:valueType>Text</pdfaProperty:valueType>
         <pdfaProperty:category>external</pdfaProperty:category>
         <pdfaProperty:description>The type of the embedded Factur-X document</pdfaProperty:description>
        </rdf:li>
        <rdf:li rdf:parseType="Resource">
         <pdfaProperty:name>Version</pdfaProperty:name>
         <pdfaProperty:valueType>Text</pdfaProperty:valueType>
         <pdfaProperty:category>external</pdfaProperty:category>
         <pdfaProperty:description>The actual version of the Factur-X XML schema</pdfaProperty:description>
        </rdf:li>
        <rdf:li rdf:parseType="Resource">
         <pdfaProperty:name>ConformanceLevel</pdfaProperty:name>
         <pdfaProperty:valueType>Text</pdfaProperty:valueType>
         <pdfaProperty:category>external</pdfaProperty:category>
         <pdfaProperty:description>The conformance level of the embedded Factur-X data</pdfaProperty:description>
        </rdf:li>
       </rdf:Seq>
      </pdfaSchema:property>
     </rdf:li>`, facturXNamespaceURI)
}

// xmlEscape escapes a string for safe inclusion in XML character data.
func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
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
