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
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
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
			err:        libreofficeapi.ErrRuntimeException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "The document 'protected_page_1.docx' is password-protected. Provide its password in the 'password' form field.",
		},
		{
			name:       "encrypted document, wrong password",
			inputPath:  protected,
			values:     map[string][]string{"password": {"bar"}},
			err:        libreofficeapi.ErrRuntimeException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "The password for the document 'protected_page_1.docx' is incorrect. Check the 'password' form field.",
		},
		{
			name:       "unencrypted document, password supplied",
			inputPath:  plain,
			values:     map[string][]string{"password": {"foo"}},
			err:        libreofficeapi.ErrUnoException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "The document 'page_1.docx' is not password-protected. Remove the 'password' form field.",
		},
		{
			name:       "inconclusive document, password supplied",
			inputPath:  legacy,
			values:     map[string][]string{"password": {"foo"}},
			err:        libreofficeapi.ErrUnoException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not open the document 'legacy.doc' with the given password. Check the 'password' form field, and omit it if the document is not password-protected.",
		},
		{
			name:       "malformed page ranges",
			inputPath:  plain,
			values:     map[string][]string{"nativePageRanges": {"foo"}},
			err:        libreofficeapi.ErrUnoException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not apply the page ranges 'foo' to the document 'page_1.docx'. Check the 'nativePageRanges' form field; valid values look like '1-4', '2' or '1,3,5-7'.",
		},
		{
			name:       "password evidence outranks page ranges",
			inputPath:  protected,
			values:     map[string][]string{"nativePageRanges": {"1-2"}},
			err:        libreofficeapi.ErrUnoException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "The document 'protected_page_1.docx' is password-protected. Provide its password in the 'password' form field.",
		},
		{
			name:       "page ranges do not excuse a runtime exception",
			inputPath:  plain,
			values:     map[string][]string{"nativePageRanges": {"1-2"}},
			err:        libreofficeapi.ErrRuntimeException,
			wantStatus: http.StatusInternalServerError,
			wantBody:   fmt.Sprintf(unattributableFailureMessage, "page_1.docx"),
		},
		{
			name:       "nothing implicated, uno exception",
			inputPath:  plain,
			err:        libreofficeapi.ErrUnoException,
			wantStatus: http.StatusInternalServerError,
			wantBody:   fmt.Sprintf(unattributableFailureMessage, "page_1.docx"),
		},
		{
			name:       "nothing implicated, runtime exception",
			inputPath:  plain,
			err:        libreofficeapi.ErrRuntimeException,
			wantStatus: http.StatusInternalServerError,
			wantBody:   fmt.Sprintf(unattributableFailureMessage, "page_1.docx"),
		},
		{
			name:       "detection cannot read the document",
			inputPath:  unreachable,
			err:        libreofficeapi.ErrUnoException,
			wantStatus: http.StatusInternalServerError,
			wantBody:   fmt.Sprintf(unattributableFailureMessage, "vanished.docx"),
		},
		{
			name:       "unreadable source",
			inputPath:  corrupted,
			err:        libreofficeapi.ErrIoException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not read the document 'corrupted.docx'. Ensure the file is not corrupted and that its extension matches its actual format.",
		},
		{
			name:       "rejected source",
			inputPath:  corrupted,
			err:        libreofficeapi.ErrIllegalArgumentException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice could not read the document 'corrupted.docx'. Ensure the file is not corrupted and that its extension matches its actual format.",
		},
		{
			name:       "unconvertible document",
			inputPath:  corrupted,
			err:        libreofficeapi.ErrCannotConvertException,
			wantStatus: http.StatusBadRequest,
			wantBody:   "LibreOffice read the document 'corrupted.docx' but could not convert it to PDF. The document may be corrupted or rely on an unsupported feature.",
		},
		{
			name:       "core dumped past the retry cap",
			inputPath:  plain,
			err:        libreofficeapi.ErrCoreDumped,
			wantStatus: http.StatusInternalServerError,
			wantBody:   http.StatusText(http.StatusInternalServerError),
		},
		{
			name:       "unmapped exit code",
			inputPath:  plain,
			err:        fmt.Errorf("convert to PDF: exit status 7"),
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

			uno := &libreofficeapi.ApiMock{
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

			err := convertRoute(uno, new(gotenberg.PdfEngineMock)).Handler(c)
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
