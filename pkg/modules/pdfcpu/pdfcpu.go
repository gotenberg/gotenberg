package pdfcpu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(PdfCpu))
}

// PdfCpu abstracts the CLI tool pdfcpu and implements the
// [gotenberg.PdfEngine] interface.
type PdfCpu struct {
	binPath string
}

type pdfcpuBookmark struct {
	Title    string           `json:"title"`
	Page     int              `json:"page"`
	Children []pdfcpuBookmark `json:"kids,omitempty"`
}

type pdfcpuBookmarks struct {
	Bookmarks []pdfcpuBookmark `json:"bookmarks"`
}

// Descriptor returns a [PdfCpu]'s module descriptor.
func (engine *PdfCpu) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "pdfcpu",
		New: func() gotenberg.Module { return new(PdfCpu) },
	}
}

// Provision sets the engine properties.
func (engine *PdfCpu) Provision(ctx *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("PDFCPU_BIN_PATH")
	if !ok {
		return errors.New("PDFCPU_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine *PdfCpu) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("pdfcpu binary path does not exist: %w", err)
	}

	return nil
}

// Debug returns additional debug data.
func (engine *PdfCpu) Debug() map[string]any {
	debug := make(map[string]any)

	cmd := exec.Command(engine.binPath, "version") //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	output, err := cmd.Output()
	if err != nil {
		debug["version"] = err.Error()
		return debug
	}

	debug["version"] = "Unable to determine pdfcpu version"

	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if after, ok := strings.CutPrefix(line, "pdfcpu:"); ok {
			debug["version"] = strings.TrimSpace(after)
			break
		}
	}

	return debug
}

// Merge combines multiple PDFs into a single PDF.
func (engine *PdfCpu) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	args := make([]string, 0, 2+len(inputPaths))
	args = append(args, "merge", outputPath)
	args = append(args, inputPaths...)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with pdfcpu: %w", err)
}

// Split splits a given PDF file.
func (engine *PdfCpu) Split(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
	var args []string

	switch mode.Mode {
	case gotenberg.SplitModeIntervals:
		args = append(args, "split", "-mode", "span", inputPath, outputDirPath, mode.Span)
	case gotenberg.SplitModePages:
		if mode.Unify {
			outputPath := fmt.Sprintf("%s/%s", outputDirPath, filepath.Base(inputPath))
			args = append(args, "trim", "-pages", mode.Span, inputPath, outputPath)
			break
		}
		args = append(args, "extract", "-mode", "page", "-pages", mode.Span, inputPath, outputDirPath)
	default:
		return nil, fmt.Errorf("split PDFs using mode '%s' with pdfcpu: %w", mode.Mode, gotenberg.ErrPdfSplitModeNotSupported)
	}

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return nil, fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return nil, fmt.Errorf("split PDFs with pdfcpu: %w", err)
	}

	var outputPaths []string
	err = filepath.Walk(outputDirPath, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			return pathErr
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
			outputPaths = append(outputPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory to find resulting PDFs from split with pdfcpu: %w", err)
	}

	sort.Sort(digitSuffixSort(outputPaths))

	return outputPaths, nil
}

// Flatten is not available in this implementation.
func (engine *PdfCpu) Flatten(ctx context.Context, logger *zap.Logger, inputPath string) error {
	return fmt.Errorf("flatten PDF with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// Convert is not available in this implementation.
func (engine *PdfCpu) Convert(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
	return fmt.Errorf("convert PDF to '%+v' with pdfcpu: %w", formats, gotenberg.ErrPdfEngineMethodNotSupported)
}

// ReadMetadata is not available in this implementation.
func (engine *PdfCpu) ReadMetadata(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]any, error) {
	return nil, fmt.Errorf("read PDF metadata with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// WriteMetadata is not available in this implementation.
func (engine *PdfCpu) WriteMetadata(ctx context.Context, logger *zap.Logger, metadata map[string]any, inputPath string) error {
	return fmt.Errorf("write PDF metadata with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// PageCount is not available in this implementation.
func (engine *PdfCpu) PageCount(ctx context.Context, logger *zap.Logger, inputPath string) (int, error) {
	return 0, fmt.Errorf("page count with pdfcpu: %w", gotenberg.ErrPdfEngineMethodNotSupported)
}

// ReadBookmarks reads the document outline (bookmarks) of a PDF file using pdfcpu.
func (engine *PdfCpu) ReadBookmarks(ctx context.Context, logger *zap.Logger, inputPath string) ([]gotenberg.Bookmark, error) {
	tmpPath := fmt.Sprintf("%s.read.json", inputPath)
	args := []string{"bookmarks", "export", inputPath, tmpPath}
	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return nil, fmt.Errorf("create command: %w", err)
	}

	defer func() {
		err := os.Remove(tmpPath)
		if err != nil && !os.IsNotExist(err) {
			logger.Error(fmt.Sprintf("remove temporary bookmarks JSON file: %v", err))
		}
	}()

	_, cmdErr := cmd.Exec()

	// Check file existence and size.
	info, statErr := os.Stat(tmpPath)

	if cmdErr != nil {
		// If the file wasn't created, or it was created but is 0 bytes,
		// it means pdfcpu had no bookmarks to write.
		if os.IsNotExist(statErr) || (statErr == nil && info.Size() == 0) {
			return make([]gotenberg.Bookmark, 0), nil
		}

		// Fallback: Check the error string just in case pdfcpu failed without
		// touching the file.
		if strings.Contains(strings.ToLower(cmdErr.Error()), "no bookmarks") {
			return make([]gotenberg.Bookmark, 0), nil
		}
		return nil, fmt.Errorf("read bookmarks with pdfcpu: %w", cmdErr)
	}

	// If cmd succeeded, but output a 0-byte file anyway.
	if info != nil && info.Size() == 0 {
		return make([]gotenberg.Bookmark, 0), nil
	}

	// Read the file content.
	jsonBytes, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("read temporary bookmarks JSON file: %w", err)
	}

	// Check if the content is just empty whitespace.
	if len(bytes.TrimSpace(jsonBytes)) == 0 {
		return make([]gotenberg.Bookmark, 0), nil
	}

	var data pdfcpuBookmarks
	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal bookmarks: %w", err)
	}

	// Safety check: Does the parsed JSON actually contain bookmarks?
	if len(data.Bookmarks) == 0 {
		return make([]gotenberg.Bookmark, 0), nil
	}

	var mapBookmarks func(bookmarks []pdfcpuBookmark) []gotenberg.Bookmark
	mapBookmarks = func(bookmarks []pdfcpuBookmark) []gotenberg.Bookmark {
		res := make([]gotenberg.Bookmark, len(bookmarks))
		for i, b := range bookmarks {
			res[i] = gotenberg.Bookmark{
				Title:    b.Title,
				Page:     b.Page,
				Children: mapBookmarks(b.Children),
			}
		}
		return res
	}

	return mapBookmarks(data.Bookmarks), nil
}

// WriteBookmarks adds a document outline (bookmarks) to a PDF file using pdfcpu.
func (engine *PdfCpu) WriteBookmarks(ctx context.Context, logger *zap.Logger, inputPath string, bookmarks []gotenberg.Bookmark) error {
	if len(bookmarks) == 0 {
		return nil
	}

	var mapBookmarks func(bookmarks []gotenberg.Bookmark) []pdfcpuBookmark
	mapBookmarks = func(bookmarks []gotenberg.Bookmark) []pdfcpuBookmark {
		res := make([]pdfcpuBookmark, len(bookmarks))
		for i, b := range bookmarks {
			res[i] = pdfcpuBookmark{
				Title:    b.Title,
				Page:     b.Page,
				Children: mapBookmarks(b.Children),
			}
		}
		return res
	}

	data := pdfcpuBookmarks{
		Bookmarks: mapBookmarks(bookmarks),
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal bookmarks: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.json", inputPath)
	err = os.WriteFile(tmpPath, jsonBytes, 0o600)
	if err != nil {
		return fmt.Errorf("write temporary bookmarks JSON file: %w", err)
	}

	defer func() {
		err := os.Remove(tmpPath)
		if err != nil {
			logger.Error(fmt.Sprintf("remove temporary bookmarks JSON file: %v", err))
		}
	}()

	args := []string{"bookmarks", "import", "-replace", inputPath, tmpPath, inputPath}
	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("write bookmarks with pdfcpu: %w", err)
	}

	return nil
}

// EmbedFiles embeds files into a PDF. All files are embedded as file attachments
// without modifying the main PDF content.
func (engine *PdfCpu) EmbedFiles(ctx context.Context, logger *zap.Logger, filePaths []string, inputPath string) error {
	if len(filePaths) == 0 {
		return nil
	}

	logger.Debug(fmt.Sprintf("embedding %d file(s) to %s: %v", len(filePaths), inputPath, filePaths))

	args := make([]string, 0, 3+len(filePaths))
	args = append(args, "attachments", "add", inputPath)
	args = append(args, filePaths...)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command for attaching files: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("attach files with pdfcpu: %w", err)
	}

	return nil
}

// Encrypt adds password protection to a PDF file using pdfcpu.
func (engine *PdfCpu) Encrypt(ctx context.Context, logger *zap.Logger, inputPath, userPassword, ownerPassword string) error {
	if userPassword == "" {
		return errors.New("user password cannot be empty")
	}

	if ownerPassword == "" {
		ownerPassword = userPassword
	}

	args := make([]string, 0, 11)
	args = append(args, "encrypt")
	args = append(args, "-mode", "aes")
	args = append(args, "-upw", userPassword)
	args = append(args, "-opw", ownerPassword)
	args = append(args, "-perm", "all")
	args = append(args, inputPath, inputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("encrypt PDF with pdfcpu: %w", err)
	}

	return nil
}

// Interface guards.
var (
	_ gotenberg.Module      = (*PdfCpu)(nil)
	_ gotenberg.Provisioner = (*PdfCpu)(nil)
	_ gotenberg.Validator   = (*PdfCpu)(nil)
	_ gotenberg.Debuggable  = (*PdfCpu)(nil)
	_ gotenberg.PdfEngine   = (*PdfCpu)(nil)
)
