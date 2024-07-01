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

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestFormDataChromiumOptions(t *testing.T) {
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
			scenario: "invalid failOnHttpStatusCodes form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"failOnHttpStatusCodes": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.FailOnHttpStatusCodes = nil
				return options
			}(),
		},
		{
			scenario: "valid failOnHttpStatusCodes form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"failOnHttpStatusCodes": {
						`[399,499,599]`,
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.FailOnHttpStatusCodes = []int64{399, 499, 599}
				return options
			}(),
		},
		{
			scenario: "invalid cookies form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"cookies": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions: DefaultOptions(),
		},
		{
			scenario: "invalid cookies form field (missing required values)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"cookies": {
						"[{}]",
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				// No validation in this method, so it still instantiates
				// an empty item.
				options.Cookies = []Cookie{{}}
				return options
			}(),
		},
		{
			scenario: "valid cookies form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"cookies": {
						`[{"name":"foo","value":"bar","domain":".foo.bar"}]`,
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.Cookies = []Cookie{{
					Name:   "foo",
					Value:  "bar",
					Domain: ".foo.bar",
				}}
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
			_, actual := FormDataChromiumOptions(tc.ctx.Context)

			if !reflect.DeepEqual(actual, tc.expectedOptions) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedOptions, actual)
			}
		})
	}
}

func TestFormDataChromiumPdfOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario        string
		ctx             *api.ContextMock
		expectedOptions PdfOptions
	}{
		{
			scenario:        "no custom form fields",
			ctx:             &api.ContextMock{Context: new(api.Context)},
			expectedOptions: DefaultPdfOptions(),
		},
		{
			scenario: "custom form fields (Options & PdfOptions)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"landscape": {
						"true",
					},
					"emulatedMediaType": {
						"screen",
					},
				})
				return ctx
			}(),
			expectedOptions: func() PdfOptions {
				options := DefaultPdfOptions()
				options.Landscape = true
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

func TestFormDataChromiumScreenshotOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario        string
		ctx             *api.ContextMock
		expectedOptions ScreenshotOptions
	}{
		{
			scenario:        "no custom form fields",
			ctx:             &api.ContextMock{Context: new(api.Context)},
			expectedOptions: DefaultScreenshotOptions(),
		},
		{
			scenario: "invalid format form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"format": {
						"gif",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Format = ""
				return options
			}(),
		},
		{
			scenario: "valid png format form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"format": {
						"png",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Format = "png"
				return options
			}(),
		},
		{
			scenario: "valid jpeg format form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"format": {
						"jpeg",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Format = "jpeg"
				return options
			}(),
		},
		{
			scenario: "valid webp format form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"format": {
						"webp",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Format = "webp"
				return options
			}(),
		},
		{
			scenario: "invalid quality form field (not an integer)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"quality": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Quality = 0
				return options
			}(),
		},
		{
			scenario: "invalid quality form field (< 0)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"quality": {
						"-1",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Quality = 0
				return options
			}(),
		},
		{
			scenario: "invalid quality form field (> 100)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"quality": {
						"101",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Quality = 0
				return options
			}(),
		},
		{
			scenario: "valid quality form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"quality": {
						"50",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Quality = 50
				return options
			}(),
		},
		{
			scenario: "custom form fields (Options & ScreenshotOptions)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"width": {
						"1280",
					},
					"height": {
						"800",
					},
					"clip": {
						"true",
					},
					"optimizeForSpeed": {
						"true",
					},
					"emulatedMediaType": {
						"screen",
					},
				})
				return ctx
			}(),
			expectedOptions: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.Width = 1280
				options.Height = 800
				options.Clip = true
				options.OptimizeForSpeed = true
				options.EmulatedMediaType = "screen"
				return options
			}(),
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			_, actual := FormDataChromiumScreenshotOptions(tc.ctx.Context)

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
			actual := FormDataChromiumPdfFormats(tc.ctx.Context.FormData())

			if !reflect.DeepEqual(actual, tc.expectedPdfFormats) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedPdfFormats, actual)
			}
		})
	}
}

func TestFormDataPdfMetadata(t *testing.T) {
	for _, tc := range []struct {
		scenario         string
		ctx              *api.ContextMock
		expectedMetadata map[string]interface{}
	}{
		{
			scenario:         "no metadata form field",
			ctx:              &api.ContextMock{Context: new(api.Context)},
			expectedMetadata: nil,
		},
		{
			scenario: "invalid metadata form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"metadata": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedMetadata: nil,
		},
		{
			scenario: "valid metadata form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{\"foo\":\"bar\"}",
					},
				})
				return ctx
			}(),
			expectedMetadata: map[string]interface{}{
				"foo": "bar",
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			actual := FormDataPdfMetadata(tc.ctx.Context.FormData())

			if !reflect.DeepEqual(actual, tc.expectedMetadata) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedMetadata, actual)
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
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
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
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
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

func TestScreenshotUrlRoute(t *testing.T) {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
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

			err := screenshotUrlRoute(tc.api).Handler(c)

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
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
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
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
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

func TestScreenshotHtmlRoute(t *testing.T) {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
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

			err := screenshotHtmlRoute(tc.api).Handler(c)

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
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
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
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
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

func TestScreenshotMarkdownRoute(t *testing.T) {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
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

			err := screenshotMarkdownRoute(tc.api).Handler(c)

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
		options                PdfOptions
		pdfFormats             gotenberg.PdfFormats
		metadata               map[string]interface{}
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
	}{
		{
			scenario: "ErrOmitBackgroundWithoutPrintBackground",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrOmitBackgroundWithoutPrintBackground
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidEvaluationExpression (without waitForExpression form field)",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrInvalidEvaluationExpression
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidEvaluationExpression (with waitForExpression form field)",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrInvalidEvaluationExpression
			}},
			options: func() PdfOptions {
				options := DefaultPdfOptions()
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
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrInvalidPrinterSettings
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrPageRangesSyntaxError",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrPageRangesSyntaxError
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidHttpStatusCode",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrInvalidHttpStatusCode
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrConsoleExceptions",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrConsoleExceptions
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrConnectionRefused",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrConnectionRefused
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from Chromium",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return errors.New("foo")
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "PDF engine convert error",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
				return errors.New("foo")
			}},
			options:                DefaultPdfOptions(),
			pdfFormats:             gotenberg.PdfFormats{PdfA: "foo"},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success with PDF formats",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
				return nil
			}},
			options:                DefaultPdfOptions(),
			pdfFormats:             gotenberg.PdfFormats{PdfA: gotenberg.PdfA1b},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
		{
			scenario: "PDF engine write metadata error",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
				return errors.New("foo")
			}},
			options: DefaultPdfOptions(),
			metadata: map[string]interface{}{
				"Creator":  "foo",
				"Producer": "bar",
			},
			expectError:     true,
			expectHttpError: false,
		},
		{
			scenario: "cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetCancelled(true)
				return ctx
			}(),
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
					return nil
				},
			},
			options:    DefaultPdfOptions(),
			pdfFormats: gotenberg.PdfFormats{PdfA: gotenberg.PdfA1b},
			metadata: map[string]interface{}{
				"Creator":  "foo",
				"Producer": "bar",
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			err := convertUrl(tc.ctx.Context, tc.api, tc.engine, "", tc.options, tc.pdfFormats, tc.metadata)

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

func TestScreenshotUrl(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    *api.ContextMock
		api                    Api
		options                ScreenshotOptions
		expectError            bool
		expectHttpError        bool
		expectHttpStatus       int
		expectOutputPathsCount int
	}{
		{
			scenario: "ErrInvalidEvaluationExpression (without waitForExpression form field)",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrInvalidEvaluationExpression
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidEvaluationExpression (with waitForExpression form field)",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrInvalidEvaluationExpression
			}},
			options: func() ScreenshotOptions {
				options := DefaultScreenshotOptions()
				options.WaitForExpression = "foo"

				return options
			}(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidHttpStatusCode",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrInvalidHttpStatusCode
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrConsoleExceptions",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrConsoleExceptions
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrConnectionRefused",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrConnectionRefused
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from Chromium",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return errors.New("foo")
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "cannot add output paths",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetCancelled(true)
				return ctx
			}(),
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return nil
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return nil
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			err := screenshotUrl(tc.ctx.Context, tc.api, "", tc.options)

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
