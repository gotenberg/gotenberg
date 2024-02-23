package pdfengines

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestMergeHandler(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		engine                 gotenberg.PdfEngine
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
	}{
		{
			scenario:               "missing at least one mandatory file",
			ctx:                    &api.ContextMock{Context: new(api.Context)},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from PDF engine",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
				})
				return ctx
			}(),
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
			scenario: "cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
				})
				ctx.SetCancelled(true)
				return ctx
			}(),
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
			scenario: "success",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
				})
				return ctx
			}(),
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
			scenario: "error from PDF engine (convert)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfa": {
						gotenberg.PdfA1b,
					},
				})
				return ctx
			}(),
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
			scenario: "success with PDF/A & PDF/UA form fields",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
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

			err := mergeRoute(tc.engine).Handler(c)

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

func TestConvertHandler(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		engine                 gotenberg.PdfEngine
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
		expectOutputPaths      []string
	}{
		{
			scenario:               "missing at least one mandatory file",
			ctx:                    &api.ContextMock{Context: new(api.Context)},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "no PDF formats",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				return ctx
			}(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from PDF engine",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfa": {
						gotenberg.PdfA1b,
					},
				})
				return ctx
			}(),
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
			scenario: "cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfa": {
						gotenberg.PdfA1b,
					},
				})
				ctx.SetCancelled(true)
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success with PDF/A & PDF/UA form fields (single file)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
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
			scenario: "success with PDF/A & PDF/UA form fields (many files)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
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
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 2,
			expectOutputPaths:      []string{"/file.pdf", "/file2.pdf"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertRoute(tc.engine).Handler(c)

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
