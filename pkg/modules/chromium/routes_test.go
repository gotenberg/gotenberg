package chromium

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/dlclark/regexp2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestFormDataChromiumOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario                string
		ctx                     *api.ContextMock
		expectedOptions         Options
		compareWithoutDeepEqual bool
		expectValidationError   bool
	}{
		{
			scenario:                "no custom form fields",
			ctx:                     &api.ContextMock{Context: new(api.Context)},
			expectedOptions:         DefaultOptions(),
			compareWithoutDeepEqual: false,
			expectValidationError:   false,
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
			compareWithoutDeepEqual: false,
			expectValidationError:   true,
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
			compareWithoutDeepEqual: false,
			expectValidationError:   false,
		},
		{
			scenario: "invalid failOnResourceHttpStatusCodes form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"failOnResourceHttpStatusCodes": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.FailOnResourceHttpStatusCodes = nil
				return options
			}(),
			compareWithoutDeepEqual: false,
			expectValidationError:   true,
		},
		{
			scenario: "valid failOnResourceHttpStatusCodes form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"failOnResourceHttpStatusCodes": {
						`[399,499,599]`,
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.FailOnResourceHttpStatusCodes = []int64{399, 499, 599}
				return options
			}(),
			compareWithoutDeepEqual: false,
			expectValidationError:   false,
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
			expectedOptions:         DefaultOptions(),
			compareWithoutDeepEqual: false,
			expectValidationError:   true,
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
			compareWithoutDeepEqual: false,
			expectValidationError:   true,
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
			compareWithoutDeepEqual: false,
			expectValidationError:   false,
		},
		{
			scenario: "invalid extraHttpHeaders form field: cannot unmarshall",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						"foo",
					},
				})
				return ctx
			}(),
			expectedOptions:         DefaultOptions(),
			compareWithoutDeepEqual: false,
			expectValidationError:   true,
		},
		{
			scenario: "invalid extraHttpHeaders form field: invalid scope",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						`{"foo":"bar;scope;;"}`,
					},
				})
				return ctx
			}(),
			expectedOptions:         DefaultOptions(),
			compareWithoutDeepEqual: false,
			expectValidationError:   true,
		},
		{
			scenario: "invalid extraHttpHeaders form field: invalid scope regex pattern",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						`{"foo":"bar;scope=*."}`,
					},
				})
				return ctx
			}(),
			expectedOptions:         DefaultOptions(),
			compareWithoutDeepEqual: false,
			expectValidationError:   true,
		},
		{
			scenario: "valid extraHttpHeaders form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"extraHttpHeaders": {
						`{"foo":"bar","baz":"qux;scope=https?:\\/\\/([a-zA-Z0-9-]+\\.)*qux\\.com\\/.*"}`,
					},
				})
				return ctx
			}(),
			expectedOptions: func() Options {
				options := DefaultOptions()
				options.ExtraHttpHeaders = []ExtraHttpHeader{
					{
						Name:  "foo",
						Value: "bar",
					},
					{
						Name:  "baz",
						Value: "qux",
						Scope: regexp2.MustCompile(`https?:\/\/([a-zA-Z0-9-]+\.)*qux\.com\/.*`, 0),
					},
				}
				return options
			}(),
			compareWithoutDeepEqual: true,
			expectValidationError:   false,
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
			expectedOptions:       DefaultOptions(),
			expectValidationError: true,
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
			expectValidationError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			form, actual := FormDataChromiumOptions(tc.ctx.Context)

			if tc.compareWithoutDeepEqual {
				if len(tc.expectedOptions.ExtraHttpHeaders) != len(actual.ExtraHttpHeaders) {
					t.Fatalf("expected %d extra HTTP headers, but got %d", len(tc.expectedOptions.ExtraHttpHeaders), len(actual.ExtraHttpHeaders))
				}

				sort.Slice(tc.expectedOptions.ExtraHttpHeaders, func(i, j int) bool {
					return tc.expectedOptions.ExtraHttpHeaders[i].Name < tc.expectedOptions.ExtraHttpHeaders[j].Name
				})
				sort.Slice(actual.ExtraHttpHeaders, func(i, j int) bool {
					return actual.ExtraHttpHeaders[i].Name < actual.ExtraHttpHeaders[j].Name
				})

				for i := range tc.expectedOptions.ExtraHttpHeaders {
					if tc.expectedOptions.ExtraHttpHeaders[i].Name != actual.ExtraHttpHeaders[i].Name {
						t.Fatalf("expected '%s' extra HTTP header, but got '%s'", tc.expectedOptions.ExtraHttpHeaders[i].Name, tc.expectedOptions.ExtraHttpHeaders[i].Name)
					}

					if tc.expectedOptions.ExtraHttpHeaders[i].Value != actual.ExtraHttpHeaders[i].Value {
						t.Fatalf("expected '%s' as value for extra HTTP header '%s', but got '%s'", tc.expectedOptions.ExtraHttpHeaders[i].Value, tc.expectedOptions.ExtraHttpHeaders[i].Name, actual.ExtraHttpHeaders[i].Value)
					}

					var expectedScope string
					if tc.expectedOptions.ExtraHttpHeaders[i].Scope != nil {
						expectedScope = tc.expectedOptions.ExtraHttpHeaders[i].Scope.String()
					}
					var actualScope string
					if actual.ExtraHttpHeaders[i].Scope != nil {
						actualScope = actual.ExtraHttpHeaders[i].Scope.String()
					}

					if expectedScope != actualScope {
						t.Fatalf("expected '%s' as scope for extra HTTP header '%s', but got '%s'", expectedScope, tc.expectedOptions.ExtraHttpHeaders[i].Name, actualScope)
					}
				}
			} else {
				if !reflect.DeepEqual(actual, tc.expectedOptions) {
					t.Fatalf("expected %+v but got: %+v", tc.expectedOptions, actual)
				}
			}

			err := form.Validate()

			if tc.expectValidationError && err == nil {
				t.Fatal("expected validation error but got none", err)
			}

			if !tc.expectValidationError && err != nil {
				t.Fatalf("expected no validation error but got: %v", err)
			}
		})
	}
}

func TestFormDataChromiumPdfOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario              string
		ctx                   *api.ContextMock
		expectedOptions       PdfOptions
		expectValidationError bool
	}{
		{
			scenario:              "no custom form fields",
			ctx:                   &api.ContextMock{Context: new(api.Context)},
			expectedOptions:       DefaultPdfOptions(),
			expectValidationError: false,
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
			expectValidationError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			form, actual := FormDataChromiumPdfOptions(tc.ctx.Context)

			if !reflect.DeepEqual(actual, tc.expectedOptions) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedOptions, actual)
			}

			err := form.Validate()

			if tc.expectValidationError && err == nil {
				t.Fatal("expected validation error but got none", err)
			}

			if !tc.expectValidationError && err != nil {
				t.Fatalf("expected no validation error but got: %v", err)
			}
		})
	}
}

func TestFormDataChromiumScreenshotOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario              string
		ctx                   *api.ContextMock
		expectedOptions       ScreenshotOptions
		expectValidationError bool
	}{
		{
			scenario:              "no custom form fields",
			ctx:                   &api.ContextMock{Context: new(api.Context)},
			expectedOptions:       DefaultScreenshotOptions(),
			expectValidationError: false,
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
			expectValidationError: true,
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
			expectValidationError: false,
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
			expectValidationError: false,
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
			expectValidationError: false,
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
			expectValidationError: true,
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
			expectValidationError: true,
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
			expectValidationError: true,
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
			expectValidationError: false,
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
			expectValidationError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			form, actual := FormDataChromiumScreenshotOptions(tc.ctx.Context)

			if !reflect.DeepEqual(actual, tc.expectedOptions) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedOptions, actual)
			}

			err := form.Validate()

			if tc.expectValidationError && err == nil {
				t.Fatal("expected validation error but got none", err)
			}

			if !tc.expectValidationError && err != nil {
				t.Fatalf("expected no validation error but got: %v", err)
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
					t.Fatalf("expected no error but got: %v", err)
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
					t.Fatalf("expected no error but got: %v", err)
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
					t.Fatalf("expected no error but got: %v", err)
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
					t.Fatalf("expected no error but got: %v", err)
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
					t.Fatalf("expected no error but got: %v", err)
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
					t.Fatalf("expected no error but got: %v", err)
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
					t.Fatalf("expected no error but got: %v", err)
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
					t.Fatalf("expected no error but got: %v", err)
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
		splitMode              gotenberg.SplitMode
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
			scenario: "ErrInvalidResourceHttpStatusCode",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrInvalidResourceHttpStatusCode
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
			scenario: "ErrLoadingFailed",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrLoadingFailed
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrResourceLoadingFailed",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return ErrResourceLoadingFailed
			}},
			options:                DefaultPdfOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
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
			scenario: "PDF engine split error",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
				return nil, errors.New("foo")
			}},
			options:                DefaultPdfOptions(),
			splitMode:              gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "success with split mode",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
				return []string{inputPath}, nil
			}},
			options:                DefaultPdfOptions(),
			splitMode:              gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1"},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 1,
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
			scenario: "success with split mode and PDF formats",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{PdfMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
				return nil
			}},
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{inputPath}, nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			options:                DefaultPdfOptions(),
			splitMode:              gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1"},
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
			tc.ctx.SetMkdirAll(&gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
				return nil
			}})
			tc.ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
				return nil
			}})
			err := convertUrl(tc.ctx.Context, tc.api, tc.engine, "", tc.options, tc.splitMode, tc.pdfFormats, tc.metadata)

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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
				return ErrInvalidHttpStatusCode
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrInvalidResourceHttpStatusCode",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrInvalidResourceHttpStatusCode
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
				return ErrConsoleExceptions
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrLoadingFailed",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrLoadingFailed
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusBadRequest,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "ErrResourceLoadingFailed",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
				return ErrResourceLoadingFailed
			}},
			options:                DefaultScreenshotOptions(),
			expectError:            true,
			expectHttpError:        true,
			expectHttpStatus:       http.StatusConflict,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from Chromium",
			ctx:      &api.ContextMock{Context: new(api.Context)},
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
			api: &ApiMock{ScreenshotMock: func(ctx context.Context, logger *zap.Logger, url string, outputPaths []string, options ScreenshotOptions) error {
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
