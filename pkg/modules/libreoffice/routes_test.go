package libreoffice

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v9/pkg/modules/libreoffice/api"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/pdf"
)

// compoundFile writes a document whose header marks it as a compound file. Over
// an OOXML extension, that means an encrypted payload.
func compoundFile(t *testing.T, dir, name string) string {
	t.Helper()

	content := append(
		[]byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1},
		bytes.Repeat([]byte{0x00}, 64)...,
	)

	return writeTestFile(t, dir, name, content)
}

// zipPackage writes a minimal, unencrypted OOXML package.
func zipPackage(t *testing.T, dir, name string) string {
	t.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f, err := w.Create("[Content_Types].xml")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	_, err = f.Write([]byte("<Types/>"))
	if err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return writeTestFile(t, dir, name, buf.Bytes())
}

func writeTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(dir, name)
	err := os.WriteFile(path, content, 0o600)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	return path
}

// TestConvertRoute_FailureStatus pins the branch table that decides whether a
// LibreOffice failure is the client's fault. See
// https://github.com/gotenberg/gotenberg/issues/1588.
func TestConvertRoute_FailureStatus(t *testing.T) {
	dir := t.TempDir()

	var (
		protected   = compoundFile(t, dir, "protected_page_1.docx")
		plain       = zipPackage(t, dir, "page_1.docx")
		legacy      = compoundFile(t, dir, "legacy.doc")
		corrupted   = writeTestFile(t, dir, "corrupted.docx", []byte("not a document"))
		unreachable = filepath.Join(dir, "vanished.docx")
	)

	for _, tc := range []struct {
		name       string
		inputPath  string
		values     map[string][]string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "encrypted document, no password",
			inputPath:  protected,
			err:        libreofficeapi.ErrLoadFailed,
			wantStatus: http.StatusBadRequest,
			wantBody:   "The document 'protected_page_1.docx' is password-protected. Provide its password in the 'password' form field.",
		},
		{
			name:       "encrypted document, wrong password",
			inputPath:  protected,
			values:     map[string][]string{"password": {"bar"}},
			err:        libreofficeapi.ErrInvalidPassword,
			wantStatus: http.StatusBadRequest,
			wantBody:   "The password for the document 'protected_page_1.docx' is incorrect. Check the 'password' form field.",
		},
		{
			name:       "unreadable document",
			inputPath:  corrupted,
			err:        libreofficeapi.ErrLoadFailed,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not read the document 'corrupted.docx'. Ensure the file is not corrupted and that its extension matches its actual format.",
		},
		{
			name:       "unreadable legacy document",
			inputPath:  legacy,
			err:        libreofficeapi.ErrLoadFailed,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not read the document 'legacy.doc'. Ensure the file is not corrupted and that its extension matches its actual format.",
		},
		{
			name:       "malformed page ranges",
			inputPath:  plain,
			values:     map[string][]string{"nativePages": {"foo"}},
			err:        libreofficeapi.ErrSaveFailed,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not apply the pages 'foo' to the document 'page_1.docx'. Check the 'nativePages' form field; valid values look like '1-4', '2' or '1,3,5-7'.",
		},
		{
			name:       "nothing implicated, save failed",
			inputPath:  plain,
			err:        libreofficeapi.ErrSaveFailed,
			wantStatus: http.StatusInternalServerError,
			wantBody:   fmt.Sprintf(unattributableFailureMessage, "page_1.docx"),
		},
		{
			name:       "page ranges do not excuse a crash",
			inputPath:  plain,
			values:     map[string][]string{"nativePages": {"1-2"}},
			err:        libreofficeapi.ErrCrashed,
			wantStatus: http.StatusInternalServerError,
			wantBody:   fmt.Sprintf(unattributableFailureMessage, "page_1.docx"),
		},
		{
			name:       "detection cannot read the document",
			inputPath:  unreachable,
			err:        libreofficeapi.ErrLoadFailed,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not read the document 'vanished.docx'. Ensure the file is not corrupted and that its extension matches its actual format.",
		},
		{
			name:       "unmapped failure",
			inputPath:  plain,
			err:        fmt.Errorf("lok unknown: boom"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   http.StatusText(http.StatusInternalServerError),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &api.ContextMock{Context: new(api.Context)}
			ctx.SetDirPath(dir)
			ctx.SetFiles(map[string]string{filepath.Base(tc.inputPath): tc.inputPath})
			ctx.SetValues(tc.values)
			ctx.SetLogger(slog.New(slog.DiscardHandler))

			converter := &libreofficeapi.ApiMock{
				ExtensionsMock: func() []string {
					return []string{".docx", ".doc"}
				},
				PdfMock: func(_ context.Context, _ *slog.Logger, _, _ string, _ libreofficeapi.Options) error {
					// Mirror the wrapping done by [libreofficeapi.Api.Pdf].
					return fmt.Errorf("supervisor run task: %w", tc.err)
				},
			}

			c := echo.New().NewContext(
				httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", nil),
				httptest.NewRecorder(),
			)
			c.Set("context", ctx.Context)

			err := convertRoute(converter, new(pdf.Processor)).Handler(c)
			if err == nil {
				t.Fatal("expected an error, got none")
			}

			status, message := api.ParseError(err)
			if status != tc.wantStatus {
				t.Errorf("status = %d, want %d (message: %s)", status, tc.wantStatus, message)
			}
			if message != tc.wantBody {
				t.Errorf("message =\n%s\nwant\n%s", message, tc.wantBody)
			}
		})
	}
}

// TestConvertRoute_Stages checks that the PDF formats apply natively to every
// document, and that the stages then run on the converted PDFs.
func TestConvertRoute_Stages(t *testing.T) {
	dir := t.TempDir()

	pdfFixture, err := os.ReadFile("../../../test/integration/testdata/page_1.pdf")
	if err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"page_1.docx": zipPackage(t, dir, "page_1.docx"),
		"page_2.docx": zipPackage(t, dir, "page_2.docx"),
	}

	ctx := &api.ContextMock{Context: &api.Context{Context: context.Background()}}
	ctx.SetDirPath(dir)
	ctx.SetFiles(files)
	ctx.SetValues(map[string][]string{
		"nativePdfa": {libreofficeapi.PdfA1b},
		"stages":     {`[{"op":"merge"}]`},
	})
	ctx.SetLogger(slog.New(slog.DiscardHandler))

	var native []libreofficeapi.PdfFormats
	converter := &libreofficeapi.ApiMock{
		ExtensionsMock: func() []string {
			return []string{".docx"}
		},
		PdfMock: func(_ context.Context, _ *slog.Logger, _, outputPath string, options libreofficeapi.Options) error {
			native = append(native, options.PdfFormats)
			return os.WriteFile(outputPath, pdfFixture, 0o600)
		},
	}

	c := echo.New().NewContext(
		httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", nil),
		httptest.NewRecorder(),
	)
	c.Set("context", ctx.Context)

	err = convertRoute(converter, new(pdf.Processor)).Handler(c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(native) != 2 {
		t.Fatalf("%d conversions, want 2", len(native))
	}

	for _, formats := range native {
		if formats.PdfA != libreofficeapi.PdfA1b {
			t.Errorf("native formats = %+v, want PDF/A %q", formats, libreofficeapi.PdfA1b)
		}
	}

	outputs := ctx.OutputPaths()
	if len(outputs) != 1 {
		t.Fatalf("%d outputs, want 1 merged PDF", len(outputs))
	}

	if name := ctx.OriginalFilename(outputs[0]); name != "page_1.docx.pdf" {
		t.Errorf("output name = %q, want page_1.docx.pdf", name)
	}
}

// TestConvertRoute_PaperNeedsBothSides checks that a custom paper size needs
// both sides unless a named size supplies the other one.
func TestConvertRoute_PaperNeedsBothSides(t *testing.T) {
	dir := t.TempDir()

	for _, tc := range []struct {
		name    string
		values  map[string][]string
		wantErr bool
	}{
		{name: "width alone", values: map[string][]string{"paperWidth": {"8in"}}, wantErr: true},
		{name: "width over a named size", values: map[string][]string{"paperSize": {"A4"}, "paperWidth": {"8in"}}},
		{name: "both sides", values: map[string][]string{"paperWidth": {"8in"}, "paperHeight": {"10in"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &api.ContextMock{Context: &api.Context{Context: context.Background()}}
			ctx.SetDirPath(dir)
			ctx.SetFiles(map[string]string{"page_1.docx": zipPackage(t, dir, "page_1.docx")})
			ctx.SetValues(tc.values)
			ctx.SetLogger(slog.New(slog.DiscardHandler))

			var got libreofficeapi.Options
			converter := &libreofficeapi.ApiMock{
				ExtensionsMock: func() []string { return []string{".docx"} },
				PdfMock: func(_ context.Context, _ *slog.Logger, _, _ string, options libreofficeapi.Options) error {
					got = options
					return nil
				},
			}

			c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", nil), httptest.NewRecorder())
			c.Set("context", ctx.Context)

			err := convertRoute(converter, new(pdf.Processor)).Handler(c)
			if tc.wantErr {
				status, message := api.ParseError(err)
				if status != http.StatusBadRequest || !strings.Contains(message, "'paperWidth' and 'paperHeight'") {
					t.Fatalf("got %d %q, want a 400 naming both fields", status, message)
				}
				return
			}

			if err != nil {
				t.Fatalf("handler: %v", err)
			}

			if got.PaperWidth != 8 || got.PaperHeight == 0 {
				t.Errorf("paper = %gx%g, want 8 inches wide with a height", got.PaperWidth, got.PaperHeight)
			}
		})
	}
}

// An invalid PDF/A value is a client error caught before LibreOffice runs.
func TestConvertRoute_InvalidNativePdfa(t *testing.T) {
	dir := t.TempDir()

	ctx := &api.ContextMock{Context: &api.Context{Context: context.Background()}}
	ctx.SetDirPath(dir)
	ctx.SetFiles(map[string]string{"page_1.docx": zipPackage(t, dir, "page_1.docx")})
	ctx.SetValues(map[string][]string{"nativePdfa": {"foo"}})
	ctx.SetLogger(slog.New(slog.DiscardHandler))

	converter := &libreofficeapi.ApiMock{
		ExtensionsMock: func() []string { return []string{".docx"} },
		PdfMock: func(_ context.Context, _ *slog.Logger, _, _ string, _ libreofficeapi.Options) error {
			t.Fatal("LibreOffice ran with an invalid PDF/A value")
			return nil
		},
	}

	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", nil), httptest.NewRecorder())
	c.Set("context", ctx.Context)

	err := convertRoute(converter, new(pdf.Processor)).Handler(c)
	status, message := api.ParseError(err)
	if status != http.StatusBadRequest || !strings.Contains(message, "'nativePdfa'") {
		t.Fatalf("got %d %q, want a 400 naming 'nativePdfa'", status, message)
	}
}

// TestConvertRoute_Watermark checks that the rotation stays unset unless the
// client sets one, 0 included, and that the plain and tiled watermarks exclude
// each other, as LibreOffice draws the tiled one only without the plain one.
func TestConvertRoute_Watermark(t *testing.T) {
	dir := t.TempDir()

	for _, tc := range []struct {
		name      string
		values    map[string][]string
		wantAngle *int
		wantErr   string
	}{
		{name: "rotation unset", values: map[string][]string{"nativeWatermarkText": {"DRAFT"}}},
		{name: "rotation 0", values: map[string][]string{"nativeWatermarkRotation": {"0"}}, wantAngle: new(0)},
		{name: "rotation 360", values: map[string][]string{"nativeWatermarkRotation": {"360"}}, wantAngle: new(0)},
		{name: "rotation 45", values: map[string][]string{"nativeWatermarkRotation": {"45"}}, wantAngle: new(450)},
		{
			name:    "plain and tiled watermarks",
			values:  map[string][]string{"nativeWatermarkText": {"DRAFT"}, "nativeTiledWatermarkText": {"DRAFT"}},
			wantErr: "Invalid form data: set 'nativeWatermarkText' or 'nativeTiledWatermarkText', not both",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &api.ContextMock{Context: &api.Context{Context: context.Background()}}
			ctx.SetDirPath(dir)
			ctx.SetFiles(map[string]string{"page_1.docx": zipPackage(t, dir, "page_1.docx")})
			ctx.SetValues(tc.values)
			ctx.SetLogger(slog.New(slog.DiscardHandler))

			var got libreofficeapi.Options
			converter := &libreofficeapi.ApiMock{
				ExtensionsMock: func() []string { return []string{".docx"} },
				PdfMock: func(_ context.Context, _ *slog.Logger, _, _ string, options libreofficeapi.Options) error {
					got = options
					return nil
				},
			}

			c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", nil), httptest.NewRecorder())
			c.Set("context", ctx.Context)

			err := convertRoute(converter, new(pdf.Processor)).Handler(c)
			if tc.wantErr != "" {
				status, message := api.ParseError(err)
				if status != http.StatusBadRequest || message != tc.wantErr {
					t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("handler: %v", err)
			}

			switch {
			case tc.wantAngle == nil && got.NativeWatermarkRotateAngle != nil:
				t.Errorf("rotation = %d, want unset", *got.NativeWatermarkRotateAngle)
			case tc.wantAngle != nil && got.NativeWatermarkRotateAngle == nil:
				t.Errorf("rotation unset, want %d", *tc.wantAngle)
			case tc.wantAngle != nil && *got.NativeWatermarkRotateAngle != *tc.wantAngle:
				t.Errorf("rotation = %d, want %d", *got.NativeWatermarkRotateAngle, *tc.wantAngle)
			}
		})
	}
}

// TestConvertRoute_UnknownFields checks that the route answers 400 for a field
// it does not read, and points a Gotenberg 8 one to its replacement, before
// LibreOffice runs.
func TestConvertRoute_UnknownFields(t *testing.T) {
	dir := t.TempDir()

	for _, tc := range []struct {
		name       string
		values     map[string][]string
		fileFields []string
		wantBody   string
	}{
		{
			name:     "misspelled field",
			values:   map[string][]string{"landscpae": {"true"}},
			wantBody: "Invalid form data: form field 'landscpae' is unknown to this route",
		},
		{
			name:     "renamed field",
			values:   map[string][]string{"pdfa": {"PDF/A-2b"}},
			wantBody: `Invalid form data: form field 'pdfa' is unknown to this route; Gotenberg 9 renamed it 'nativePdfa', or add {"op":"remediate","profile":"PDF/A-2b"} to 'stages'`,
		},
		{
			name:     "PDF/A profile LibreOffice does not write",
			values:   map[string][]string{"pdfa": {"PDF/A-1a"}},
			wantBody: `Invalid form data: form field 'pdfa' is unknown to this route; Gotenberg 9 renamed it 'nativePdfa', which takes PDF/A-1b, PDF/A-2b, PDF/A-3b or PDF/A-4, or add {"op":"remediate","profile":"PDF/A-1a"} to 'stages'`,
		},
		{
			name:     "image quality",
			values:   map[string][]string{"imageQuality": {"50"}},
			wantBody: `Invalid form data: form field 'imageQuality' is unknown to this route; Gotenberg 9 lowers image quality at export: use 'nativeQuality' or 'nativeReduceImageResolution'; {"op":"optimize"} is lossless`,
		},
		{
			name:     "removed field",
			values:   map[string][]string{"firstPageOnLeft": {"true"}},
			wantBody: "Invalid form data: form field 'firstPageOnLeft' is unknown to this route; Gotenberg 9 removed it: LibreOffice ignores it",
		},
		{
			name:     "post-processing field",
			values:   map[string][]string{"merge": {"true"}},
			wantBody: `Invalid form data: form field 'merge' is unknown to this route; Gotenberg 9 moved it to a stage: add {"op":"merge"} to 'stages'`,
		},
		{
			name:       "Gotenberg 8 file field",
			fileFields: []string{"files", "embeds"},
			wantBody:   `Invalid form data: form file field 'embeds' is unknown to this route; Gotenberg 9 reads every file from 'files': upload it there, and add {"op":"attach","file":"data.xml"} to 'stages'`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &api.ContextMock{Context: &api.Context{Context: context.Background()}}
			ctx.SetDirPath(dir)
			ctx.SetFiles(map[string]string{"page_1.docx": zipPackage(t, dir, "page_1.docx")})
			ctx.SetValues(tc.values)
			ctx.SetFileFields(tc.fileFields...)
			ctx.SetLogger(slog.New(slog.DiscardHandler))

			converter := &libreofficeapi.ApiMock{
				ExtensionsMock: func() []string { return []string{".docx"} },
				PdfMock: func(_ context.Context, _ *slog.Logger, _, _ string, _ libreofficeapi.Options) error {
					t.Fatal("LibreOffice ran with an unknown form field")
					return nil
				},
			}

			c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/forms/libreoffice/convert", nil), httptest.NewRecorder())
			c.Set("context", ctx.Context)

			status, message := api.ParseError(convertRoute(converter, new(pdf.Processor)).Handler(c))
			if status != http.StatusBadRequest || message != tc.wantBody {
				t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantBody)
			}
		})
	}
}
