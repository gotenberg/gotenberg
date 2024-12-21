package pdfcpu

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestPdfCpu_Descriptor(t *testing.T) {
	descriptor := new(PdfCpu).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(PdfCpu))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestPdfCpu_Provision(t *testing.T) {
	engine := new(PdfCpu)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := engine.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestPdfCpu_Validate(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		binPath     string
		expectError bool
	}{
		{
			scenario:    "empty bin path",
			binPath:     "",
			expectError: true,
		},
		{
			scenario:    "bin path does not exist",
			binPath:     "/foo",
			expectError: true,
		},
		{
			scenario:    "validate success",
			binPath:     os.Getenv("PDFTK_BIN_PATH"),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(PdfCpu)
			engine.binPath = tc.binPath
			err := engine.Validate()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestPdfCpu_Merge(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctx         context.Context
		inputPaths  []string
		expectError bool
	}{
		{
			scenario:    "invalid context",
			ctx:         nil,
			expectError: true,
		},
		{
			scenario: "invalid input path",
			ctx:      context.TODO(),
			inputPaths: []string{
				"foo",
			},
			expectError: true,
		},
		{
			scenario: "single file success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
			expectError: false,
		},
		{
			scenario: "many files success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(PdfCpu)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
			outputDir, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			defer func() {
				err = os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			err = engine.Merge(tc.ctx, zap.NewNop(), tc.inputPaths, outputDir+"/foo.pdf")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestPdfCpu_Split(t *testing.T) {
	for _, tc := range []struct {
		scenario               string
		ctx                    context.Context
		mode                   gotenberg.SplitMode
		inputPath              string
		expectError            bool
		expectedError          error
		expectOutputPathsCount int
	}{
		{
			scenario:               "ErrPdfSplitModeNotSupported",
			expectError:            true,
			expectedError:          gotenberg.ErrPdfSplitModeNotSupported,
			expectOutputPathsCount: 0,
		},
		{
			scenario:               "invalid context",
			ctx:                    nil,
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			expectError:            true,
			expectOutputPathsCount: 0,
		},
		{
			scenario:               "invalid input path",
			ctx:                    context.TODO(),
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			inputPath:              "",
			expectError:            true,
			expectOutputPathsCount: 0,
		},
		{
			scenario:               "success (intervals)",
			ctx:                    context.TODO(),
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModeIntervals, Span: "1"},
			inputPath:              "/tests/test/testdata/pdfengines/sample1.pdf",
			expectError:            false,
			expectOutputPathsCount: 3,
		},
		{
			scenario:               "success (pages)",
			ctx:                    context.TODO(),
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1"},
			inputPath:              "/tests/test/testdata/pdfengines/sample1.pdf",
			expectError:            false,
			expectOutputPathsCount: 1,
		},
		{
			scenario:               "success (pages & unify)",
			ctx:                    context.TODO(),
			mode:                   gotenberg.SplitMode{Mode: gotenberg.SplitModePages, Span: "1-2", Unify: true},
			inputPath:              "/tests/test/testdata/pdfengines/sample1.pdf",
			expectError:            false,
			expectOutputPathsCount: 1,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(PdfCpu)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
			outputDir, err := fs.MkdirAll()
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			defer func() {
				err = os.RemoveAll(fs.WorkingDirPath())
				if err != nil {
					t.Fatalf("expected no error while cleaning up but got: %v", err)
				}
			}()

			outputPaths, err := engine.Split(tc.ctx, zap.NewNop(), tc.mode, tc.inputPath, outputDir)

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
			}

			if tc.expectOutputPathsCount != len(outputPaths) {
				t.Errorf("expected %d output paths but got %d", tc.expectOutputPathsCount, len(outputPaths))
			}
		})
	}
}

func TestPdfCpu_Convert(t *testing.T) {
	mod := new(PdfCpu)
	err := mod.Convert(context.TODO(), zap.NewNop(), gotenberg.PdfFormats{}, "", "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestLibreOfficePdfEngine_ReadMetadata(t *testing.T) {
	engine := new(PdfCpu)
	_, err := engine.ReadMetadata(context.Background(), zap.NewNop(), "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}

func TestLibreOfficePdfEngine_WriteMetadata(t *testing.T) {
	engine := new(PdfCpu)
	err := engine.WriteMetadata(context.Background(), zap.NewNop(), nil, "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}
