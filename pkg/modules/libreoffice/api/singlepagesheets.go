package api

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// topLeftCellAttr matches the topLeftCell attribute that an OOXML worksheet
// uses (on <sheetView> and, for frozen panes, <pane>) to store the cell that
// was at the top-left of the window when the workbook was saved.
var topLeftCellAttr = regexp.MustCompile(` topLeftCell="[^"]*"`)

// maxDecompressedWorksheet bounds how much a single worksheet may decompress to
// while rewriting it. It guards against a decompression bomb and keeps memory
// predictable. A worksheet larger than this is left untouched, so a pathological
// workbook falls back to the original file rather than being rewritten.
const maxDecompressedWorksheet = 128 << 20 // 128 MiB

// resetCalcScrollPosition returns a path to a copy of inputPath whose worksheet
// scroll positions have been reset to the top-left cell, or inputPath unchanged
// when the reset does not apply or cannot be performed safely.
//
// LibreOffice's SinglePageSheets export starts each single page at the sheet's
// saved topLeftCell, dropping every row and column above and to the left of it.
// A workbook saved scrolled away from A1 therefore renders truncated. Removing
// the attribute before the conversion makes the whole used range render.
// See https://github.com/gotenberg/gotenberg/issues/1222.
//
// The function never fails the conversion. On a non-xlsx input, a workbook that
// carries no scroll position, or any read, rewrite or validation error, it
// returns the original path so a malformed rewrite can never reach LibreOffice.
func resetCalcScrollPosition(ctx context.Context, logger *slog.Logger, inputPath string) string {
	// Resolve the extension to a literal so the sanitized filename is never
	// derived from the (user-controlled) upload name.
	var ext string
	switch strings.ToLower(filepath.Ext(inputPath)) {
	case ".xlsx":
		ext = ".xlsx"
	case ".xlsm":
		ext = ".xlsm"
	default:
		return inputPath
	}

	src, err := os.ReadFile(inputPath)
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("reset calc scroll position: read input: %s; using the original file", err))
		return inputPath
	}

	out, changed, err := stripWorksheetScrollPosition(src)
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("reset calc scroll position: %s; using the original file", err))
		return inputPath
	}
	if !changed {
		// The common case: nothing was saved scrolled, so nothing to do.
		return inputPath
	}

	// A rewrite that dropped, renamed or corrupted an entry must never reach
	// LibreOffice; fall back to the original workbook if it does not round-trip.
	if err = validateWorkbook(src, out); err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("reset calc scroll position: %s; using the original file", err))
		return inputPath
	}

	// Write the sanitized copy alongside the input, inside the request working
	// directory that LibreOffice already reads from. The pattern is constant,
	// so the resulting name carries no user-controlled path component.
	dst, err := os.CreateTemp(filepath.Dir(inputPath), "singlepagesheets-*"+ext)
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("reset calc scroll position: create sanitized file: %s; using the original file", err))
		return inputPath
	}
	defer dst.Close()

	_, err = dst.Write(out)
	if err != nil {
		_ = os.Remove(dst.Name()) //nolint:gosec // G703: dst is from os.CreateTemp in the working directory, not a caller-controlled path
		logger.WarnContext(ctx, fmt.Sprintf("reset calc scroll position: write sanitized file: %s; using the original file", err))
		return inputPath
	}

	logger.DebugContext(ctx, "reset calc scroll position: cleared worksheet topLeftCell for SinglePageSheets export")
	return dst.Name()
}

// stripWorksheetScrollPosition rewrites the worksheet XML entries of an xlsx
// workbook, removing the topLeftCell attribute, and reports whether anything
// changed. Every non-worksheet entry, and every worksheet that does not carry
// the attribute, is copied byte-for-byte without recompression.
func stripWorksheetScrollPosition(src []byte) ([]byte, bool, error) {
	reader, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, false, fmt.Errorf("open workbook: %w", err)
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	changed := false

	for _, file := range reader.File {
		rewritten, ok, err := rewriteWorksheet(file)
		if err != nil {
			return nil, false, err
		}

		if ok {
			// Recompress only the worksheets that actually changed.
			header := file.FileHeader
			header.Method = zip.Deflate
			w, err := writer.CreateHeader(&header)
			if err != nil {
				return nil, false, fmt.Errorf("write worksheet %q: %w", file.Name, err)
			}
			_, err = w.Write(rewritten)
			if err != nil {
				return nil, false, fmt.Errorf("write worksheet %q: %w", file.Name, err)
			}
			changed = true
			continue
		}

		err = copyZipEntry(writer, file)
		if err != nil {
			return nil, false, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, false, fmt.Errorf("finalize workbook: %w", err)
	}
	if !changed {
		return nil, false, nil
	}
	return buf.Bytes(), true, nil
}

// rewriteWorksheet returns file's contents with topLeftCell removed, and
// whether file is a worksheet that carried the attribute. A worksheet without
// the attribute, or any other entry, returns ok false so the caller copies it
// verbatim.
func rewriteWorksheet(file *zip.File) ([]byte, bool, error) {
	if !strings.HasPrefix(file.Name, "xl/worksheets/") || !strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
		return nil, false, nil
	}

	rc, err := file.Open()
	if err != nil {
		return nil, false, fmt.Errorf("open worksheet %q: %w", file.Name, err)
	}
	defer rc.Close()

	// Read at most maxDecompressedWorksheet+1 bytes so a decompression bomb
	// cannot exhaust memory; a genuine overflow aborts the rewrite.
	data, err := io.ReadAll(io.LimitReader(rc, maxDecompressedWorksheet+1))
	if err != nil {
		return nil, false, fmt.Errorf("read worksheet %q: %w", file.Name, err)
	}
	if len(data) > maxDecompressedWorksheet {
		return nil, false, fmt.Errorf("worksheet %q exceeds %d bytes", file.Name, maxDecompressedWorksheet)
	}

	if !bytes.Contains(data, []byte("topLeftCell")) {
		return nil, false, nil
	}
	return topLeftCellAttr.ReplaceAll(data, nil), true, nil
}

// copyZipEntry writes file into writer without decompressing and recompressing
// it, preserving its exact bytes.
func copyZipEntry(writer *zip.Writer, file *zip.File) error {
	w, err := writer.CreateRaw(&file.FileHeader)
	if err != nil {
		return fmt.Errorf("copy entry %q: %w", file.Name, err)
	}
	rc, err := file.OpenRaw()
	if err != nil {
		return fmt.Errorf("open entry %q: %w", file.Name, err)
	}
	_, err = io.Copy(w, rc)
	if err != nil {
		return fmt.Errorf("copy entry %q: %w", file.Name, err)
	}
	return nil
}

// validateWorkbook checks that out reopens as a zip holding exactly the same
// entry names as src, rejecting a rewrite that lost, renamed or added an entry
// or produced a broken central directory. The entry payloads themselves are
// not re-read: unchanged entries are copied byte-for-byte from a workbook that
// already parsed, and rewritten worksheets are produced by the standard library
// writer, so re-decompressing everything would only add a decompression-bomb
// surface without catching a failure this transform can introduce.
func validateWorkbook(src, out []byte) error {
	original, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return fmt.Errorf("reopen original workbook: %w", err)
	}
	rewritten, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		return fmt.Errorf("reopen rewritten workbook: %w", err)
	}

	if len(rewritten.File) != len(original.File) {
		return fmt.Errorf("entry count changed from %d to %d", len(original.File), len(rewritten.File))
	}

	names := make(map[string]struct{}, len(original.File))
	for _, file := range original.File {
		names[file.Name] = struct{}{}
	}
	for _, file := range rewritten.File {
		_, ok := names[file.Name]
		if !ok {
			return fmt.Errorf("unexpected entry %q", file.Name)
		}
	}
	return nil
}
