package libreoffice

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/uno"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestConvertHandler(t *testing.T) {
	tests := []struct {
		name                   string
		ctx                    *api.ContextMock
		unoAPI                 uno.API
		engine                 gotenberg.PDFEngine
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			name: "nominal behavior",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectOutputPathsCount: 1,
		},
		{
			name: "nominal behavior, but with 3 documents",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectOutputPathsCount: 3,
		},
		{
			name: "cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetCancelled(true)

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr: true,
		},
		{
			name: "invalid form data: no documents",
			ctx:  &api.ContextMock{Context: &api.Context{}},
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "invalid form data: both nativePdfA1aFormat and nativePdfFormat are set",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"nativePdfA1aFormat": {
						"true",
					},
					"nativePdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})
				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "invalid form data: both nativePdfA1aFormat and pdfFormat are set",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"nativePdfA1aFormat": {
						"true",
					},
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})
				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "invalid form data: both nativePdfFormat and pdfFormat are set",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"nativePdfFormat": {
						gotenberg.FormatPDFA1a,
					},
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "convert to PDF fail",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return errors.New("foo")
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr: true,
		},
		{
			name: "invalid page ranges",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return uno.ErrMalformedPageRanges
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "convert 3 documents and merge them",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
			},
			expectOutputPathsCount: 1,
		},
		{
			name: "merge fail",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectErr: true,
		},
		{
			name: "convert 3 documents, merge them, and convert them to a PDF format",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
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
			name: "convert 3 documents, merge them, but convert them to PDF format fail",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
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
			name: "convert 3 documents, merge them, but PDF format not available",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
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
			name: "convert 3 documents and merge them, but cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
				})
				ctx.SetCancelled(true)

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			engine: gotenberg.PDFEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
			},
			expectErr: true,
		},
		{
			name: "convert to PDF format",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return nil
				},
			},
			expectOutputPathsCount: 1,
		},
		{
			name: "convert to PDF format using nativePdfA1aFormat",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"nativePdfA1aFormat": {
						"true",
					},
				})
				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return nil
				},
			},
			expectOutputPathsCount: 1,
		},
		{
			name: "convert to PDF format fail",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			engine: gotenberg.PDFEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
					return errors.New("foo")
				},
			},
			expectErr: true,
		},
		{
			name: "PDF format not available",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
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
			name: "invalid form data: both nativePdfA1aFormat and htmlFormat are set",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"nativePdfA1aFormat": {
						"true",
					},
					"htmlFormat": {
						"true",
					},
				})
				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "invalid form data: both pdfFormat and htmlFormat are set",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						gotenberg.FormatPDFA1a,
					},
					"htmlFormat": {
						"true",
					},
				})
				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "invalid form data: both htmlFormat and nativePageRanges are set",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"htmlFormat": {
						"true",
					},
					"nativePageRanges": {
						"1-2",
					},
				})
				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "invalid form data: merge specified with multiple input files and htmlFormat",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/bar/bar.docx",
					"baz.docx": "/baz/baz.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
					"htmlFormat": {
						"true",
					},
				})
				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			unoAPI: uno.APIMock{
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			name: "nominal behavior with htmlFormat",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"htmlFormat": {
						"true",
					},
				})
				return ctx
			}(),
			unoAPI: uno.APIMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options uno.Options) error {
					return nil
				},
				ExtensionsMock: func() []string {
					return []string{
						".docx",
					}
				},
			},
			expectOutputPathsCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertRoute(tc.unoAPI, tc.engine).Handler(c)

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
