package pdfengines

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestFormDataPdfSplitMode(t *testing.T) {
	for _, tc := range []struct {
		scenario              string
		ctx                   *api.ContextMock
		mandatory             bool
		expectedSplitMode     gotenberg.SplitMode
		expectValidationError bool
	}{
		{
			scenario:              "no custom form fields",
			ctx:                   &api.ContextMock{Context: new(api.Context)},
			mandatory:             false,
			expectedSplitMode:     gotenberg.SplitMode{},
			expectValidationError: false,
		},
		{
			scenario:              "no custom form fields (mandatory)",
			ctx:                   &api.ContextMock{Context: new(api.Context)},
			mandatory:             true,
			expectedSplitMode:     gotenberg.SplitMode{},
			expectValidationError: true,
		},
		{
			scenario: "invalid splitMode",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"splitMode": {
						"foo",
					},
				})
				return ctx
			}(),
			mandatory:             false,
			expectedSplitMode:     gotenberg.SplitMode{},
			expectValidationError: true,
		},
		{
			scenario: "invalid splitSpan (intervals)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"splitMode": {
						"intervals",
					},
					"splitSpan": {
						"1-2",
					},
				})
				return ctx
			}(),
			mandatory:             false,
			expectedSplitMode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals},
			expectValidationError: true,
		},
		{
			scenario: "splitSpan inferior to 1 (intervals)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"splitMode": {
						"intervals",
					},
					"splitSpan": {
						"-1",
					},
				})
				return ctx
			}(),
			mandatory:             false,
			expectedSplitMode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals},
			expectValidationError: true,
		},
		{
			scenario: "invalid splitUnify (intervals)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"splitMode": {
						"intervals",
					},
					"splitSpan": {
						"1",
					},
					"splitUnify": {
						"true",
					},
				})
				return ctx
			}(),
			mandatory:             false,
			expectedSplitMode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1", Unify: true},
			expectValidationError: true,
		},
		{
			scenario: "valid form fields (intervals)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"splitMode": {
						"intervals",
					},
					"splitSpan": {
						"1",
					},
				})
				return ctx
			}(),
			mandatory:             false,
			expectedSplitMode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			expectValidationError: false,
		},
		{
			scenario: "valid form fields (pages)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"splitMode": {
						"pages",
					},
					"splitSpan": {
						"1-2",
					},
					"splitUnify": {
						"true",
					},
				})
				return ctx
			}(),
			mandatory:             false,
			expectedSplitMode:     gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1-2", Unify: true},
			expectValidationError: false,
		},
		{
			scenario: "valid form fields (mandatory)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetValues(map[string][]string{
					"splitMode": {
						"intervals",
					},
					"splitSpan": {
						"1",
					},
				})
				return ctx
			}(),
			mandatory:             true,
			expectedSplitMode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			expectValidationError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			form := tc.ctx.Context.FormData()
			actual := FormDataPdfSplitMode(form, tc.mandatory)

			if !reflect.DeepEqual(actual, tc.expectedSplitMode) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedSplitMode, actual)
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

func TestFormDataPdfFormats(t *testing.T) {
	for _, tc := range []struct {
		scenario              string
		ctx                   *api.ContextMock
		expectedPdfFormats    gotenberg.PdfFormats
		expectValidationError bool
	}{
		{
			scenario:              "no custom form fields",
			ctx:                   &api.ContextMock{Context: new(api.Context)},
			expectedPdfFormats:    gotenberg.PdfFormats{},
			expectValidationError: false,
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
			expectedPdfFormats:    gotenberg.PdfFormats{PdfA: "foo", PdfUa: true},
			expectValidationError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			form := tc.ctx.Context.FormData()
			actual := FormDataPdfFormats(form)

			if !reflect.DeepEqual(actual, tc.expectedPdfFormats) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedPdfFormats, actual)
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

func TestFormDataPdfMetadata(t *testing.T) {
	for _, tc := range []struct {
		scenario              string
		ctx                   *api.ContextMock
		mandatory             bool
		expectedMetadata      map[string]interface{}
		expectValidationError bool
	}{
		{
			scenario:              "no metadata form field",
			ctx:                   &api.ContextMock{Context: new(api.Context)},
			mandatory:             false,
			expectedMetadata:      nil,
			expectValidationError: false,
		},
		{
			scenario:              "no metadata form field (mandatory)",
			ctx:                   &api.ContextMock{Context: new(api.Context)},
			mandatory:             true,
			expectedMetadata:      nil,
			expectValidationError: true,
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
			mandatory:             false,
			expectedMetadata:      nil,
			expectValidationError: true,
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
			mandatory: false,
			expectedMetadata: map[string]interface{}{
				"foo": "bar",
			},
			expectValidationError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			form := tc.ctx.Context.FormData()
			actual := FormDataPdfMetadata(form, tc.mandatory)

			if !reflect.DeepEqual(actual, tc.expectedMetadata) {
				t.Fatalf("expected %+v but got: %+v", tc.expectedMetadata, actual)
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

func TestMergeStub(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		engine      gotenberg.PdfEngine
		inputPaths  []string
		expectError bool
	}{
		{
			scenario:    "no input path (nil)",
			inputPaths:  nil,
			expectError: true,
		},
		{
			scenario:    "no input path (empty)",
			inputPaths:  make([]string, 0),
			expectError: true,
		},
		{
			scenario:    "only one input path",
			inputPaths:  []string{"my.pdf"},
			expectError: false,
		},
		{
			scenario: "merge error",
			engine: &gotenberg.PdfEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return errors.New("foo")
				},
			},
			inputPaths:  []string{"my.pdf", "my2.pdf"},
			expectError: true,
		},
		{
			scenario: "merge success",
			engine: &gotenberg.PdfEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
			},
			inputPaths:  []string{"my.pdf", "my2.pdf"},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			_, err := MergeStub(new(api.Context), tc.engine, tc.inputPaths)

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestSplitPdfStub(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         *api.ContextMock
		engine      gotenberg.PdfEngine
		mode        gotenberg.SplitMode
		expectError bool
	}{
		{
			scenario:    "no split mode",
			mode:        gotenberg.SplitMode{},
			ctx:         &api.ContextMock{Context: new(api.Context)},
			expectError: false,
		},
		{
			scenario: "cannot create subdirectory",
			mode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetMkdirAll(&gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
					return errors.New("cannot create subdirectory")
				}})
				return ctx
			}(),
			expectError: true,
		},
		{
			scenario: "split error",
			mode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetMkdirAll(&gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
					return nil
				}})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return nil, errors.New("foo")
				},
			},
			expectError: true,
		},
		{
			scenario: "rename error",
			mode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetMkdirAll(&gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
					return nil
				}})
				ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
					return errors.New("cannot rename")
				}})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{inputPath}, nil
				},
			},
			expectError: true,
		},
		{
			scenario: "success (intervals)",
			mode:     gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetMkdirAll(&gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
					return nil
				}})
				ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
					return nil
				}})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{inputPath}, nil
				},
			},
			expectError: false,
		},
		{
			scenario: "success (pages)",
			mode:     gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1-2", Unify: true},
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetMkdirAll(&gotenberg.MkdirAllMock{MkdirAllMock: func(path string, perm os.FileMode) error {
					return nil
				}})
				ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
					return nil
				}})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{inputPath}, nil
				},
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			dirPath := fmt.Sprintf("%s/%s", os.TempDir(), uuid.NewString())
			tc.ctx.SetDirPath(dirPath)
			tc.ctx.SetLogger(zap.NewNop())

			_, err := SplitPdfStub(tc.ctx.Context, tc.engine, tc.mode, []string{"my.pdf", "my2.pdf"})

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestConvertStub(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		engine      gotenberg.PdfEngine
		pdfFormats  gotenberg.PdfFormats
		expectError bool
	}{
		{
			scenario:    "no PDF formats",
			pdfFormats:  gotenberg.PdfFormats{},
			expectError: false,
		},
		{
			scenario: "convert error",
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return errors.New("foo")
				},
			},
			pdfFormats: gotenberg.PdfFormats{
				PdfA:  gotenberg.PdfA3b,
				PdfUa: true,
			},
			expectError: true,
		},
		{
			scenario: "convert success",
			engine: &gotenberg.PdfEngineMock{
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
			},
			pdfFormats: gotenberg.PdfFormats{
				PdfA:  gotenberg.PdfA3b,
				PdfUa: true,
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			_, err := ConvertStub(new(api.Context), tc.engine, tc.pdfFormats, []string{"my.pdf", "my2.pdf"})

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestWriteMetadataStub(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		engine      gotenberg.PdfEngine
		metadata    map[string]interface{}
		expectError bool
	}{
		{
			scenario:    "no metadata (nil)",
			metadata:    nil,
			expectError: false,
		},
		{
			scenario:    "no metadata (empty)",
			metadata:    make(map[string]interface{}, 0),
			expectError: false,
		},
		{
			scenario: "write metadata error",
			engine: &gotenberg.PdfEngineMock{
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
					return errors.New("foo")
				},
			},
			metadata:    map[string]interface{}{"foo": "bar"},
			expectError: true,
		},
		{
			scenario: "write metadata success",
			engine: &gotenberg.PdfEngineMock{
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
					return nil
				},
			},
			metadata:    map[string]interface{}{"foo": "bar"},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			err := WriteMetadataStub(new(api.Context), tc.engine, tc.metadata, []string{"my.pdf", "my2.pdf"})

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

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
			scenario: "invalid metadata form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"foo",
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
			scenario: "PDF engine merge error",
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
			scenario: "PDF engine convert error",
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
			scenario: "PDF engine write metadata error",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf":  "/file.pdf",
					"file2.pdf": "/file2.pdf",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				MergeMock: func(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
					return nil
				},
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
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
				ctx.SetValues(map[string][]string{
					"pdfa": {
						gotenberg.PdfA1b,
					},
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
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
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
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

func TestSplitHandler(t *testing.T) {
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
			scenario: "no split mode",
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
			scenario: "error from PDF engine (split)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return nil, errors.New("foo")
				},
			},
			expectError:            true,
			expectHttpError:        false,
			expectOutputPathsCount: 0,
		},
		{
			scenario: "error from PDF engine (convert)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
					"pdfua": {
						"true",
					},
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{inputPath}, nil
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
			scenario: "error from PDF engine (write metadata)",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{inputPath}, nil
				},
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
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
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
				})
				ctx.SetCancelled(true)
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{inputPath}, nil
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
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"splitMode": {
						gotenberg.SplitModeIntervals,
					},
					"splitSpan": {
						"1",
					},
					"pdfua": {
						"true",
					},
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				SplitMock: func(ctx context.Context, logger *zap.Logger, mode gotenberg.SplitMode, inputPath, outputDirPath string) ([]string, error) {
					return []string{"file_split_1.pdf", "file_split_2.pdf"}, nil
				},
				ConvertMock: func(ctx context.Context, logger *zap.Logger, formats gotenberg.PdfFormats, inputPath, outputPath string) error {
					return nil
				},
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
					return nil
				},
			},
			expectError:            false,
			expectHttpError:        false,
			expectOutputPathsCount: 2,
			expectOutputPaths:      []string{"/file/file_0.pdf", "/file/file_1.pdf"},
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
			c := echo.New().NewContext(nil, nil)
			c.Set("context", tc.ctx.Context)

			err := splitRoute(tc.engine).Handler(c)

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
			scenario: "cannot rename many files",
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
				ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
					return errors.New("cannot rename")
				}})
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
				ctx.SetPathRename(&gotenberg.PathRenameMock{RenameMock: func(oldpath, newpath string) error {
					return nil
				}})
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

func TestReadMetadataHandler(t *testing.T) {
	for _, tc := range []struct {
		scenario         string
		ctx              *api.ContextMock
		engine           gotenberg.PdfEngine
		expectError      bool
		expectedError    error
		expectHttpError  bool
		expectHttpStatus int
		expectedJson     string
	}{
		{
			scenario:         "missing at least one mandatory file",
			ctx:              &api.ContextMock{Context: new(api.Context)},
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "error from PDF engine",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				ReadMetadataMock: func(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
					return nil, errors.New("foo")
				},
			},
			expectError:     true,
			expectHttpError: false,
		},
		{
			scenario: "success",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				ReadMetadataMock: func(ctx context.Context, logger *zap.Logger, inputPath string) (map[string]interface{}, error) {
					return map[string]interface{}{
						"foo": "bar",
						"bar": "foo",
					}, nil
				},
			},
			expectError:     true,
			expectedError:   api.ErrNoOutputFile,
			expectHttpError: false,
			expectedJson:    `{"file.pdf":{"bar":"foo","foo":"bar"}}`,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			tc.ctx.SetLogger(zap.NewNop())
			req := httptest.NewRequest(http.MethodPost, "/forms/pdfengines/metadata/read", nil)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set("context", tc.ctx.Context)

			err := readMetadataRoute(tc.engine).Handler(c)

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

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}

			if err != nil && tc.expectHttpError && isHttpError {
				status, _ := httpErr.HttpError()
				if status != tc.expectHttpStatus {
					t.Errorf("expected %d as HTTP status code but got %d", tc.expectHttpStatus, status)
				}
			}

			if tc.expectedJson != "" && tc.expectedJson != strings.TrimSpace(rec.Body.String()) {
				t.Errorf("expected '%s' as HTTP response but got '%s'", tc.expectedJson, strings.TrimSpace(rec.Body.String()))
			}
		})
	}
}

func TestWriteMetadataHandler(t *testing.T) {
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
			scenario: "no metadata form field",
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
			scenario: "invalid metadata form field",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"foo",
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
			scenario: "no metadata",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"document.docx": "/document.docx",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{}",
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
			scenario: "error from PDF engine",
			ctx: func() *api.ContextMock {
				ctx := &api.ContextMock{Context: new(api.Context)}
				ctx.SetFiles(map[string]string{
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
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
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				ctx.SetCancelled(true)
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
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
					"file.pdf": "/file.pdf",
				})
				ctx.SetValues(map[string][]string{
					"metadata": {
						"{\"Creator\": \"foo\", \"Producer\": \"bar\" }",
					},
				})
				return ctx
			}(),
			engine: &gotenberg.PdfEngineMock{
				WriteMetadataMock: func(ctx context.Context, logger *zap.Logger, metadata map[string]interface{}, inputPath string) error {
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

			err := writeMetadataRoute(tc.engine).Handler(c)

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
