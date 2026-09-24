package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestConversionRequestAttributes(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "in.docx")
	content := []byte("hello world")
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	options := Options{
		Landscape:  true,
		PageRanges: "1-3",
		PdfFormats: gotenberg.PdfFormats{PdfA: "PDF/A-2b", PdfUa: true},
	}

	got := map[string]any{}
	for _, kv := range conversionRequestAttributes(tmp, options) {
		got[string(kv.Key)] = kv.Value.AsInterface()
	}

	if got["gotenberg.libreoffice.pdf_a"] != "PDF/A-2b" {
		t.Errorf("pdf_a = %v, want PDF/A-2b", got["gotenberg.libreoffice.pdf_a"])
	}
	if got["gotenberg.libreoffice.pdf_ua"] != true {
		t.Errorf("pdf_ua = %v, want true", got["gotenberg.libreoffice.pdf_ua"])
	}
	if got["gotenberg.conversion.landscape"] != true {
		t.Errorf("landscape = %v, want true", got["gotenberg.conversion.landscape"])
	}
	if got["gotenberg.conversion.has_page_ranges"] != true {
		t.Errorf("has_page_ranges = %v, want true", got["gotenberg.conversion.has_page_ranges"])
	}
	if got["gotenberg.conversion.input.bytes"] != int64(len(content)) {
		t.Errorf("input.bytes = %v, want %d", got["gotenberg.conversion.input.bytes"], len(content))
	}
}
