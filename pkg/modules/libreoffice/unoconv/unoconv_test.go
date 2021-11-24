package unoconv

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func TestUnoconv_Descriptor(t *testing.T) {
	descriptor := Unoconv{}.Descriptor()

	actual := reflect.TypeOf(descriptor.New())
	expect := reflect.TypeOf(new(Unoconv))

	if actual != expect {
		t.Errorf("expected '%'s' but got '%s'", expect, actual)
	}
}

func TestUnoconv_Provision(t *testing.T) {
	mod := new(Unoconv)
	ctx := gotenberg.NewContext(gotenberg.ParsedFlags{}, nil)

	err := mod.Provision(ctx)
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestUnoconv_Validate(t *testing.T) {
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
			binPath: os.Getenv("UNOCONV_BIN_PATH"),
		},
	} {
		mod := new(Unoconv)
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

func TestChromium_Metrics(t *testing.T) {
	metrics, err := new(Unoconv).Metrics()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if len(metrics) != 1 {
		t.Errorf("expected %d metrics, but got %d", 1, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != 0 {
		t.Errorf("expected %d unoconv instances, but got %f", 0, actual)
	}
}

func TestUnoconv_Unoconv(t *testing.T) {
	mod := new(Unoconv)

	_, err := mod.Unoconv()
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}

func TestUnoconv_PDF(t *testing.T) {
	for i, tc := range []struct {
		ctx       context.Context
		logger    *zap.Logger
		inputPath string
		options   Options
		expectErr bool
	}{
		{
			logger:    zap.NewNop(),
			expectErr: true,
		},
		{
			ctx:       context.Background(),
			logger:    zap.NewExample(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				Landscape:  true,
				PageRanges: "1-2",
				PDFArchive: true,
			},
		},
		{
			ctx:       context.Background(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			options: Options{
				PageRanges: "foo",
			},
			expectErr: true,
		},
		{
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.TODO())
				defer cancel()

				return ctx
			}(),
			logger:    zap.NewNop(),
			inputPath: "/tests/test/testdata/libreoffice/sample1.docx",
			expectErr: true,
		},
	} {
		func() {
			mod := new(Unoconv)

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

			err = mod.PDF(tc.ctx, tc.logger, tc.inputPath, outputDir+"/foo.pdf", tc.options)

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}
}

func TestUnoconv_Extensions(t *testing.T) {
	mod := new(Unoconv)
	extensions := mod.Extensions()

	actual := len(extensions)
	expect := 76

	if actual != expect {
		t.Errorf("expected %d extensions but got %d", expect, actual)
	}
}
