package ghostscript

import (
	"context"
	"errors"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
	"os"
	"reflect"
	"testing"
)

func TestGhostscript_Descriptor(t *testing.T) {
	descriptor := Ghostscript{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Ghostscript))

	if actual != expect {
		t.Errorf("expected '%s' but got '%s'", expect, actual)
	}
}

func TestGhostscript_Provision(t *testing.T) {
	mod := new(Ghostscript)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := mod.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestGhostscript_Validate(t *testing.T) {
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
			binPath: os.Getenv("GHOSTSCRIPT_BIN_PATH"),
		},
	} {
		mod := new(Ghostscript)
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

func TestGhostscript_Metrics(t *testing.T) {
	metrics, err := new(Ghostscript).Metrics()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if len(metrics) != 1 {
		t.Errorf("expected %d metrics, but got %d", 1, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != 0 {
		t.Errorf("expected %d Ghostscript instances, but got %f", 0, actual)
	}
}

func TestGhostscript_Merge(t *testing.T) {
	mod := new(Ghostscript)
	err := mod.Merge(context.TODO(), zap.NewNop(), []string{}, "")

	if !errors.Is(err, gotenberg.ErrPDFEngineMethodNotAvailable) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPDFEngineMethodNotAvailable, err)
	}
}

func TestGhostscript_Convert(t *testing.T) {
	for i, tc := range []struct {
		ctx        context.Context
		inputPath  string
		expectErr  bool
		format string
	}{
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA1b,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA1a,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA2b,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA2a,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA2u,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA3b,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA3a,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA3u,
			inputPath: "/tests/test/testdata/pdfengines/sample1.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA1b,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA1a,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA2b,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA2a,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA2u,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA3b,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA3a,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			format: gotenberg.FormatPDFA3u,
			inputPath: "/tests/test/testdata/pdfengines/sample2.pdf",
			expectErr: true,
		},
		{
			ctx:       nil,
			expectErr: true,
		},
		{
			ctx: context.TODO(),
			inputPath: "foo",
			expectErr: true,
		},
	} {
		func() {
			mod := new(Ghostscript)

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

			err = mod.Convert(tc.ctx, zap.NewNop(), tc.format, tc.inputPath, outputDir+"/foo.pdf")

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}

}
