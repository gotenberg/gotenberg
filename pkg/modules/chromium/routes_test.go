package chromium

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
)

func TestFormDataChromiumPdfOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario        string
		ctx             *api.ContextMock
		expectedOptions Options
	}{
		{
			scenario:        "no custom form fields",
			ctx:             &api.ContextMock{Context: new(api.Context)},
			expectedOptions: DefaultOptions(),
		},
		{
			scenario: "deprecated userAgent form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"userAgent": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.ExtraHttpHeaders = map[string]string{
					"User-Agent": "foo",
				}
				return options
			}(),
		},
		{
			scenario: "invalid extraHttpHeaders form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions: DefaultOptions(),
		},
		{
			scenario: "valid extraHttpHeaders form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						`{"foo":"bar"}`,
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.ExtraHttpHeaders = map[string]string{
					"foo": "bar",
				}
				return options
			}(),
		},
		{
			scenario: "invalid emulatedMediaType form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"emulatedMediaType": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions: DefaultOptions(),
		},
		{
			scenario: "valid emulatedMediaType form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"emulatedMediaType": {
						"screen",
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.EmulatedMediaType = "screen"
				return options
			}(),
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			_, actual := FormDataChromiumPdfOptions(tc.ctx.Context)

			if !reflect.DeepEqual(actual, tc.expectedOptions) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedOptions, actual)
			}
		})
	}
}

func TestFormDataChromiumPdfFormats(t *testing.T) {
	for _, tc := range []struct {
		scenario           string
		ctx                *api.ContextMock
		expectedPdfFormats gotenberg.PdfFormats
	}{
		{
			scenario:           "no custom form fields",
			ctx:                &api.ContextMock{Context: new(api.Context)},
			expectedPdfFormats: gotenberg.PdfFormats{},
		},
		{
			scenario: "deprecated pdfFormat form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"pdfFormat": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedPdfFormats: gotenberg.PdfFormats{PdfA: "foo"},
		},
		{
			scenario: "pdfa and pdfua form fields",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"pdfa": {
						"foo",
					},
					"pdfua": {
						"true",
					},
				})
				return ctx
			}(),
			expectedPdfFormats: gotenberg.PdfFormats{PdfA: "foo", PdfUa: true},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			actual := FormDataChromiumPdfFormats(tc.ctx.Context)

			if !reflect.DeepEqual(actual, tc.expectedPdfFormats) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedPdfFormats, actual)
			}
		})
	}
}

func TestConvertUrlRoute(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		api                    Api
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
	}{
		{
			scenario:               "missing mandatory url form field",
			ctx:                    &api.ContextMock{Context: new(api.Context)},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "empty url form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"url": {
						"",
					},
				})
				return ctx
			}(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from Chromium",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"url": {
						"foo",
					},
				})
				return ctx
			}(),
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return errors.New("foo")
			}},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"url": {
						"foo",
					},
				})
				return ctx
			}(),
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertUrlRoute(tc.api, nil).Handler(c)

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

func TestConvertHtmlRoute(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		api                    Api
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
	}{
		{
			scenario:               "missing mandatory index.html form file",
			ctx:                    &api.ContextMock{Context: new(api.Context)},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from Chromium",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"index.html": "/index.html",
				})
				return ctx
			}(),
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return errors.New("foo")
			}},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"index.html": "/index.html",
				})
				return ctx
			}(),
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertHtmlRoute(tc.api, nil).Handler(c)

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

func TestConvertMarkdownRoute(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		api                    Api
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
	}{
		{
			scenario:               "missing mandatory index.html form file",
			ctx:                    &api.ContextMock{Context: new(api.Context)},
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "missing mandatory markdown form files",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"index.html": "/index.html",
				})
				return ctx
			}(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "markdown file requested in index.html not found",
			ctx: func() *api.ContextMock {
				dirPath := fmt.Sprintf("%s/%s", os.TempDir(), uuid.NewString())
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetDirPath(dirPath)
				ctx.SetFiles(map[string]string{
					"index.html":    fmt.Sprintf("%s/index.html", dirPath),
					"wrong_name.md": fmt.Sprintf("%s/wrong_name.md", dirPath),
				})

				err := os.MkdirAll(dirPath, 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", dirPath), []byte("<div>{{ toHTML \"markdown.md\" }}</div>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return ctx
			}(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "non-existing markdown file",
			ctx: func() *api.ContextMock {
				dirPath := fmt.Sprintf("%s/%s", os.TempDir(), uuid.NewString())
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetDirPath(dirPath)
				ctx.SetFiles(map[string]string{
					"index.html":  fmt.Sprintf("%s/index.html", dirPath),
					"markdown.md": fmt.Sprintf("%s/markdown.md", dirPath),
				})

				err := os.MkdirAll(dirPath, 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", dirPath), []byte("<div>{{ toHTML \"markdown.md\" }}</div>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return ctx
			}(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from Chromium",
			ctx: func() *api.ContextMock {
				dirPath := fmt.Sprintf("%s/%s", os.TempDir(), uuid.NewString())
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetDirPath(dirPath)
				ctx.SetFiles(map[string]string{
					"index.html":  fmt.Sprintf("%s/index.html", dirPath),
					"markdown.md": fmt.Sprintf("%s/markdown.md", dirPath),
				})

				err := os.MkdirAll(dirPath, 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", dirPath), []byte("<div>{{ toHTML \"markdown.md\" }}</div>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/markdown.md", dirPath), []byte("# Hello World!"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return ctx
			}(),
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return errors.New("foo")
			}},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success",
			ctx: func() *api.ContextMock {
				dirPath := fmt.Sprintf("%s/%s", os.TempDir(), uuid.NewString())
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetDirPath(dirPath)
				ctx.SetFiles(map[string]string{
					"index.html":  fmt.Sprintf("%s/index.html", dirPath),
					"markdown.md": fmt.Sprintf("%s/markdown.md", dirPath),
				})

				err := os.MkdirAll(dirPath, 0o755)
				if err != nil {
					t.Fatalf(fmt.Sprintf("expected no error but got: %v", err))
				}

				err = os.WriteFile(fmt.Sprintf("%s/index.html", dirPath), []byte("<div>{{ toHTML \"markdown.md\" }}</div>"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				err = os.WriteFile(fmt.Sprintf("%s/markdown.md", dirPath), []byte("# Hello World!"), 0o755)
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return ctx
			}(),
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.ctx.DirPath() != "" {
				defer func() {
					err := os.RemoveAll(tc.ctx.DirPath())
					if err != nil {
						t.Fatalf("expected no error but got: %v", err)
					}
				}()
			}

			tc.ctx.SetLogger(zap.NewNop())
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := convertMarkdownRoute(tc.api, nil).Handler(c)

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

func TestConvertUrl(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		api                    Api
		engine                 gotenberg.PdfEngine
		pdfFormats             gotenberg.PdfFormats
		options                Options
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
	}{
		{
			scenario: "ErrUrlNotAuthorized",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return ErrUrlNotAuthorized
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusForbidden,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrOmitBackgroundWithoutPrintBackground",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return ErrOmitBackgroundWithoutPrintBackground
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidEvaluationExpression (without waitForExpression form field)",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return ErrInvalidEvaluationExpression
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidEvaluationExpression (with waitForExpression form field)",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return ErrInvalidEvaluationExpression
			}},
			options: func() Options {
				options := DefaultOptions()
				options.WaitForExpression = "foo"

				return options
			}(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidPrinterSettings",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return ErrInvalidPrinterSettings
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrPageRangesSyntaxError",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return ErrPageRangesSyntaxError
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrConsoleExceptions",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return ErrConsoleExceptions
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from Chromium",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return errors.New("foo")
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrPdfFormatNotSupported",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
				return gotenberg.ErrPdfFormatNotSupported
			}},
			pdfFormats:             gotenberg.PdfFormats{PdfA: "foo"},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from PDF engine",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
				return errors.New("foo")
			}},
			pdfFormats:             gotenberg.PdfFormats{PdfA: "foo"},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success with pdfFormat form field",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
				return nil
			}},
			pdfFormats:             gotenberg.PdfFormats{PdfA: gotenberg.PdfA1a},
			options:                DefaultOptions(),
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetCancelled(true)
				return ctx
			}(),
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			options:                DefaultOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{func(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
				return nil
			}},
			options:                DefaultOptions(),
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			err := convertUrl(tc.ctx.Context, tc.api, tc.engine, "", tc.pdfFormats, tc.options)

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
