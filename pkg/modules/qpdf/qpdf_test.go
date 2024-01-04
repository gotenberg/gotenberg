package qpdf

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestQPdf_Descriptor(t *testing.T) {
	descriptor := new(QPdf).Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(QPdf))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestQPdf_Provision(t *testing.T) {
	engine := new(QPdf)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := engine.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestQPdf_Validate(t *testing.T) {
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
			binPath:     os.Getenv("QPDF_BIN_PATH"),
			expectError: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(QPdf)
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

func TestQPdf_Merge(t *testing.T) {
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
		},
		{
			scenario: "many files success",
			ctx:      context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			engine := new(QPdf)
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

func TestQPdf_Convert(t *testing.T) {
	engine := new(QPdf)
	err := engine.Convert(context.TODO(), zap.NewNop(), gotenberg.PdfFormats{}, "", "")

	if !errors.Is(err, gotenberg.ErrPdfEngineMethodNotSupported) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPdfEngineMethodNotSupported, err)
	}
}
