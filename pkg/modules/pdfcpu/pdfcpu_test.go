package pdfcpu

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func TestPDFcpu_Descriptor(t *testing.T) {
	descriptor := PDFcpu{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(PDFcpu))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestPDFcpu_Provision(t *testing.T) {
	mod := new(PDFcpu)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := mod.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestPDFcpu_Merge(t *testing.T) {
	for i, tc := range []struct {
		inputPaths []string
		expectErr  bool
	}{
		{
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
		},
		{
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
		},
		{
			inputPaths: []string{
				"foo",
			},
			expectErr: true,
		},
	} {
		func() {
			mod := new(PDFcpu)

			err := mod.Provision(nil)
			if err != nil {
				t.Fatalf("test %d: expected error but got: %v", i, err)
			}

			outputDir, err := gotenberg.MkdirAll()
			if err != nil {
				t.Fatalf("test %d: expected error but got: %v", i, err)
			}

			defer func() {
				err := os.RemoveAll(outputDir)
				if err != nil {
					t.Fatalf("test %d: expected no error but got: %v", i, err)
				}
			}()

			err = mod.Merge(nil, nil, tc.inputPaths, outputDir+"/foo.pdf")

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}
}

func TestPDFcpu_Convert(t *testing.T) {
	mod := new(PDFcpu)
	err := mod.Convert(context.TODO(), zap.NewNop(), "", "", "")

	if !errors.Is(err, gotenberg.ErrPDFEngineMethodNotAvailable) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPDFEngineMethodNotAvailable, err)
	}
}
