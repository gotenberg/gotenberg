package chromium

import (
	"context"
	"errors"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestFormDataChromiumPDFOptions(t *testing.T) {
	for i, tc := range []struct {
		ctx     *api.ContextMock
		options Options
	}{
		{
			ctx:     &api.ContextMock{Context: &api.Context{}},
			options: DefaultOptions(),
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						"foo",
					},
				})

				return ctx
			}(),
			options: DefaultOptions(),
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						`{"foo":"bar"}`,
					},
				})

				return ctx
			}(),
			options: func() Options {
				options := DefaultOptions()
				options.ExtraHTTPHeaders = map[string]string{
					"foo": "bar",
				}

				return options
			}(),
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						"foo",
					},
				})

				return ctx
			}(),
			options: DefaultOptions(),
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"emulatedMediaType": {
						"foo",
					},
				})

				return ctx
			}(),
			options: DefaultOptions(),
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"emulatedMediaType": {
						"screen",
					},
				})

				return ctx
			}(),
			options: func() Options {
				options := DefaultOptions()
				options.EmulatedMediaType = "screen"

				return options
			}(),
		},
	} {
		_, actual := FormDataChromiumPDFOptions(tc.ctx.Context)

		if !reflect.DeepEqual(actual, tc.options) {
			t.Errorf("test %d: expected %+v but got: %+v", i, tc.options, actual)
		}
	}
}

func TestConvertURLHandler(t *testing.T) {
	for i, tc := range []struct {
		ctx                    *api.ContextMock
		api                    API
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			ctx:              &api.ContextMock{Context: &api.Context{}},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"url": {
						"",
					},
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"url": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return errors.New("foo")
				}

				return chromiumAPI
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"url": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			expectOutputPathsCount: 1,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"url": {
						"foo",
					},
					"extraLinkTags": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"url": {
						"foo",
					},
					"extraScriptTags": {
						"foo",
					},
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetValues(map[string][]string{
					"url": {
						"foo",
					},
					"extraLinkTags": {
						`[{"href":"https://cdn.foo/foo.css"},{"href":"https://cdn.bar/bar.css"}]`,
					},
					"extraScriptTags": {
						`[{"src":"https://cdn.foo/foo.js"},{"src":"https://cdn.bar/bar.js"}]`,
					},
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			expectOutputPathsCount: 1,
		},
	} {
		c := echo.New().NewContext(nil, nil)
		c.Set("context", tc.ctx.Context)

		err := convertURLRoute(tc.api, nil).Handler(c)

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

func TestConvertHTMLHandler(t *testing.T) {
	for i, tc := range []struct {
		ctx                    *api.ContextMock
		api                    API
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			ctx:              &api.ContextMock{Context: &api.Context{}},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.html": "/foo/foo.html",
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"index.html": "/foo/foo.html",
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return errors.New("foo")
				}

				return chromiumAPI
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"index.html": "/foo/foo.html",
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			expectOutputPathsCount: 1,
		},
	} {
		c := echo.New().NewContext(nil, nil)
		c.Set("context", tc.ctx.Context)

		err := convertHTMLRoute(tc.api, nil).Handler(c)

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

func TestConvertMarkdownHandler(t *testing.T) {
	for i, tc := range []struct {
		ctx                    *api.ContextMock
		api                    API
		outputDir              string
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			ctx:              &api.ContextMock{Context: &api.Context{}},
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"foo.html": "/foo/foo.html",
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"index.html": "/foo/foo.html",
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"index.html":  "/foo/foo.html",
					"markdown.md": "/foo/markdown.md",
				})

				return ctx
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"index.html":   "/tests/test/testdata/chromium/markdown/sample2/index.html",
					"markdown1.md": "/foo/markdown1.md",
				})

				return ctx
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"index.html":   "/tests/test/testdata/chromium/markdown/sample1/index.html",
					"markdown1.md": "/foo/markdown1.md",
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return errors.New("foo")
				}

				return chromiumAPI
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetFiles(map[string]string{
					"index.html":   "/tests/test/testdata/chromium/markdown/sample1/index.html",
					"markdown1.md": "/tests/test/testdata/chromium/markdown/sample1/markdown1.md",
					"markdown2.md": "/tests/test/testdata/chromium/markdown/sample1/markdown2.md",
					"markdown3.md": "/tests/test/testdata/chromium/markdown/sample1/markdown3.md",
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return errors.New("foo")
				}

				return chromiumAPI
			}(),
			expectErr: true,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}

				ctx.SetDirPath("/tmp/foo")
				ctx.SetFiles(map[string]string{
					"index.html":   "/tests/test/testdata/chromium/markdown/sample1/index.html",
					"markdown1.md": "/tests/test/testdata/chromium/markdown/sample1/markdown1.md",
					"markdown2.md": "/tests/test/testdata/chromium/markdown/sample1/markdown2.md",
					"markdown3.md": "/tests/test/testdata/chromium/markdown/sample1/markdown3.md",
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return errors.New("foo")
				}

				return chromiumAPI
			}(),
			outputDir: "/tmp/foo",
			expectErr: true,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}

				ctx.SetDirPath("/tmp/foo")
				ctx.SetFiles(map[string]string{
					"index.html":   "/tests/test/testdata/chromium/markdown/sample1/index.html",
					"markdown1.md": "/tests/test/testdata/chromium/markdown/sample1/markdown1.md",
					"markdown2.md": "/tests/test/testdata/chromium/markdown/sample1/markdown2.md",
					"markdown3.md": "/tests/test/testdata/chromium/markdown/sample1/markdown3.md",
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			outputDir:              "/tmp/foo",
			expectOutputPathsCount: 1,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}

				ctx.SetDirPath("/tmp/foo")
				ctx.SetFiles(map[string]string{
					"index.html":   "/tests/test/testdata/chromium/markdown/sample3/index.html",
					"markdown1.md": "/tests/test/testdata/chromium/markdown/sample3/markdown.md",
				})

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			outputDir:              "/tmp/foo",
			expectOutputPathsCount: 1,
		},
	} {
		func() {
			if tc.outputDir != "" {
				err := os.MkdirAll(tc.outputDir, 0755)

				if err != nil {
					t.Fatalf("test %d: expected error but got: %v", i, err)
				}

				defer func() {
					err := os.RemoveAll(tc.outputDir)
					if err != nil {
						t.Fatalf("test %d: expected no error but got: %v", i, err)
					}
				}()
			}

			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertMarkdownRoute(tc.api, nil).Handler(c)

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
		}()
	}
}

func TestConvertURL(t *testing.T) {
	for i, tc := range []struct {
		ctx                    *api.ContextMock
		api                    API
		engine                 gotenberg.PDFEngine
		PDFformat              string
		options                Options
		expectErr              bool
		expectHTTPErr          bool
		expectHTTPStatus       int
		expectOutputPathsCount int
	}{
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return ErrURLNotAuthorized
				}

				return chromiumAPI
			}(),
			options:          DefaultOptions(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusForbidden,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return ErrOmitBackgroundWithoutPrintBackground
				}

				return chromiumAPI
			}(),
			options:          DefaultOptions(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return ErrInvalidEvaluationExpression
				}

				return chromiumAPI
			}(),
			options:   DefaultOptions(),
			expectErr: true,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return ErrInvalidEvaluationExpression
				}

				return chromiumAPI
			}(),
			options: func() Options {
				options := DefaultOptions()
				options.WaitForExpression = "foo"

				return options
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return ErrInvalidPrinterSettings
				}

				return chromiumAPI
			}(),
			options:          DefaultOptions(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return ErrPageRangesSyntaxError
				}

				return chromiumAPI
			}(),
			options:          DefaultOptions(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return ErrConsoleExceptions
				}

				return chromiumAPI
			}(),
			options:          DefaultOptions(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusConflict,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return errors.New("foo")
				}

				return chromiumAPI
			}(),
			options:   DefaultOptions(),
			expectErr: true,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return gotenberg.ErrPDFFormatNotAvailable
					},
				}
			}(),
			PDFformat:        "foo",
			options:          DefaultOptions(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return errors.New("foo")
					},
				}
			}(),
			PDFformat: "foo",
			options:   DefaultOptions(),
			expectErr: true,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			engine: func() gotenberg.PDFEngine {
				return &ProtoPDFEngine{
					convert: func(_ context.Context, _ *zap.Logger, _, _, _ string) error {
						return nil
					},
				}
			}(),
			PDFformat:              "foo",
			options:                DefaultOptions(),
			expectOutputPathsCount: 1,
		},
		{
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: &api.Context{}}
				ctx.SetCancelled(true)

				return ctx
			}(),
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			options:   DefaultOptions(),
			expectErr: true,
		},
		{
			ctx: &api.ContextMock{Context: &api.Context{}},
			api: func() API {
				chromiumAPI := struct{ ProtoAPI }{}
				chromiumAPI.pdf = func(_ context.Context, _ *zap.Logger, _, _ string, _ Options) error {
					return nil
				}

				return chromiumAPI
			}(),
			options:                DefaultOptions(),
			expectOutputPathsCount: 1,
		},
	} {
		err := convertURL(tc.ctx.Context, tc.api, tc.engine, "", tc.PDFformat, tc.options)

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
