package poppler

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestPoppler_Descriptor(t *testing.T) {
	descriptor := new(Poppler).Descriptor()

	if descriptor.ID != "poppler" {
		t.Errorf("expected ID 'poppler', got '%s'", descriptor.ID)
	}

	if _, ok := descriptor.New().(*Poppler); !ok {
		t.Error("expected New to return a *Poppler")
	}
}

func TestPoppler_Provision(t *testing.T) {
	for _, tc := range []struct {
		name      string
		envValue  string
		setEnv    bool
		expectErr bool
	}{
		{name: "env not set", setEnv: false, expectErr: true},
		{name: "env set", setEnv: true, envValue: "/usr/bin/pdftoppm", expectErr: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PDFTOPPM_BIN_PATH", "")
			if !tc.setEnv {
				os.Unsetenv("PDFTOPPM_BIN_PATH")
			} else {
				t.Setenv("PDFTOPPM_BIN_PATH", tc.envValue)
			}

			engine := new(Poppler)
			err := engine.Provision(nil)

			if tc.expectErr && err == nil {
				t.Fatal("expected an error, got none")
			}
			if !tc.expectErr {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if engine.binPath != tc.envValue {
					t.Errorf("expected binPath '%s', got '%s'", tc.envValue, engine.binPath)
				}
			}
		})
	}
}

func TestPoppler_Validate(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "pdftoppm")
	if err := os.WriteFile(existing, []byte("binary"), 0o755); err != nil {
		t.Fatalf("create fake binary: %v", err)
	}

	for _, tc := range []struct {
		name      string
		binPath   string
		expectErr bool
	}{
		{name: "existing path", binPath: existing, expectErr: false},
		{name: "missing path", binPath: filepath.Join(t.TempDir(), "nope"), expectErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := &Poppler{binPath: tc.binPath}
			err := engine.Validate()

			if tc.expectErr && err == nil {
				t.Fatal("expected an error, got none")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestPoppler_Routes(t *testing.T) {
	routes, err := new(Poppler).Routes()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Path != "/forms/pdfengines/convert/image" {
		t.Errorf("unexpected route path: %s", routes[0].Path)
	}
}

func TestFormatFlag(t *testing.T) {
	for _, tc := range []struct {
		format   string
		expected string
	}{
		{"png", "-png"},
		{"jpeg", "-jpeg"},
		{"tiff", "-tiff"},
		{"", "-png"},
		{"unknown", "-png"},
	} {
		if got := formatFlag(tc.format); got != tc.expected {
			t.Errorf("formatFlag(%q) = %q, want %q", tc.format, got, tc.expected)
		}
	}
}

func TestImageExtension(t *testing.T) {
	for _, tc := range []struct {
		format   string
		expected string
	}{
		{"png", "png"},
		{"jpeg", "jpeg"},
		{"tiff", "tiff"},
		{"", "png"},
		{"unknown", "png"},
	} {
		if got := imageExtension(tc.format); got != tc.expected {
			t.Errorf("imageExtension(%q) = %q, want %q", tc.format, got, tc.expected)
		}
	}
}

func TestImagePaths(t *testing.T) {
	dir := t.TempDir()
	// pdftoppm zero-pads page numbers, so the lexical sort matches page order.
	names := []string{"page-03.png", "page-01.png", "page-02.png"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("img"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := imagePaths(dir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	want := []string{
		filepath.Join(dir, "page-01.png"),
		filepath.Join(dir, "page-02.png"),
		filepath.Join(dir, "page-03.png"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("imagePaths() = %v, want %v", got, want)
	}
}

// Ensure Poppler satisfies the interfaces it is expected to.
var (
	_ gotenberg.Module      = (*Poppler)(nil)
	_ gotenberg.Validator   = (*Poppler)(nil)
	_ gotenberg.Provisioner = (*Poppler)(nil)
	_ gotenberg.Debuggable  = (*Poppler)(nil)
	_ api.Router            = (*Poppler)(nil)
)
