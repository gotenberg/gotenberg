package ghostscript

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
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
		t.Fatalf("expected no error but got: %v", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("expected %d metrics, but got %d", 1, len(metrics))
	}

	actual := metrics[0].Read()
	if actual != 0 {
		t.Errorf("expected %d Ghostscript instances, but got %f", 0, actual)
	}
}

func TestGhostscript_Merge(t *testing.T) {
	mod := new(Ghostscript)
	err := mod.Convert(context.TODO(), zap.NewNop(), "", "", "")

	if !errors.Is(err, gotenberg.ErrPDFEngineMethodNotAvailable) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPDFEngineMethodNotAvailable, err)
	}
}

func TestGhostscript_Convert(t *testing.T) {
	mod := new(Ghostscript)
	err := mod.Convert(context.TODO(), zap.NewNop(), "", "", "")

	if !errors.Is(err, gotenberg.ErrPDFEngineMethodNotAvailable) {
		t.Errorf("expected error %v, but got: %v", gotenberg.ErrPDFEngineMethodNotAvailable, err)
	}
}
