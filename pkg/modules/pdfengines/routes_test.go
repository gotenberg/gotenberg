package pdfengines

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"go.uber.org/zap"
)

func TestMergeHandler(t *testing.T) {
	for i, tc := range []struct {
		ctx                    *api.MockContext
		engine                 gotenberg.PDFEngine
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			ctx:              &api.MockContext{Context: &api.Context{}},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})

				return ctx
			}(),
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
						return errors.New("foo")
					},
				}
			}(),
			expectErr: true,
		},
		{
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
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
						return nil
					},
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return gotenberg.ErrPDFFormatNotAvailable
					},
				}
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
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
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
						return nil
					},
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return errors.New("foo")
					},
				}
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetCancelled(true)
				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})

				return ctx
			}(),
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
						return nil
					},
				}
			}(),
			expectErr: true,
		},
		{
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
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
						return nil
					},
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return nil
					},
				}
			}(),
			expectOutputPathsCount: 1,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
				})

				return ctx
			}(),
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					merge: func(_ context.Context, _ *zap.Logger, _ []string, _ string) error {
						return nil
					},
				}
			}(),
			expectOutputPathsCount: 1,
		},
	} {
		err := mergeRoute(tc.engine).Handler(tc.ctx.Context)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		var httpErr api.HTTPError
		isHTTPErr := errors.As(err, &httpErr)

		if tc.expectHTTPErr && !isHTTPErr {
			t.Errorf("test %d: expected HTTP error but got: %v", i, err)
		}

		if !tc.expectHTTPErr && isHTTPErr {
			t.Errorf("test %d: expected no HTTP error but got one: %v", i, httpErr)
		}

		if err != nil && tc.expectHTTPErr && isHTTPErr {
			status, _ := httpErr.HTTPError()
			if status != tc.expectHTTPStatus {
				t.Errorf("test %d: expected %d HTTP status code but got %d", i, tc.expectHTTPStatus, status)
			}
		}

		if tc.expectOutputPathsCount != len(tc.ctx.OutputPaths()) {
			t.Errorf("test %d: expected %d output paths but got %d", i, tc.expectOutputPathsCount, len(tc.ctx.OutputPaths()))
		}
	}
}

func TestConvertHandler(t *testing.T) {
	for i, tc := range []struct {
		ctx                    *api.MockContext
		engine                 gotenberg.PDFEngine
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			ctx:              &api.MockContext{Context: &api.Context{}},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
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
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return gotenberg.ErrPDFFormatNotAvailable
					},
				}
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
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
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return errors.New("foo")
					},
				}
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetCancelled(true)
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
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return nil
					},
				}
			}(),
			expectErr: true,
		},
		{
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
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return nil
					},
				}
			}(),
			expectOutputPathsCount: 1,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetFiles(map[string]string{
					"foo.pdf": "/foo/foo.pdf",
					"bar.pdf": "/foo/bar.pdf",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return nil
					},
				}
			}(),
			expectOutputPathsCount: 2,
		},
	} {
		err := convertRoute(tc.engine).Handler(tc.ctx.Context)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		var httpErr api.HTTPError
		isHTTPErr := errors.As(err, &httpErr)

		if tc.expectHTTPErr && !isHTTPErr {
			t.Errorf("test %d: expected HTTP error but got: %v", i, err)
		}

		if !tc.expectHTTPErr && isHTTPErr {
			t.Errorf("test %d: expected no HTTP error but got one: %v", i, httpErr)
		}

		if err != nil && tc.expectHTTPErr && isHTTPErr {
			status, _ := httpErr.HTTPError()
			if status != tc.expectHTTPStatus {
				t.Errorf("test %d: expected %d HTTP status code but got %d", i, tc.expectHTTPStatus, status)
			}
		}

		if tc.expectOutputPathsCount != len(tc.ctx.OutputPaths()) {
			t.Errorf("test %d: expected %d output paths but got %d", i, tc.expectOutputPathsCount, len(tc.ctx.OutputPaths()))
		}
	}
}
