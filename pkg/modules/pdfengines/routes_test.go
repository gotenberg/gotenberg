package pdfengines

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestMergeHandler(t *testing.T) {
	tests := []struct {
		name                   string
		ctx                    *api.MockContext
		engine                 gotenberg.PDFEngine
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			name: "nominal behavior",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
			},
			expectOutputPathsCount: 1,
		},
		{
			name:             "invalid form data: no PDF",
			ctx:              &api.MockContext{Context: &api.Context{}},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "merge fail",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectErr: true,
		},
		{
			name: "nominal behavior with a PDF format",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return nil
				},
			},
			expectOutputPathsCount: 1,
		},
		{
			name: "convert to PDF format fail",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectErr: true,
		},
		{
			name: "invalid PDF format",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return gotenberg.ErrPDFFormatNotAvailable
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "cannot add output paths",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetCancelled(true)

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := mergeRoute(tc.engine).Handler(c)

			if tc.expectErr && err == nil {
				t.Fatal("expected error from merge handler, but got none")
			}

			if !tc.expectErr && err != nil {
				t.Fatalf("expected no error from merge handler, but got: %v", err)
			}

			var httpErr api.HTTPError
			isHTTPErr := errors.As(err, &httpErr)

			if tc.expectHTTPErr && !isHTTPErr {
				t.Errorf("expected HTTP error from merge handler, but got: %v", err)
			}

			if !tc.expectHTTPErr && isHTTPErr {
				t.Errorf("expected no HTTP error from merge handler, but got one: %v", httpErr)
			}

			if err != nil && tc.expectHTTPErr && isHTTPErr {
				status, _ := httpErr.HTTPError()
				if status != tc.expectHTTPStatus {
					t.Errorf("expected %d HTTP status code from merge handler, but got %d", tc.expectHTTPStatus, status)
				}
			}

			if tc.expectOutputPathsCount != len(tc.ctx.OutputPaths()) {
				t.Errorf("expected %d output paths from merge handler, but got %d", tc.expectOutputPathsCount, len(tc.ctx.OutputPaths()))
			}
		})
	}
}

func TestConvertHandler(t *testing.T) {
	tests := []struct {
		name                   string
		ctx                    *api.MockContext
		engine                 gotenberg.PDFEngine
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			name: "nominal behavior",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return nil
				},
			},
			expectOutputPathsCount: 1,
		},
		{
			name: "nominal behavior, but with 3 PDFs",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
					"bar.pdf": "/bar/bar.pdf",
					"baz.pdf": "/baz/baz.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return nil
				},
			},
			expectOutputPathsCount: 3,
		},
		{
			name: "invalid form data: no PDF",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "invalid form data: no PDF format",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "convert to PDF format fail",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectErr: true,
		},
		{
			name: "PDF format not available",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return gotenberg.ErrPDFFormatNotAvailable
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "cannot add output paths",
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})
				ctx.SetCancelled(true)

				return ctx
			}(),
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return nil
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertRoute(tc.engine).Handler(c)

			if tc.expectErr && err == nil {
				t.Fatal("expected error from convert handler, but got none")
			}

			if !tc.expectErr && err != nil {
				t.Fatalf("expected no error from convert handler, but got: %v", err)
			}

			var httpErr api.HTTPError
			isHTTPErr := errors.As(err, &httpErr)

			if tc.expectHTTPErr && !isHTTPErr {
				t.Errorf("expected HTTP error from convert handler, but got: %v", err)
			}

			if !tc.expectHTTPErr && isHTTPErr {
				t.Errorf("expected no HTTP error from convert handler, but got one: %v", httpErr)
			}

			if err != nil && tc.expectHTTPErr && isHTTPErr {
				status, _ := httpErr.HTTPError()
				if status != tc.expectHTTPStatus {
					t.Errorf("expected %d HTTP status code from convert handler, but got %d", tc.expectHTTPStatus, status)
				}
			}

			if tc.expectOutputPathsCount != len(tc.ctx.OutputPaths()) {
				t.Errorf("expected %d output paths from convert handler, but got %d", tc.expectOutputPathsCount, len(tc.ctx.OutputPaths()))
			}
		})
	}
}
