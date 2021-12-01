package qpdf

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func TestQPDF_Descriptor(t *testing.T) {
	descriptor := QPDF{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(QPDF))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestQPDF_Provision(t *testing.T) {
	mod := new(QPDF)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := mod.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestQPDF_Validate(t *testing.T) {
	for i, tc := range []struct {
		binPath   string
		expectErr bool
	}{
		{
			expectErr: true,
		},
		{
			binPath:   "/foo",
			expectErr: true,
		},
		{
			binPath: os.Getenv("QPDF_BIN_PATH"),
		},
	} {
		mod := new(QPDF)
		mod.binPath = tc.binPath
		err := mod.Validate()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestQPDF_Metrics(t *testing.T) {
	metrics, err := new(QPDF).Metrics()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if len(metrics) != 1 {
		t.Errorf("expected %d metrics, but got %d", 1, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != 0 {
		t.Errorf("expected %d QPDF instances, but got %f", 0, actual)
	}
}

func TestQPDF_Merge(t *testing.T) {
	for i, tc := range []struct {
		ctx        context.Context
		inputPaths []string
		expectErr  bool
	}{
		{
			ctx: context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
			},
		},
		{
			ctx: context.TODO(),
			inputPaths: []string{
				"/tests/test/testdata/pdfengines/sample1.pdf",
				"/tests/test/testdata/pdfengines/sample2.pdf",
			},
		},
		{
			ctx:       nil,
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			inputPaths: []string{
				"foo",
			},
			expectErr: true,
		},
	} {
		func() {
			mod := new(QPDF)

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

			err = mod.Merge(tc.ctx, zap.NewNop(), tc.inputPaths, outputDir+"/foo.pdf")

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}
}

func TestQPDF_Convert(t *testing.T) {
	mod := new(QPDF)
	err := mod.Convert(context.TODO(), zap.NewNop(), "", "", "")

	if !errors.Is(err, gotenberg.ErrPDFEngineMethodNotAvailable) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPDFEngineMethodNotAvailable, err)
	}
}
