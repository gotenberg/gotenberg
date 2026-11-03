package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dlclark/regexp2/v2"
	"github.com/gotenberg/litho/pkg/models"
	litho "github.com/gotenberg/litho/pkg/pdf"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

const testdata = "../../../test/integration/testdata"

// testContext returns a request context working in a temporary directory, and
// copies the named fixtures into it.
func testContext(t *testing.T, fixtures ...string) (*api.Context, map[string]string) {
	t.Helper()

	dir := t.TempDir()
	mock := &api.ContextMock{Context: &api.Context{Context: context.Background()}}
	mock.SetDirPath(dir)
	mock.SetLogger(slog.New(slog.DiscardHandler))

	paths := make(map[string]string, len(fixtures))
	for _, name := range fixtures {
		src := filepath.Join(testdata, name)
		if strings.HasPrefix(name, "pem/") {
			src = filepath.Join(testdata, name)
		}

		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}

		dst := filepath.Join(dir, filepath.Base(name))
		err = os.WriteFile(dst, data, 0o600)
		if err != nil {
			t.Fatal(err)
		}

		paths[filepath.Base(name)] = dst
	}

	mock.SetFiles(paths)

	return mock.Context, paths
}

func testProcessor() *Processor {
	return &Processor{
		fonts:       litho.WithSystemFonts(),
		renderFonts: litho.WithRenderSystemFonts(),
		httpClient:  func(context.Context) *http.Client { return http.DefaultClient },
	}
}

func mustStages(t *testing.T, raw string) Stages {
	t.Helper()

	stages, err := ParseStages(raw)
	if err != nil {
		t.Fatal(err)
	}

	return stages
}

func openOutput(t *testing.T, path string, opts ...litho.OpenOption) *litho.Reader {
	t.Helper()

	r, err := litho.OpenFile(context.Background(), path, opts...)
	if err != nil {
		t.Fatalf("open output %s: %v", path, err)
	}

	t.Cleanup(func() { _ = r.Close() })

	return r
}

func pageCount(t *testing.T, path string) int {
	t.Helper()

	n, err := openOutput(t, path).PageCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return n
}

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fixtures  []string
		inputs    []string
		stages    string
		wantNames []string
		check     func(t *testing.T, outputs []string)
	}{
		{
			name:      "no stages keeps the inputs",
			fixtures:  []string{"page_1.pdf"},
			inputs:    []string{"page_1.pdf"},
			stages:    `[]`,
			wantNames: []string{"page_1.pdf"},
		},
		{
			name:      "merge in input order, named after the first",
			fixtures:  []string{"page_1.pdf", "page_2.pdf", "pages_12.pdf"},
			inputs:    []string{"page_1.pdf", "page_2.pdf", "pages_12.pdf"},
			stages:    `[{"op":"merge"}]`,
			wantNames: []string{"page_1.pdf"},
			check: func(t *testing.T, outputs []string) {
				if n := pageCount(t, outputs[0]); n != 14 {
					t.Fatalf("merged page count = %d, want 14", n)
				}
			},
		},
		{
			name:      "split every n pages, then edit each part",
			fixtures:  []string{"pages_12.pdf"},
			inputs:    []string{"pages_12.pdf"},
			stages:    `[{"op":"split","every":5},{"op":"set-metadata","title":"Part"}]`,
			wantNames: []string{"pages_12_0.pdf", "pages_12_1.pdf", "pages_12_2.pdf"},
			check: func(t *testing.T, outputs []string) {
				for i, want := range []int{5, 5, 2} {
					if n := pageCount(t, outputs[i]); n != want {
						t.Errorf("part %d page count = %d, want %d", i, n, want)
					}

					meta, err := openOutput(t, outputs[i]).Metadata(context.Background())
					if err != nil || meta["Title"] != "Part" {
						t.Errorf("part %d metadata = %v, %v, want the title set after the split", i, meta, err)
					}
				}
			},
		},
		{
			name:      "split by ranges",
			fixtures:  []string{"pages_12.pdf"},
			inputs:    []string{"pages_12.pdf"},
			stages:    `[{"op":"split","ranges":["1-3","last"]}]`,
			wantNames: []string{"pages_12_0.pdf", "pages_12_1.pdf"},
			check: func(t *testing.T, outputs []string) {
				if a, b := pageCount(t, outputs[0]), pageCount(t, outputs[1]); a != 3 || b != 1 {
					t.Fatalf("part page counts = %d, %d, want 3, 1", a, b)
				}
			},
		},
		{
			name:      "select, delete and insert",
			fixtures:  []string{"pages_12.pdf", "page_1.pdf"},
			inputs:    []string{"pages_12.pdf"},
			stages:    `[{"op":"select","pages":"1-6"},{"op":"delete","pages":"even"},{"op":"insert","source":"page_1.pdf","at":3}]`,
			wantNames: []string{"pages_12.pdf"},
			check: func(t *testing.T, outputs []string) {
				if n := pageCount(t, outputs[0]); n != 4 {
					t.Fatalf("page count = %d, want 6 selected, 3 deleted, 1 inserted", n)
				}
			},
		},
		{
			name:      "rotate and stamps",
			fixtures:  []string{"pages_12.pdf", "watermark.png", "page_2.pdf"},
			inputs:    []string{"pages_12.pdf"},
			stages:    `[{"op":"rotate","angle":90,"pages":"1"},{"op":"stamp-text","text":"DRAFT","pages":"1","color":"#ff0000","opacity":0.5},{"op":"stamp-image","image":"watermark.png","width":50,"height":50},{"op":"stamp-pdf","source":"page_2.pdf","scale":0.5}]`,
			wantNames: []string{"pages_12.pdf"},
			check: func(t *testing.T, outputs []string) {
				page, err := openOutput(t, outputs[0]).Page(context.Background(), 1)
				if err != nil || page.Rotate != 90 {
					t.Fatalf("page = %+v, %v, want rotated by 90", page, err)
				}

				// Stamps are marked as artifacts, which text extraction skips, so
				// compare what the second page draws: it was stamped without
				// being rotated.
				stamped := render(t, outputs[0], 2)
				original := render(t, filepath.Join(filepath.Dir(outputs[0]), "pages_12.pdf"), 2)
				if slices.Equal(stamped, original) {
					t.Fatal("the stamps left the page unchanged")
				}
			},
		},
		{
			name:      "bookmarks and attachments",
			fixtures:  []string{"page_1.pdf", "embed_1.xml"},
			inputs:    []string{"page_1.pdf"},
			stages:    `[{"op":"set-bookmarks","bookmarks":[{"title":"Start","pageNumber":1}]},{"op":"attach","file":"embed_1.xml"}]`,
			wantNames: []string{"page_1.pdf"},
			check: func(t *testing.T, outputs []string) {
				r := openOutput(t, outputs[0])

				bookmarks, err := r.Bookmarks(context.Background())
				if err != nil || len(bookmarks) != 1 || bookmarks[0].Title != "Start" {
					t.Fatalf("bookmarks = %v, %v", bookmarks, err)
				}

				files, err := r.EmbeddedFiles(context.Background())
				if err != nil || len(files) != 1 || files[0].Name != "embed_1.xml" {
					t.Fatalf("embedded files = %v, %v", files, err)
				}
			},
		},
		{
			name:      "encrypt with optimize",
			fixtures:  []string{"page_1.pdf"},
			inputs:    []string{"page_1.pdf"},
			stages:    `[{"op":"optimize"},{"op":"encrypt","userPassword":"open","ownerPassword":"own"}]`,
			wantNames: []string{"page_1.pdf"},
			check: func(t *testing.T, outputs []string) {
				_, err := litho.OpenFile(context.Background(), outputs[0])
				if !errors.Is(err, litho.ErrInvalidPassword) {
					t.Fatalf("open without the password: %v, want ErrInvalidPassword", err)
				}

				if r := openOutput(t, outputs[0], litho.WithPassword("open")); !r.IsEncrypted() {
					t.Fatal("output is not encrypted")
				}
			},
		},
		{
			name:      "remediate to PDF/A-2b",
			fixtures:  []string{"page_1.pdf"},
			inputs:    []string{"page_1.pdf"},
			stages:    `[{"op":"set-metadata","title":"Archived"},{"op":"remediate","profile":"PDF/A-2b"}]`,
			wantNames: []string{"page_1.pdf"},
			check: func(t *testing.T, outputs []string) {
				violations, err := openOutput(t, outputs[0]).ValidatePDFAReport(context.Background(), models.PDFA2b)
				if err != nil || len(violations) > 0 {
					t.Fatalf("PDF/A-2b violations = %v, %v", violations, err)
				}
			},
		},
		{
			name:      "sign with a PEM certificate and key",
			fixtures:  []string{"page_1.pdf", "pem/cert.pem", "pem/key.pem"},
			inputs:    []string{"page_1.pdf"},
			stages:    `[{"op":"sign","certificate":"cert.pem","key":"key.pem","reason":"Approved"}]`,
			wantNames: []string{"page_1.pdf"},
			check: func(t *testing.T, outputs []string) {
				signatures, err := openOutput(t, outputs[0]).ExtractSignatures(context.Background())
				if err != nil || len(signatures) != 1 {
					t.Fatalf("signatures = %v, %v, want one", signatures, err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, tc.fixtures...)

			inputs := make([]string, len(tc.inputs))
			for i, name := range tc.inputs {
				inputs[i] = paths[name]
			}

			stages := mustStages(t, tc.stages)

			files := make(map[string]string)
			for _, name := range stages.Files() {
				files[name] = paths[name]
			}

			outputs, err := testProcessor().Run(ctx, stages, inputs, files)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			names := make([]string, len(outputs))
			for i, out := range outputs {
				names[i] = ctx.OriginalFilename(out)
			}

			if !slices.Equal(names, tc.wantNames) {
				t.Fatalf("output names = %v, want %v", names, tc.wantNames)
			}

			if tc.check != nil {
				tc.check(t, outputs)
			}
		})
	}
}

func TestRunDecryptsInputs(t *testing.T) {
	ctx, paths := testContext(t, "page_1.pdf")
	processor := testProcessor()

	encrypted, err := processor.Run(ctx, mustStages(t, `[{"op":"encrypt","userPassword":"open"}]`), []string{paths["page_1.pdf"]}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = processor.Run(ctx, mustStages(t, `[{"op":"rotate","angle":90}]`), encrypted, nil)
	if !errors.Is(err, litho.ErrInvalidPassword) {
		t.Fatalf("an encrypted input without decrypt: %v, want ErrInvalidPassword", err)
	}

	decrypted, err := processor.Run(ctx, mustStages(t, `[{"op":"decrypt","password":"open"},{"op":"rotate","angle":90}]`), encrypted, nil)
	if err != nil {
		t.Fatal(err)
	}

	if openOutput(t, decrypted[0]).IsEncrypted() {
		t.Fatal("decrypt left the output encrypted")
	}
}

func TestRunClientErrors(t *testing.T) {
	ctx, paths := testContext(t, "page_1.pdf")

	_, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"rotate","angle":90,"pages":"5"}]`), []string{paths["page_1.pdf"]}, nil)
	if err == nil {
		t.Fatal("a page past the end must fail")
	}

	status, message := api.ParseError(handleError(err, ctx.DirPath()))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (message: %s)", status, message)
	}

	if !strings.Contains(message, `Stage 'rotate' failed on 'page_1.pdf'`) || strings.Contains(message, ctx.DirPath()) {
		t.Fatalf("message = %q, want the stage and filename without disk paths", message)
	}
}

// TestRunEditAttribution pins the stage a failed edit is reported under:
// litho applies a queued edit when a later stage or the write runs, and the
// error names the stage that queued it, not the one that applied it.
func TestRunEditAttribution(t *testing.T) {
	const stamp = `{"op":"stamp-pdf","source":"page_2.pdf","sourcePage":9}`

	for _, tc := range []struct {
		name  string
		after string
	}{
		{name: "alone"},
		{name: "rotate", after: `,{"op":"rotate","angle":90}`},
		{name: "select", after: `,{"op":"select","pages":"1-2"}`},
		{name: "delete", after: `,{"op":"delete","pages":"1"}`},
		{name: "redact", after: `,{"op":"redact","regions":[{"page":1,"x":0,"y":0,"width":10,"height":10}]}`},
		{name: "convert-colors", after: `,{"op":"convert-colors","target":"gray"}`},
		{name: "flatten-transparency", after: `,{"op":"flatten-transparency"}`},
		{name: "remediate", after: `,{"op":"remediate","profile":"PDF/A-2b"}`},
		{name: "optimize", after: `,{"op":"optimize"}`},
		{name: "set-metadata, then attach", after: `,{"op":"set-metadata","title":"T"},{"op":"attach","file":"embed_1.xml"}`},
		{name: "sign", after: `,{"op":"sign","certificate":"cert.pem","key":"key.pem"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, "pages_3.pdf", "page_2.pdf", "embed_1.xml", "pem/cert.pem", "pem/key.pem")
			files := map[string]string{
				"page_2.pdf":  paths["page_2.pdf"],
				"embed_1.xml": paths["embed_1.xml"],
				"cert.pem":    paths["cert.pem"],
				"key.pem":     paths["key.pem"],
			}

			_, err := testProcessor().Run(ctx, mustStages(t, "["+stamp+tc.after+"]"), []string{paths["pages_3.pdf"]}, files)
			if err == nil {
				t.Fatal("a source page past the end must fail")
			}

			status, message := api.ParseError(handleError(err, ctx.DirPath()))
			want := "Stage 'stamp-pdf' failed on 'pages_3.pdf': pdf stamp: source page 9: page number out of bounds (source has 1 pages)"
			if status != http.StatusBadRequest || message != want {
				t.Fatalf("got %d %q, want 400 %q (error: %v)", status, message, want, err)
			}
		})
	}
}

// render returns the pixels of a page rendered at a low resolution.
func render(t *testing.T, path string, page int) []uint8 {
	t.Helper()

	img, err := openOutput(t, path).RenderPage(context.Background(), page, litho.WithRenderDPI(20))
	if err != nil {
		t.Fatalf("render page %d of %s: %v", page, path, err)
	}

	bounds := img.Bounds()
	pixels := make([]uint8, 0, bounds.Dx()*bounds.Dy()*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			pixels = append(pixels, uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
		}
	}

	return pixels
}

func TestRunRemediationHeaders(t *testing.T) {
	for _, tc := range []struct {
		name        string
		stages      string
		wantHeaders bool
	}{
		{name: "no remediation, no headers", stages: `[{"op":"rotate","angle":90}]`},
		{name: "PDF/A-2b", stages: `[{"op":"remediate","profile":"PDF/A-2b"}]`, wantHeaders: true},
		{name: "aggressive PDF/UA-1 with alternate text", stages: `[{"op":"remediate","profile":"PDF/UA-1","aggressive":true,"figureAltText":"A chart"}]`, wantHeaders: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, "page_1.pdf", "page_2.pdf")

			_, err := testProcessor().Run(ctx, mustStages(t, tc.stages), []string{paths["page_1.pdf"], paths["page_2.pdf"]}, nil)
			if err != nil {
				t.Fatal(err)
			}

			headers := ctx.OutputHeaders()
			for _, key := range []string{ResidualHeader, RolledBackHeader} {
				value, ok := headers[key]
				if ok != tc.wantHeaders {
					t.Fatalf("header %s set = %v, want %v", key, ok, tc.wantHeaders)
				}
				if !ok {
					continue
				}

				n, err := strconv.Atoi(value)
				if err != nil || n < 0 {
					t.Errorf("header %s = %q, want a count", key, value)
				}
			}
		})
	}
}

// TestRunRemediatePair pins that a remediate stage given a pair of profiles
// writes a PDF that conforms to both, its residual header 0 when it does.
func TestRunRemediatePair(t *testing.T) {
	for _, tc := range []struct {
		stages string
		pdfa   models.PDFAProfile
		pdfua  models.PDFUAProfile
	}{
		{stages: `[{"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"]}]`, pdfa: models.PDFA2b, pdfua: models.PDFUA1},
		{stages: `[{"op":"remediate","profile":["PDF/UA-2","PDF/A-4"]}]`, pdfa: models.PDFA4, pdfua: models.PDFUA2},
	} {
		t.Run(string(tc.pdfa)+" and "+string(tc.pdfua), func(t *testing.T) {
			ctx, paths := testContext(t, "page_1.pdf")

			outputs, err := testProcessor().Run(ctx, mustStages(t, tc.stages), []string{paths["page_1.pdf"]}, nil)
			if err != nil {
				t.Fatal(err)
			}

			if got := ctx.OutputHeaders()[ResidualHeader]; got != "0" {
				t.Errorf("header %s = %q, want 0", ResidualHeader, got)
			}

			r := openOutput(t, outputs[0])
			pdfa, err := r.ValidatePDFAReport(context.Background(), tc.pdfa)
			if err != nil {
				t.Fatal(err)
			}

			pdfua, err := r.ValidatePDFUAReport(context.Background(), tc.pdfua)
			if err != nil {
				t.Fatal(err)
			}

			for _, v := range slices.Concat(pdfa, pdfua) {
				t.Errorf("the output fails %s %s", v.Specification, v.Error())
			}
		})
	}
}

// recordingHandler keeps the records a logger writes.
type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())

	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

// TestRunRemediatePairLogsResidualByProfile pins that the warning of a pair
// that falls short groups its residual by profile, and that the residual
// header counts the violations of both. Without the system's fonts, the
// standard font stays unembedded, which PDF/A-2b and PDF/UA-1 both forbid.
func TestRunRemediatePairLogsResidualByProfile(t *testing.T) {
	ctx, _ := testContext(t)
	handler := new(recordingHandler)
	mock := &api.ContextMock{Context: ctx}
	mock.SetLogger(slog.New(handler))

	input := filepath.Join(ctx.DirPath(), "helvetica.pdf")
	if err := os.WriteFile(input, unembeddedFontPDF(), 0o600); err != nil {
		t.Fatal(err)
	}

	processor := testProcessor()
	processor.fonts = nil

	_, err := processor.Run(ctx, mustStages(t, `[{"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"]}]`), []string{input}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var residual map[string][]string
	for _, r := range handler.records {
		if r.Message != "remediation incomplete" {
			continue
		}

		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "residual" {
				residual, _ = a.Value.Any().(map[string][]string)
			}

			return true
		})
	}

	if len(residual["PDF/A-2b"]) == 0 || len(residual["PDF/UA-1"]) == 0 || len(residual) != 2 {
		t.Fatalf("residual = %v, want the violations of PDF/A-2b and of PDF/UA-1", residual)
	}

	total := len(residual["PDF/A-2b"]) + len(residual["PDF/UA-1"])
	if got := ctx.OutputHeaders()[ResidualHeader]; got != strconv.Itoa(total) {
		t.Errorf("header %s = %q, want %d, the violations of both", ResidualHeader, got, total)
	}
}

// unembeddedFontPDF is a one-page PDF drawing text in Helvetica, which it
// does not embed.
func unembeddedFontPDF() []byte {
	const content = "BT /F1 24 Tf 72 700 Td (Hello) Tj ET\n"

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")

	offsets := make([]int, len(objects))
	for i, o := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}

	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f\r\n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n\r\n", offset)
	}

	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)

	return buf.Bytes()
}

func TestRemediatedProfilesOf(t *testing.T) {
	pair := remediatedProfiles{target: "PDF/A-4", pdfua: "PDF/UA-2"}
	for spec, want := range map[string]string{
		"ISO_19005_4":    "PDF/A-4",
		"ISO_14289_2":    "PDF/UA-2",
		"ISO_32005":      "PDF/UA-2",
		"HYBRID_INVOICE": "PDF/A-4",
	} {
		if got := pair.of(spec); got != want {
			t.Errorf("of(%q) = %q, want %q", spec, got, want)
		}
	}

	if got := (remediatedProfiles{pdfua: "PDF/UA-1"}).of("ISO_32000_1"); got != "PDF/UA-1" {
		t.Errorf("a PDF/UA remediation files %q under %q", "ISO_32000_1", got)
	}

	if got := (remediatedProfiles{target: "PDF/A-2b"}).of("ISO_14289_1"); got != "PDF/A-2b" {
		t.Errorf("a PDF/A remediation files %q under %q", "ISO_14289_1", got)
	}
}

func TestRunRepairFacturX(t *testing.T) {
	ctx, paths := testContext(t, "page_1.pdf", "embed_1.xml")

	embedded, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"remediate","profile":"Factur-X","xml":"embed_1.xml","level":"EN 16931"}]`), []string{paths["page_1.pdf"]}, map[string]string{"embed_1.xml": paths["embed_1.xml"]})
	if err != nil {
		t.Fatal(err)
	}

	repaired, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"remediate","profile":"Factur-X"}]`), embedded, nil)
	if err != nil {
		t.Fatalf("repair an embedded invoice: %v", err)
	}

	// embed_1.xml is no CII invoice, and a repair leaves the XML as it is: it
	// must not add a violation to the ones the XML causes.
	fatals := func(path string) []string {
		report, err := openOutput(t, path).ValidateFacturXReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		var out []string
		for _, v := range report.Fatals() {
			out = append(out, v.Description)
		}

		return out
	}

	before, after := fatals(embedded[0]), fatals(repaired[0])
	for _, v := range after {
		if !slices.Contains(before, v) {
			t.Errorf("the repair added a violation: %s", v)
		}
	}

	_, err = testProcessor().Run(ctx, mustStages(t, `[{"op":"remediate","profile":"Factur-X"}]`), []string{paths["page_1.pdf"]}, nil)
	if err == nil {
		t.Fatal("repairing a PDF without an invoice must fail")
	}

	status, message := api.ParseError(err)
	if status != http.StatusBadRequest || !strings.Contains(message, "no hybrid invoice attachment found") {
		t.Fatalf("status %d, message %q, want a 400 naming the missing invoice", status, message)
	}
}

func TestRunRender(t *testing.T) {
	for _, tc := range []struct {
		name      string
		stages    string
		wantNames []string
		wantType  string
	}{
		{
			name:      "every page as png, next to the document",
			stages:    `[{"op":"render","dpi":36}]`,
			wantNames: []string{"pages_3.pdf", "pages_3_1.png", "pages_3_2.png", "pages_3_3.png"},
			wantType:  "image/png",
		},
		{
			name:      "selected pages of the merged document as jpeg",
			stages:    `[{"op":"merge"},{"op":"render","pages":"1,4","dpi":36,"format":"jpeg","quality":80}]`,
			wantNames: []string{"pages_3.pdf", "pages_3_1.jpeg", "pages_3_4.jpeg"},
			wantType:  "image/jpeg",
		},
		{
			name:      "an encrypted output, opened with its password",
			stages:    `[{"op":"encrypt","userPassword":"u"},{"op":"render","pages":"1","dpi":36}]`,
			wantNames: []string{"pages_3.pdf", "pages_3_1.png"},
			wantType:  "image/png",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixtures := []string{"pages_3.pdf"}
			if strings.Contains(tc.stages, "merge") {
				fixtures = append(fixtures, "page_1.pdf")
			}

			ctx, paths := testContext(t, fixtures...)
			inputs := make([]string, len(fixtures))
			for i, f := range fixtures {
				inputs[i] = paths[f]
			}

			outputs, err := testProcessor().Run(ctx, mustStages(t, tc.stages), inputs, nil)
			if err != nil {
				t.Fatal(err)
			}

			names := make([]string, len(outputs))
			for i, path := range outputs {
				names[i] = ctx.OriginalFilename(path)
			}
			if !slices.Equal(names, tc.wantNames) {
				t.Fatalf("outputs = %v, want %v", names, tc.wantNames)
			}

			for _, path := range outputs[1:] {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}

				if got := http.DetectContentType(data); got != tc.wantType {
					t.Errorf("%s: content type %s, want %s", ctx.OriginalFilename(path), got, tc.wantType)
				}

				cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("decode %s: %v", ctx.OriginalFilename(path), err)
				}

				// The fixtures' pages are 448 x 632 points: half as many
				// pixels at 36 dpi.
				if cfg.Width != 224 || cfg.Height != 316 {
					t.Errorf("%s: %dx%d, want a page drawn at 36 dpi", ctx.OriginalFilename(path), cfg.Width, cfg.Height)
				}
			}
		})
	}
}

// recordingFonts returns a font provider that supplies no program, and the
// /BaseFont names it was asked for.
func recordingFonts() (litho.FontProvider, func() []string) {
	var (
		mu    sync.Mutex
		asked []string
	)

	provider := litho.FontProviderFunc(func(_ context.Context, req litho.FontRequest) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		asked = append(asked, req.BaseFont)

		return nil, nil
	})

	return provider, func() []string {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(asked)
	}
}

// A stamped text uses a standard font the document does not embed: render and
// redact ask the processor's fonts for it, rather than drawing boxes. The stamp
// is written by a run of its own: on the Writer that stamps, a redaction does
// not bound the stamp's glyphs by their outlines.
func TestRunAsksForNonEmbeddedFonts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stages string
	}{
		{
			name:   "render",
			stages: `[{"op":"render","dpi":36}]`,
		},
		{
			name:   "redact",
			stages: `[{"op":"redact","regions":[{"page":1,"x":40,"y":290,"width":100,"height":60}]}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, "page_1.pdf")

			stamped, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"stamp-text","text":"HELLO","x":50,"y":300,"fontSize":48}]`), []string{paths["page_1.pdf"]}, nil)
			if err != nil {
				t.Fatal(err)
			}

			provider, asked := recordingFonts()
			processor := testProcessor()
			processor.renderFonts = litho.WithRenderFontProvider(provider)

			_, err = processor.Run(ctx, mustStages(t, tc.stages), stamped, nil)
			if err != nil {
				t.Fatal(err)
			}

			if !slices.Contains(asked(), "Helvetica") {
				t.Fatalf("fonts asked = %v, want Helvetica", asked())
			}
		})
	}
}

// Detaching a file the PDF does not embed is the client's mistake: it answers
// 400 with the names the PDF has, rather than the PDF unchanged.
func TestRunDetach(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stages   string
		attached bool
		wantErr  string
	}{
		{
			name:    "no attachment",
			stages:  `[{"op":"detach","name":"nope.xml"}]`,
			wantErr: `Stage 'detach' failed on 'page_1.pdf': no attachment named "nope.xml"; the PDF has none`,
		},
		{
			name:     "another attachment",
			stages:   `[{"op":"detach","name":"nope.xml"}]`,
			attached: true,
			wantErr:  `Stage 'detach' failed on 'page_1.pdf': no attachment named "nope.xml"; the PDF's attachments are: embed_1.xml`,
		},
		{
			name:    "an attachment of the same run",
			stages:  `[{"op":"attach","file":"embed_2.xml"},{"op":"detach","name":"nope.xml"}]`,
			wantErr: `Stage 'detach' failed on 'page_1.pdf': no attachment named "nope.xml"; the PDF's attachments are: embed_2.xml`,
		},
		{
			name:    "already detached by the same run",
			stages:  `[{"op":"attach","file":"embed_2.xml"},{"op":"detach","name":"embed_2.xml"},{"op":"detach","name":"embed_2.xml"}]`,
			wantErr: `Stage 'detach' failed on 'page_1.pdf': no attachment named "embed_2.xml"; the PDF has none`,
		},
		{
			name:     "the attachment",
			stages:   `[{"op":"detach","name":"embed_1.xml"}]`,
			attached: true,
		},
		{
			name:   "attached then detached by the same run",
			stages: `[{"op":"attach","file":"embed_2.xml"},{"op":"detach","name":"embed_2.xml"}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			processor := testProcessor()
			ctx, paths := testContext(t, "page_1.pdf", "embed_1.xml", "embed_2.xml")
			files := map[string]string{"embed_1.xml": paths["embed_1.xml"], "embed_2.xml": paths["embed_2.xml"]}

			inputs := []string{paths["page_1.pdf"]}
			if tc.attached {
				var err error
				inputs, err = processor.Run(ctx, mustStages(t, `[{"op":"attach","file":"embed_1.xml"}]`), inputs, files)
				if err != nil {
					t.Fatal(err)
				}
			}

			outputs, err := processor.Run(ctx, mustStages(t, tc.stages), inputs, files)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("err = %v, want none", err)
				}

				files, err := openOutput(t, outputs[0]).EmbeddedFiles(context.Background())
				if err != nil || len(files) != 0 {
					t.Fatalf("embedded files = %v, %v, want none", files, err)
				}

				return
			}

			status, message := api.ParseError(err)
			if status != http.StatusBadRequest || message != tc.wantErr {
				t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
			}
		})
	}
}

// fileSize returns the size of the file at path.
func fileSize(t *testing.T, path string) int64 {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	return info.Size()
}

// hasObjectStreams reports whether the PDF at path packs objects into object
// streams.
func hasObjectStreams(t *testing.T, path string) bool {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return bytes.Contains(data, []byte("/ObjStm"))
}

// optimize rewrites the document whole, since litho compacts on a rewrite
// only, but refuses to drop what a rewrite cannot keep.
func TestRunOptimize(t *testing.T) {
	t.Run("alone, it compacts", func(t *testing.T) {
		ctx, paths := testContext(t, "pages_12.pdf")

		outputs, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"optimize"}]`), []string{paths["pages_12.pdf"]}, nil)
		if err != nil {
			t.Fatal(err)
		}

		if fileSize(t, outputs[0]) >= fileSize(t, paths["pages_12.pdf"]) || !hasObjectStreams(t, outputs[0]) {
			t.Fatalf("optimized %d bytes from %d, with object streams %t: want smaller and packed", fileSize(t, outputs[0]), fileSize(t, paths["pages_12.pdf"]), hasObjectStreams(t, outputs[0]))
		}
	})

	t.Run("after select, it compacts", func(t *testing.T) {
		ctx, paths := testContext(t, "pages_12.pdf")

		outputs, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"select","pages":"1-6"},{"op":"optimize"}]`), []string{paths["pages_12.pdf"]}, nil)
		if err != nil {
			t.Fatal(err)
		}

		if pageCount(t, outputs[0]) != 6 || !hasObjectStreams(t, outputs[0]) {
			t.Fatalf("want 6 packed pages, got %d, packed %t", pageCount(t, outputs[0]), hasObjectStreams(t, outputs[0]))
		}
	})

	t.Run("a declared PDF/A-1 keeps its cross-reference table", func(t *testing.T) {
		processor := testProcessor()
		ctx, paths := testContext(t, "page_1.pdf")

		pdfa, err := processor.Run(ctx, mustStages(t, `[{"op":"remediate","profile":"PDF/A-1b"}]`), []string{paths["page_1.pdf"]}, nil)
		if err != nil {
			t.Fatal(err)
		}

		outputs, err := processor.Run(ctx, mustStages(t, `[{"op":"optimize"}]`), pdfa, nil)
		if err != nil {
			t.Fatal(err)
		}

		part, err := openOutput(t, outputs[0]).DeclaredPDFAPart(context.Background())
		if err != nil || part != 1 || hasObjectStreams(t, outputs[0]) {
			t.Fatalf("part = %d, %v, packed %t: want PDF/A-1 unpacked", part, err, hasObjectStreams(t, outputs[0]))
		}
	})

	for _, tc := range []struct {
		name      string
		prepare   string
		fixtures  []string
		stages    string
		wantErr   string
		encrypted bool
	}{
		{
			name:     "signed",
			prepare:  `[{"op":"sign","certificate":"cert.pem","key":"key.pem"}]`,
			fixtures: []string{"pem/cert.pem", "pem/key.pem"},
			stages:   `[{"op":"optimize"}]`,
			wantErr:  "Stage 'optimize' failed on 'page_1.pdf': the PDF is signed, and optimizing would remove its signatures",
		},
		{
			name:    "encrypted",
			prepare: `[{"op":"encrypt","ownerPassword":"owner"}]`,
			stages:  `[{"op":"optimize"}]`,
			wantErr: `Stage 'optimize' failed on 'page_1.pdf': the PDF is encrypted, and optimizing would write it decrypted: on /forms/pdf/pipe, add a "decrypt" stage before "optimize" to remove the encryption, or an "encrypt" stage to keep one`,
		},
		{
			name:    "encrypted, then decrypted",
			prepare: `[{"op":"encrypt","ownerPassword":"owner"}]`,
			stages:  `[{"op":"decrypt","password":"owner"},{"op":"optimize"}]`,
		},
		{
			name:    "encrypted with an owner password only, then decrypted without password",
			prepare: `[{"op":"encrypt","ownerPassword":"owner"}]`,
			stages:  `[{"op":"decrypt"},{"op":"optimize"}]`,
		},
		{
			name:      "encrypted, then encrypted again",
			prepare:   `[{"op":"encrypt","ownerPassword":"owner"}]`,
			stages:    `[{"op":"optimize"},{"op":"encrypt","ownerPassword":"other"}]`,
			encrypted: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			processor := testProcessor()
			ctx, paths := testContext(t, append([]string{"page_1.pdf"}, tc.fixtures...)...)
			files := make(map[string]string, len(paths))
			maps.Copy(files, paths)

			inputs, err := processor.Run(ctx, mustStages(t, tc.prepare), []string{paths["page_1.pdf"]}, files)
			if err != nil {
				t.Fatal(err)
			}

			outputs, err := processor.Run(ctx, mustStages(t, tc.stages), inputs, nil)
			if tc.wantErr != "" {
				status, message := api.ParseError(err)
				if status != http.StatusBadRequest || message != tc.wantErr {
					t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if encrypted := openOutput(t, outputs[0], litho.WithPassword("other")).IsEncrypted(); encrypted != tc.encrypted {
				t.Fatalf("encrypted = %t, want %t", encrypted, tc.encrypted)
			}
		})
	}
}

// set-form with a field the PDF does not have answers 400 naming it, and the
// fields the PDF has, whatever stage comes last in the run.
func TestRunSetFormUnknownField(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture string
		stages  string
		wantErr string
	}{
		{
			name:    "form",
			fixture: "form.pdf",
			stages:  `[{"op":"set-form","fields":{"nope":"x"}}]`,
			wantErr: `Stage 'set-form' failed on 'form.pdf': form field "nope" not found; the PDF's fields are: name`,
		},
		{
			name:    "form, a known field and two unknown ones",
			fixture: "form.pdf",
			stages:  `[{"op":"set-form","fields":{"name":"Gotenberg","zed":"z","nope":"x"}},{"op":"rotate","angle":90}]`,
			wantErr: `Stage 'set-form' failed on 'form.pdf': form fields "nope", "zed" not found; the PDF's fields are: name`,
		},
		{
			name:    "no form",
			fixture: "page_1.pdf",
			stages:  `[{"op":"set-form","fields":{"nope":"x"}}]`,
			wantErr: `Stage 'set-form' failed on 'page_1.pdf': form field "nope" not found; the PDF has no form fields`,
		},
		{
			name:    "a name only the XFA data gives is not a field set-form fills",
			fixture: "xfa.pdf",
			stages:  `[{"op":"set-form","fields":{"nope":"x"}}]`,
			wantErr: `Stage 'set-form' failed on 'xfa.pdf': form field "nope" not found; the PDF's fields are: name`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, tc.fixture)

			_, err := testProcessor().Run(ctx, mustStages(t, tc.stages), []string{paths[tc.fixture]}, nil)

			status, message := api.ParseError(err)
			if status != http.StatusBadRequest || message != tc.wantErr {
				t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
			}

			if !errors.Is(err, litho.ErrFormFieldNotFound) {
				t.Fatalf("the logged error lost litho's: %v", err)
			}
		})
	}

	// The known field still fills.
	ctx, paths := testContext(t, "form.pdf")
	outputs, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"set-form","fields":{"name":"Gotenberg"}}]`), []string{paths["form.pdf"]}, nil)
	if err != nil {
		t.Fatal(err)
	}

	values, err := openOutput(t, outputs[0]).FormData(context.Background())
	if err != nil || values["name"] != "Gotenberg" {
		t.Fatalf("form data = %v, %v, want name filled", values, err)
	}
}

// roundTripFunc adapts a function to an [http.RoundTripper].
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// A timestamp authority the outbound filters refuse is logged with its URL,
// without its query, and the rule, while the client only learns it could not
// be reached.
func TestRefusalLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	refused := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, gotenberg.FilterOutboundURL(req.Context(), req.URL.String(), nil, []*regexp2.Regexp{regexp2.MustCompile(`^https://tsa\.example\.com/`, regexp2.None)}, time.Now().Add(time.Second))
	})

	req := httptest.NewRequest(http.MethodPost, "https://tsa.example.com/stamp?key=secret", nil)
	_, err := refusalLogging{base: refused, logger: logger}.RoundTrip(req)
	if !errors.Is(err, gotenberg.ErrFiltered) {
		t.Fatalf("err = %v, want the refusal", err)
	}

	record := buf.String()
	for _, want := range []string{`"level":"WARN"`, `"msg":"outbound request blocked by the PDF filters"`, `"url":"https://tsa.example.com/stamp"`, `"rule":"deny-list"`} {
		if !strings.Contains(record, want) {
			t.Errorf("record %s lacks %s", record, want)
		}
	}
	if strings.Contains(record, "secret") {
		t.Errorf("record %s leaks the query", record)
	}

	// A transport error that is no refusal is the caller's to report.
	buf.Reset()
	failing := roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("connection refused") })
	_, _ = refusalLogging{base: failing, logger: logger}.RoundTrip(req)
	if buf.Len() != 0 {
		t.Errorf("logged %s for a transport error", buf.String())
	}
}

// A stage that rewrites a signed PDF would drop its signatures: it answers
// 400, while an incremental edit keeps them.
func TestRunSignedInput(t *testing.T) {
	const signedMessage = "the PDF is signed, and this stage rewrites it, which would remove its signatures"

	for _, tc := range []struct {
		name     string
		stages   string
		fixtures []string
		wantOp   string
	}{
		{name: "select", stages: `[{"op":"select","pages":"1"}]`, wantOp: "select"},
		{name: "delete", stages: `[{"op":"delete","pages":"1"}]`, wantOp: "delete"},
		{name: "insert", stages: `[{"op":"insert","source":"page_2.pdf"}]`, fixtures: []string{"page_2.pdf"}, wantOp: "insert"},
		{name: "encrypt", stages: `[{"op":"encrypt","userPassword":"u"}]`, wantOp: "encrypt"},
		{name: "flatten-form", stages: `[{"op":"flatten-form"}]`, wantOp: "flatten-form"},
		{name: "redact", stages: `[{"op":"redact","regions":[{"page":1,"x":0,"y":0,"width":10,"height":10}]}]`, wantOp: "redact"},
		{name: "remediate", stages: `[{"op":"remediate","profile":"PDF/A-2b"}]`, wantOp: "remediate"},
		{name: "convert-colors", stages: `[{"op":"convert-colors","target":"gray"}]`, wantOp: "convert-colors"},
		{name: "flatten-transparency", stages: `[{"op":"flatten-transparency"}]`, wantOp: "flatten-transparency"},
		{name: "split", stages: `[{"op":"split","every":1}]`, wantOp: "split"},
		{name: "merge", stages: `[{"op":"merge"}]`, fixtures: []string{"page_2.pdf"}, wantOp: "merge"},
		{name: "an incremental edit, then a rewrite", stages: `[{"op":"stamp-text","text":"DRAFT"},{"op":"select","pages":"1"}]`, wantOp: "select"},
		{name: "stamp-text", stages: `[{"op":"stamp-text","text":"DRAFT"}]`},
		{name: "set-metadata", stages: `[{"op":"set-metadata","title":"Signed"}]`},
		{name: "rotate", stages: `[{"op":"rotate","angle":90}]`},
		{name: "attach", stages: `[{"op":"attach","file":"embed_1.xml"}]`, fixtures: []string{"embed_1.xml"}},
		{name: "autotag", stages: `[{"op":"autotag"}]`},
		{name: "set-bookmarks", stages: `[{"op":"set-bookmarks","bookmarks":[{"title":"One","pageNumber":1}]}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			processor := testProcessor()
			ctx, paths := testContext(t, append([]string{"page_1.pdf", "pem/cert.pem", "pem/key.pem"}, tc.fixtures...)...)
			files := make(map[string]string, len(paths))
			maps.Copy(files, paths)

			signed, err := processor.Run(ctx, mustStages(t, `[{"op":"sign","certificate":"cert.pem","key":"key.pem"}]`), []string{paths["page_1.pdf"]}, files)
			if err != nil {
				t.Fatal(err)
			}

			inputs := signed
			if tc.name == "merge" {
				inputs = append(inputs, paths["page_2.pdf"])
			}

			outputs, err := processor.Run(ctx, mustStages(t, tc.stages), inputs, files)
			if tc.wantOp != "" {
				status, message := api.ParseError(err)
				want := fmt.Sprintf("Stage '%s' failed on 'page_1.pdf': %s", tc.wantOp, signedMessage)
				if status != http.StatusBadRequest || message != want {
					t.Fatalf("got %d %q, want 400 %q", status, message, want)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want none", err)
			}

			signatures, err := openOutput(t, outputs[0]).ExtractSignatures(context.Background())
			if err != nil || len(signatures) != 1 {
				t.Fatalf("signatures = %v, %v, want the input's", signatures, err)
			}
		})
	}
}

// set-form refuses a value the field cannot hold with litho's message, which
// names the field and what it takes, under the set-form stage whatever stage
// applies the edit.
func TestRunSetFormValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture string
		stages  string
		wantErr string
	}{
		{
			name:    "a check box state no widget shows",
			fixture: "fields.pdf",
			stages:  `[{"op":"set-form","fields":{"agree":"true"}}]`,
			wantErr: `Stage 'set-form' failed on 'fields.pdf': value the form field cannot hold: "true" for field "agree": no widget shows that state; it takes Off or one of Yes`,
		},
		{
			name:    "a combo box value outside its options, then rotate",
			fixture: "fields.pdf",
			stages:  `[{"op":"set-form","fields":{"color":"Blue"}},{"op":"rotate","angle":90}]`,
			wantErr: `Stage 'set-form' failed on 'fields.pdf': value the form field cannot hold: "Blue" for field "color": "Blue" is none of its options: Red, Green`,
		},
		{
			name:    "several values for a field that selects one",
			fixture: "multi.pdf",
			stages:  `[{"op":"set-form","fields":{"country":["FR","DE"]}}]`,
			wantErr: `Stage 'set-form' failed on 'multi.pdf': value the form field cannot hold: ["FR" "DE"] for field "country": only a list box that allows several selections takes several`,
		},
		{
			name:    "a signature field",
			fixture: "multi.pdf",
			stages:  `[{"op":"set-form","fields":{"sig":"x"}}]`,
			wantErr: `Stage 'set-form' failed on 'multi.pdf': value the form field cannot hold: "x" for field "sig": a signature field takes a signature, not a value`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, tc.fixture)

			_, err := testProcessor().Run(ctx, mustStages(t, tc.stages), []string{paths[tc.fixture]}, nil)

			status, message := api.ParseError(handleError(err, ctx.DirPath()))
			if status != http.StatusBadRequest || message != tc.wantErr {
				t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
			}

			if !errors.Is(err, litho.ErrFormFieldValue) {
				t.Fatalf("the logged error lost litho's: %v", err)
			}
		})
	}

	// An array selects several items of a list box that allows it.
	ctx, paths := testContext(t, "multi.pdf")
	outputs, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"set-form","fields":{"tags":["b","c"]}}]`), []string{paths["multi.pdf"]}, nil)
	if err != nil {
		t.Fatal(err)
	}

	fields, err := openOutput(t, outputs[0]).FormFields(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	i := slices.IndexFunc(fields, func(f models.FormField) bool { return f.Name == "tags" })
	if i < 0 || !slices.Equal(fields[i].Values, []string{"b", "c"}) {
		t.Fatalf("tags = %+v, want b and c selected", fields)
	}
}

// Setting a radio button group draws the state on its widget and Off on the
// others. multi.pdf's "size" group has S, M and L widgets on page 2, whose on
// appearance fills their center and whose Off appearance does not.
func TestRunSetFormRadio(t *testing.T) {
	ctx, paths := testContext(t, "multi.pdf")
	path := paths["multi.pdf"]

	for _, tc := range []struct {
		state string
		want  map[string]bool
	}{
		{state: "M", want: map[string]bool{"S": false, "M": true, "L": false}},
		{state: "L", want: map[string]bool{"S": false, "M": false, "L": true}},
	} {
		outputs, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"set-form","fields":{"size":"`+tc.state+`"}}]`), []string{path}, nil)
		if err != nil {
			t.Fatalf("set size to %s: %v", tc.state, err)
		}
		path = outputs[0]

		img, err := openOutput(t, path).RenderPage(context.Background(), 2, litho.WithRenderDPI(72))
		if err != nil {
			t.Fatal(err)
		}

		// At 72 DPI a pixel is a point; the widgets sit at x 72, 112 and 152,
		// y 600 to 620, from the bottom of the 792-point page.
		for state, x := range map[string]int{"S": 82, "M": 122, "L": 162} {
			r, g, b, _ := img.At(x, 792-610).RGBA()
			if filled := r == 0 && g == 0 && b == 0; filled != tc.want[state] {
				t.Errorf("after setting %s, widget %s filled = %t, want %t", tc.state, state, filled, tc.want[state])
			}
		}
	}
}

func TestRunBookmarkPages(t *testing.T) {
	for _, tc := range []struct {
		name    string
		stages  string
		wantErr string
	}{
		{
			name:   "a page of the document",
			stages: `[{"op":"set-bookmarks","bookmarks":[{"title":"A","pageNumber":1,"children":[{"title":"B","pageNumber":1}]}]}]`,
		},
		{
			name:    "past the last page",
			stages:  `[{"op":"set-bookmarks","bookmarks":[{"title":"A","pageNumber":2}]}]`,
			wantErr: "Stage 'set-bookmarks' failed on 'page_1.pdf': bookmarks[0].pageNumber 2 is past the last page 1",
		},
		{
			name:    "page zero",
			stages:  `[{"op":"set-bookmarks","bookmarks":[{"title":"A"}]}]`,
			wantErr: "Stage 'set-bookmarks' failed on 'page_1.pdf': bookmarks[0].pageNumber must be a page from 1 to 1, got 0",
		},
		{
			name:    "a child past the last page",
			stages:  `[{"op":"set-bookmarks","bookmarks":[{"title":"A","pageNumber":1,"children":[{"title":"B","pageNumber":1},{"title":"C","pageNumber":7}]}]}]`,
			wantErr: "Stage 'set-bookmarks' failed on 'page_1.pdf': bookmarks[0].children[1].pageNumber 7 is past the last page 1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, "page_1.pdf")

			_, err := testProcessor().Run(ctx, mustStages(t, tc.stages), []string{paths["page_1.pdf"]}, nil)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			status, message := api.ParseError(err)
			if status != http.StatusBadRequest || message != tc.wantErr {
				t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
			}
		})
	}
}

func TestRunSignErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stages   string
		fixtures []string
		wantErr  string
	}{
		{
			name:     "a PEM certificate without its key",
			stages:   `[{"op":"sign","certificate":"cert.pem"}]`,
			fixtures: []string{"pem/cert.pem"},
			wantErr:  `Stage 'sign' failed on 'page_1.pdf': "certificate" "cert.pem" is a PEM file: "key" is required, the PEM private key that goes with it`,
		},
		{
			name:     "a PKCS #12 store with a wrong password",
			stages:   `[{"op":"sign","certificate":"cert.p12","certificatePassword":"nope"}]`,
			fixtures: []string{"pem/cert.p12"},
			wantErr:  `Stage 'sign' failed on 'page_1.pdf': the PKCS #12 certificate "cert.p12" did not open: "certificatePassword" is missing or wrong`,
		},
		{
			name:     "a PKCS #12 store",
			stages:   `[{"op":"sign","certificate":"cert.p12","certificatePassword":"secret"}]`,
			fixtures: []string{"pem/cert.p12"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, append([]string{"page_1.pdf"}, tc.fixtures...)...)

			_, err := testProcessor().Run(ctx, mustStages(t, tc.stages), []string{paths["page_1.pdf"]}, paths)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			status, message := api.ParseError(err)
			if status != http.StatusBadRequest || message != tc.wantErr {
				t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
			}
		})
	}
}
