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

func TestPdfCpu_Merge(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		inputPaths  []string
		expectError bool
	}{
		{
			scenario: "invalid input path",
			inputPaths: []string{
				"foo",
			},
			expectError: true,
		},
		{
			scenario: "single file success",
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
			expectError: false,
		},
		{
			scenario: "many files success",
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(PdfCpu)
			err := engine.Provision(nil)
			if err != nil {
				t.Fatalf("expected error but got: %v", err)
			}

			fs := gotenberg.NewFileSystem()
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

			err = engine.Merge(nil, nil, tc.inputPaths, outputDir+"/foo.pdf")

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
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
