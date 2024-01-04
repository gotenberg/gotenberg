package libreoffice

import (
	"context"
	"errors"
	"net/http"
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
			scenario: "ErrMalformedPageRanges",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				return ctx
			}(),
			libreOffice: &libreofficeapi.ApiMock{
				PdfMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options libreofficeapi.Options) error {
					return libreofficeapi.ErrMalformedPageRanges
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
			scenario: "ErrPdfFormatNotSupported (single file)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfa": {
						"foo",
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
					return gotenberg.ErrPdfFormatNotSupported
				},
			},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "PDF engine convert error (single file)",
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
			scenario: "cannot add output paths (single file)",
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
					"document.docx": "/document.docx",
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
			scenario: "success (many files)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx":  "/document.docx",
					"document2.docx": "/document2.docx",
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
			expectOutputPathsCount: 2,
		},
		{
			scenario: "success with non-native PDF/A & PDF/UA (single file)",
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
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "success with native PDF/A & PDF/UA (single file)",
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
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "merge error",
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
			scenario: "ErrPdfFormatNotSupported (merge)",
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
					"pdfa": {
						"foo",
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
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return gotenberg.ErrPdfFormatNotSupported
				},
			},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "PDF engine convert error (merge)",
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
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "cannot add output paths (merge)",
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
			engine: &gotenberg.PdfEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success (merge)",
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
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "success with non-native PDF/A & PDF/UA (merge)",
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
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"pdfua": {
						"true",
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
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "success with non-native PDF/A & PDF/UA (merge)",
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
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"pdfua": {
						"true",
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
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
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
		})
	}
}
