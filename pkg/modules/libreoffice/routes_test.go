package libreoffice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/exiftool"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/unoconv"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConvertHandler(t *testing.T) {
	for i, tc := range []struct {
		ctx                    *api.MockContext
		api                    unoconv.API
		engine                 gotenberg.PDFEngine
		exiftool               exiftool.API
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.extensions = func() []string {
					return []string{
						".foo",
					}
				}

				return unoconvAPI
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})
				ctx.SetValues(map[string][]string{
					"nativePdfA1aFormat": {
						"true",
					},
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return unoconv.ErrMalformedPageRanges
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return errors.New("foo")
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
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
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
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
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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

				ctx.SetLogger(zap.NewNop())

				ctx.SetCancelled(true)
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
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
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			expectErr: true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
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
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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

				ctx.SetLogger(zap.NewNop())

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
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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

				ctx.SetLogger(zap.NewNop())

				ctx.SetCancelled(true)
				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			expectErr: true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			expectOutputPathsCount: 2,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
					"metadata": {
						"{ \"Foo\": \"foo\", \"Bar\": \"bar\" }",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			exiftool: func() exiftool.API {
				exiftoolAPI := struct {
					ProtoExiftoolAPI
				}{}
				exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
					return nil, nil
				}
				exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
					return nil
				}
				return exiftoolAPI
			}(),
			expectOutputPathsCount: 1,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{ \"Foo\": \"foo\", \"Bar\": \"bar\" }",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			exiftool: func() exiftool.API {
				exiftoolAPI := struct {
					ProtoExiftoolAPI
				}{}
				exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
					return nil, nil
				}
				exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
					return nil
				}
				return exiftoolAPI
			}(),
			expectOutputPathsCount: 2,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{ \"Foo\": \"foo\", \"Bar\": \"bar\" ",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			exiftool: func() exiftool.API {
				exiftoolAPI := struct {
					ProtoExiftoolAPI
				}{}
				exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
					return nil, nil
				}
				exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
					return nil
				}
				return exiftoolAPI
			}(),
			expectErr:              true,
			expectHTTPErr:          true,
			expectHTTPStatus:       400,
			expectOutputPathsCount: 0,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{ \"Foo\": \"foo\", \"Bar\": \"bar\" }",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			exiftool: func() exiftool.API {
				exiftoolAPI := struct {
					ProtoExiftoolAPI
				}{}
				exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
					return nil, nil
				}
				exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
					return errors.New("foo")
				}
				return exiftoolAPI
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{ \"Foo\": \"foo\", \"Bar\": \"bar\" }",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			exiftool: func() exiftool.API {
				exiftoolAPI := struct {
					ProtoExiftoolAPI
				}{}
				exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
					return nil, nil
				}
				exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
					return &exiftool.MetadataValueTypeError{
						Entries: map[string]interface{}{"foo": "foo", "bar": "bar"},
					}
				}
				return exiftoolAPI
			}(),
			expectErr:              true,
			expectHTTPErr:          true,
			expectHTTPStatus:       400,
			expectOutputPathsCount: 0,
		},
		{
			ctx: func() *api.MockContext {
				ctx := &api.MockContext{Context: &api.Context{}}

				ctx.SetLogger(zap.NewNop())

				ctx.SetFiles(map[string]string{
					"foo.docx": "/foo/foo.docx",
					"bar.docx": "/foo/bar.docx",
				})
				ctx.SetValues(map[string][]string{
					"merge": {
						"true",
					},
					"metadata": {
						"{ \"Foo\": \"foo\", \"Bar\": \"bar\" }",
					},
				})

				return ctx
			}(),
			api: func() unoconv.API {
				unoconvAPI := struct{ ProtoUnoconvAPI }{}
				unoconvAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ unoconv.Options) error {
					return nil
				}
				unoconvAPI.extensions = func() []string {
					return []string{
						".docx",
					}
				}

				return unoconvAPI
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
			exiftool: func() exiftool.API {
				exiftoolAPI := struct {
					ProtoExiftoolAPI
				}{}
				exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
					return nil, nil
				}
				exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
					return &exiftool.MetadataValueTypeError{
						Entries: map[string]interface{}{"foo": "foo", "bar": "bar"},
					}
				}
				return exiftoolAPI
			}(),
			expectErr:              true,
			expectHTTPErr:          true,
			expectHTTPStatus:       400,
			expectOutputPathsCount: 0,
		},
	} {
		c := echo.New().NewContext(nil, nil)
		c.Set("context", tc.ctx.Context)

		err := convertRoute(tc.api, tc.engine, tc.exiftool).Handler(c)

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

func TestParseMetadata(t *testing.T) {
	for i, tc := range []struct {
		input     string
		expect    map[string]interface{}
		expectErr bool
	}{
		{
			input:  ``,
			expect: make(map[string]interface{}),
		},
		{
			input:  `{}`,
			expect: make(map[string]interface{}),
		},
		{
			input:     `{`,
			expectErr: true,
		},
		{
			input: `{ "foo": "foo" }`,
			expect: map[string]interface{}{
				"foo": "foo",
			},
		},
		{
			input: `{ "foo": "foo", "bar": "bar" }`,
			expect: map[string]interface{}{
				"foo": "foo",
				"bar": "bar",
			},
		},
		{
			input: `{ "foo": "foo", "bar": 123, "baz": 4.56, "qux": true, "quux": null }`,
			expect: map[string]interface{}{
				"foo":  "foo",
				"bar":  float64(123),
				"baz":  4.56,
				"qux":  true,
				"quux": nil,
			},
		},
		{
			input:     `{ "foo": "foo", "bar": 123, "baz": 4.56, "qux": true, "quux": null ] }`,
			expectErr: true,
		},
		{
			input: `{ "foo": [ "bar", "baz", "qux" ] }`,
			expect: func() map[string]interface{} {
				input := []string{"bar", "baz", "qux"}
				elements := make([]interface{}, len(input))
				for i := range elements {
					elements[i] = input[i]
				}
				return map[string]interface{}{
					"foo": elements,
				}
			}(),
		},
		{
			input: `{ "foo": { "bar": "bar", "baz": "baz" } }`,
			expect: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "bar",
					"baz": "baz",
				},
			},
		},
	} {
		actual, err := parseMetadata(tc.input, zap.NewNop())

		if !reflect.DeepEqual(actual, tc.expect) {
			t.Errorf("test %d: expected %+v but got: %+v", i, tc.expect, actual)
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestWriteMetadata(t *testing.T) {
	type Input struct {
		ctx         context.Context
		logger      *zap.Logger
		rawMetadata map[string]interface{}
		paths       []string
		exiftoolAPI exiftool.API
	}
	for i, tc := range []struct {
		input       Input
		expectErr   bool
		expectedErr error
	}{
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger:      zap.NewNop(),
				rawMetadata: make(map[string]interface{}),
				paths:       []string{},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return nil
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
				},
				paths: []string{},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return nil
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
				},
				paths: []string{"/foo"},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return nil
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
					"bar": "bar",
				},
				paths: []string{"/foo"},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return nil
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
					"bar": "bar",
				},
				paths: []string{"/foo", "/bar"},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return nil
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger:      zap.NewNop(),
				rawMetadata: make(map[string]interface{}),
				paths:       []string{},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return errors.New("foo")
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
				},
				paths: []string{},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return errors.New("foo")
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger:      zap.NewNop(),
				rawMetadata: make(map[string]interface{}),
				paths:       []string{"/foo"},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return errors.New("foo")
					}
					return exiftoolAPI
				}(),
			},
			expectErr:   true,
			expectedErr: errors.New("foo"),
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
					"bar": "bar",
				},
				paths: []string{},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return &exiftool.MetadataValueTypeError{}
					}
					return exiftoolAPI
				}(),
			},
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
					"bar": "bar",
				},
				paths: []string{"/foo", "/bar"},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return &exiftool.MetadataValueTypeError{
							Entries: map[string]interface{}{"foo": "foo", "bar": "bar"},
						}
					}
					return exiftoolAPI
				}(),
			},
			expectErr: true,
			expectedErr: api.WrapError(
				fmt.Errorf("write metadata: %w", &exiftool.MetadataValueTypeError{
					Entries: map[string]interface{}{"foo": "foo", "bar": "bar"},
				}),
				api.NewSentinelHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("Invalid metdata value types supplied by keys '%s'", []string{"foo", "bar"}),
				)),
		},
		{
			input: Input{
				ctx: func() *api.MockContext {
					ctx := &api.MockContext{Context: &api.Context{}}

					ctx.SetLogger(zap.NewNop())

					return ctx
				}(),
				logger: zap.NewNop(),
				rawMetadata: map[string]interface{}{
					"foo": "foo",
					"bar": "bar",
				},
				paths: []string{"/foo", "/bar"},
				exiftoolAPI: func() exiftool.API {
					exiftoolAPI := struct {
						ProtoExiftoolAPI
					}{}
					exiftoolAPI.readMetadata = func(_ context.Context, _ *zap.Logger, _ []string) (*[]exiftool.FileMetadata, error) {
						return nil, nil
					}
					exiftoolAPI.writeMetadata = func(_ context.Context, _ *zap.Logger, _ []string, _ *map[string]interface{}) error {
						return fmt.Errorf("foo: %w", gotenberg.ErrPDFFormatNotAvailable)
					}
					return exiftoolAPI
				}(),
			},
			expectErr:   true,
			expectedErr: fmt.Errorf("foo: %w", gotenberg.ErrPDFFormatNotAvailable),
		},
	} {
		err := writeMetadata(tc.input.ctx, tc.input.logger, tc.input.rawMetadata, tc.input.paths, tc.input.exiftoolAPI)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if tc.expectedErr != nil {
			valid := assert.IsType(t, err, tc.expectedErr)
			if !valid {
				t.Errorf("test %d: expected type %T but got: %T", i, tc.expectedErr, err)
			}
		}
	}
}
