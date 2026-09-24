package api

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildWorkbook packs entries into an in-memory xlsx-like zip.
func buildWorkbook(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry %q: %v", name, err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
	err := w.Close()
	if err != nil {
		t.Fatalf("close workbook: %v", err)
	}
	return buf.Bytes()
}

func readEntry(t *testing.T, workbook []byte, name string) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(workbook), int64(len(workbook)))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %q: %v", name, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read entry %q: %v", name, err)
		}
		return string(data)
	}
	t.Fatalf("entry %q not found", name)
	return ""
}

const scrolledSheet = `<?xml version="1.0"?><worksheet><dimension ref="A1:B83"/>` +
	`<sheetViews><sheetView tabSelected="1" topLeftCell="A37" workbookViewId="0">` +
	`<pane topLeftCell="A37"/><selection activeCell="A1" sqref="A1"/></sheetView></sheetViews>` +
	`<sheetData><row r="1"><c r="A1"><v>1</v></c></row></sheetData></worksheet>`

const topSheet = `<?xml version="1.0"?><worksheet><dimension ref="A1:B83"/>` +
	`<sheetViews><sheetView tabSelected="1" workbookViewId="0"/></sheetViews>` +
	`<sheetData><row r="1"><c r="A1"><v>1</v></c></row></sheetData></worksheet>`

func TestStripWorksheetScrollPosition(t *testing.T) {
	for _, tc := range []struct {
		scenario     string
		workbook     []byte
		expectErr    bool
		expectChange bool
	}{
		{
			scenario: "removes topLeftCell from sheetView and pane",
			workbook: buildWorkbook(t, map[string]string{
				"[Content_Types].xml":      "<Types/>",
				"xl/worksheets/sheet1.xml": scrolledSheet,
				"xl/sharedStrings.xml":     "<sst/>",
			}),
			expectChange: true,
		},
		{
			scenario: "leaves a workbook without a saved scroll position untouched",
			workbook: buildWorkbook(t, map[string]string{
				"[Content_Types].xml":      "<Types/>",
				"xl/worksheets/sheet1.xml": topSheet,
			}),
			expectChange: false,
		},
		{
			scenario: "only rewrites worksheet entries",
			workbook: buildWorkbook(t, map[string]string{
				"xl/worksheets/sheet1.xml": scrolledSheet,
				// A stray topLeftCell elsewhere must not be touched.
				"xl/workbook.xml": `<workbook topLeftCell="A9"/>`,
			}),
			expectChange: true,
		},
		{
			scenario:  "rejects a non-zip input",
			workbook:  []byte("not a zip file"),
			expectErr: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			out, changed, err := stripWorksheetScrollPosition(tc.workbook)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if changed != tc.expectChange {
				t.Fatalf("expected changed=%v, got %v", tc.expectChange, changed)
			}
			if !changed {
				return
			}

			// The rewrite must round-trip and hold the same entries.
			if err = validateWorkbook(tc.workbook, out); err != nil {
				t.Fatalf("rewritten workbook did not validate: %v", err)
			}
			if strings.Contains(readEntry(t, out, "xl/worksheets/sheet1.xml"), "topLeftCell") {
				t.Fatalf("worksheet still contains topLeftCell")
			}
			// Non-worksheet entries are copied verbatim.
			if _, ok := entryNames(t, out)["xl/workbook.xml"]; ok {
				if got := readEntry(t, out, "xl/workbook.xml"); got != `<workbook topLeftCell="A9"/>` {
					t.Fatalf("non-worksheet entry was modified: %q", got)
				}
			}
		})
	}
}

func entryNames(t *testing.T, workbook []byte) map[string]struct{} {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(workbook), int64(len(workbook)))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	names := make(map[string]struct{}, len(r.File))
	for _, f := range r.File {
		names[f.Name] = struct{}{}
	}
	return names
}

func TestResetCalcScrollPosition(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	writeFile := func(t *testing.T, name string, content []byte) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), name)
		err := os.WriteFile(path, content, 0o600)
		if err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
		return path
	}

	t.Run("non-xlsx input is returned unchanged", func(t *testing.T) {
		path := writeFile(t, "input.docx", []byte("whatever"))
		if got := resetCalcScrollPosition(ctx, logger, path); got != path {
			t.Fatalf("expected %q, got %q", path, got)
		}
	})

	t.Run("workbook without a scroll position is returned unchanged", func(t *testing.T) {
		path := writeFile(t, "input.xlsx", buildWorkbook(t, map[string]string{
			"xl/worksheets/sheet1.xml": topSheet,
		}))
		if got := resetCalcScrollPosition(ctx, logger, path); got != path {
			t.Fatalf("expected original path %q, got %q", path, got)
		}
	})

	t.Run("corrupt xlsx falls back to the original path", func(t *testing.T) {
		path := writeFile(t, "input.xlsx", []byte("PK\x03\x04 not really a zip"))
		if got := resetCalcScrollPosition(ctx, logger, path); got != path {
			t.Fatalf("expected fallback to %q, got %q", path, got)
		}
	})

	t.Run("scrolled workbook yields a sanitized copy", func(t *testing.T) {
		path := writeFile(t, "input.xlsx", buildWorkbook(t, map[string]string{
			"[Content_Types].xml":      "<Types/>",
			"xl/worksheets/sheet1.xml": scrolledSheet,
		}))
		got := resetCalcScrollPosition(ctx, logger, path)
		if got == path {
			t.Fatalf("expected a sanitized copy, got the original path")
		}
		if filepath.Dir(got) != filepath.Dir(path) {
			t.Fatalf("sanitized copy escaped the working directory: %q", got)
		}
		sanitized, err := os.ReadFile(got)
		if err != nil {
			t.Fatalf("read sanitized file: %v", err)
		}
		if strings.Contains(readEntry(t, sanitized, "xl/worksheets/sheet1.xml"), "topLeftCell") {
			t.Fatalf("sanitized worksheet still contains topLeftCell")
		}
	})
}
