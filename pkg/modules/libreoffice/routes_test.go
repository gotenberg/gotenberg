package libreoffice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	libreofficeapi "github.com/gotenberg/gotenberg/v8/pkg/modules/libreoffice/api"
)

func TestConvertRoute(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		libreOffice            libreofficeapi.Uno
		engine                 gotenberg.PdfEngine
		expectOptions          libreofficeapi.Options
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
		expectOutputPaths      []string
	}{
		{
			scenario: "missing at least one mandatory file",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			libreOffice: &libreofficeapi.ApiMock{ExtensionsMock: func() []string {
				return []string{".docx"}
			}},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "invalid quality form field (not an integer)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"quality": {
						"foo",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{ExtensionsMock: func() []string {
				return []string{".docx"}
			}},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "invalid quality form field (< 1)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"quality": {
						"0",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{ExtensionsMock: func() []string {
				return []string{".docx"}
			}},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "invalid quality form field (> 100)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"quality": {
						"101",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{ExtensionsMock: func() []string {
				return []string{".docx"}
			}},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "invalid maxImageResolution form field (not an integer)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"maxImageResolution": {
						"foo",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{ExtensionsMock: func() []string {
				return []string{".docx"}
			}},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "invalid maxImageResolution form field (not in range)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"maxImageResolution": {
						"1",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{ExtensionsMock: func() []string {
				return []string{".docx"}
			}},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "invalid metadata form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"foo",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{ExtensionsMock: func() []string {
				return []string{".docx"}
			}},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrPdfFormatNotSupported (nativePdfFormats)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return libreofficeapi.ErrInvalidPdfFormats
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrUnoException",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"nativePageRanges": {
						"foo",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return libreofficeapi.ErrUnoException
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrRuntimeException",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"password": {
						"invalid",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return libreofficeapi.ErrRuntimeException
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from LibreOffice",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return errors.New("foo")
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "PDF engine merge error",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx":  "/document.docx",
					"document2.docx": "/document2.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "PDF engine split error",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return nil, errors.New("foo")
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "PDF engine convert error",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"nativePdfFormats": {
						"false",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "PDF engine write metadata error",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
					return errors.New("foo")
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetCancelled(true)
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success (single file)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx":  "/document.docx",
					"document2.docx": "/document2.docx",
				})
				ctx.SetValues(map[string][]string{
					"quality": {
						"100",
					},
					"maxImageResolution": {
						"1200",
					},
					"merge": {
						"true",
					},
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"pdfua": {
						"true",
					},
					"nativePdfFormats": {
						"false",
					},
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "success (many files)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx":  "/document.docx",
					"document2.docx": "/document2.docx",
					"document2.doc":  "/document2.doc",
				})
				ctx.SetValues(map[string][]string{
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"pdfua": {
						"true",
					},
					"nativePdfFormats": {
						"false",
					},
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
					return nil
				}})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx", ".doc"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 3,
			expectOutputPaths:      []string{"/document.docx.pdf", "/document2.docx.pdf", "/document2.doc.pdf"},
		},
		{
			scenario: "success with native PDF/A & PDF/UA",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"pdfua": {
						"true",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "success with split (many files)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx":  "/document.docx",
					"document2.docx": "/document2.docx",
				})
				ctx.SetValues(map[string][]string{
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					inputPathNoExt := inputPath[:len(inputPath)-len(filepath.Ext(inputPath))]
					filenameNoExt := filepath.Base(inputPathNoExt)
					return []string{
						fmt.Sprintf(
							"%s/%s_%d.pdf",
							outputDirPath, filenameNoExt, 0,
						),
						fmt.Sprintf(
							"%s/%s_%d.pdf",
							outputDirPath, filenameNoExt, 1,
						),
					}, nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 4,
			expectOutputPaths:      []string{"/document_docx/document.docx_0.pdf", "/document_docx/document.docx_1.pdf", "/document2_docx/document2.docx_0.pdf", "/document2_docx/document2.docx_1.pdf"},
		},
		{
			scenario: "success with merge and split",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx":  "/document.docx",
					"document2.docx": "/document2.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					inputPathNoExt := inputPath[:len(inputPath)-len(filepath.Ext(inputPath))]
					filenameNoExt := filepath.Base(inputPathNoExt)
					return []string{
						fmt.Sprintf(
							"%s/%s_%d.pdf",
							outputDirPath, filenameNoExt, 0,
						),
						fmt.Sprintf(
							"%s/%s_%d.pdf",
							outputDirPath, filenameNoExt, 1,
						),
					}, nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 2,
		},
		{
			scenario: "success with split and native PDF/A & PDF/UA (many files)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx":  "/document.docx",
					"document2.docx": "/document2.docx",
				})
				ctx.SetValues(map[string][]string{
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"pdfua": {
						"true",
					},
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{".docx"}
				},
			},
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					inputPathNoExt := inputPath[:len(inputPath)-len(filepath.Ext(inputPath))]
					filenameNoExt := filepath.Base(inputPathNoExt)
					return []string{
						fmt.Sprintf(
							"%s/%s_%d.pdf",
							outputDirPath, filenameNoExt, 0,
						),
						fmt.Sprintf(
							"%s/%s_%d.pdf",
							outputDirPath, filenameNoExt, 1,
						),
					}, nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 4,
			expectOutputPaths:      []string{"/document_docx/document.docx_0.pdf", "/document_docx/document.docx_1.pdf", "/document2_docx/document2.docx_0.pdf", "/document2_docx/document2.docx_1.pdf"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			tc.ctx.SetMkdirAll(&gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
				return nil
			}})
			tc.ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
				return nil
			}})
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertRoute(tc.libreOffice, tc.engine).Handler(c)

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			var httpErr api.HttpError
			isHttpError := errors.As(err, &httpErr)

			if tc.expectHttpError && !isHttpError {
				t.Errorf("expected an HTTP error but got: %v", err)
			}

			if !tc.expectHttpError && isHttpError {
				t.Errorf("expected no HTTP error but got one: %v", httpErr)
			}

			if err != nil && tc.expectHttpError && isHttpError {
				status, _ := httpErr.HttpError()
				if status != tc.expectHttpStatus {
					t.Errorf("expected %d as HTTP status code but got %d", tc.expectHttpStatus, status)
				}
			}

			if tc.expectOutputPathsCount != len(tc.ctx.OutputPaths()) {
				t.Errorf("expected %d output paths but got %d", tc.expectOutputPathsCount, len(tc.ctx.OutputPaths()))
			}

			for _, path := range tc.expectOutputPaths {
				if !slices.Contains(tc.ctx.OutputPaths(), path) {
					t.Errorf("expected '%s' in output paths %v", path, tc.ctx.OutputPaths())
				}
			}
		})
	}
}
