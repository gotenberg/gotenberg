package exiftool

import (
	"errors"
	"slices"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestBuildExifToolWriteArgs_String(t *testing.T) {
	args, err := buildExifToolWriteArgs(map[string]any{"Title": "sample"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"-Title=sample"}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestBuildExifToolWriteArgs_StringSlice(t *testing.T) {
	args, err := buildExifToolWriteArgs(map[string]any{"Keywords": []string{"first", "second"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"-Keywords=first", "-Keywords=second"}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestBuildExifToolWriteArgs_AnySliceOfStrings(t *testing.T) {
	args, err := buildExifToolWriteArgs(map[string]any{"Keywords": []any{"a", "b"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"-Keywords=a", "-Keywords=b"}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestBuildExifToolWriteArgs_AnySliceMixedRejected(t *testing.T) {
	_, err := buildExifToolWriteArgs(map[string]any{"Keywords": []any{"a", 42}})
	if !errors.Is(err, gotenberg.ErrPdfEngineMetadataValueNotSupported) {
		t.Fatalf("expected ErrPdfEngineMetadataValueNotSupported, got %v", err)
	}
}

func TestBuildExifToolWriteArgs_Numbers(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   any
		want string
	}{
		{"int", 42, "-K=42"},
		{"int64", int64(42), "-K=42"},
		{"float32", float32(1.5), "-K=1.5"},
		{"float64", 1.7, "-K=1.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, err := buildExifToolWriteArgs(map[string]any{"K": tc.in})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(args) != 1 || args[0] != tc.want {
				t.Fatalf("args = %v, want [%q]", args, tc.want)
			}
		})
	}
}

func TestBuildExifToolWriteArgs_Bool(t *testing.T) {
	args, err := buildExifToolWriteArgs(map[string]any{"Marked": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"-Marked=true"}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestBuildExifToolWriteArgs_InvalidKey(t *testing.T) {
	for _, key := range []string{
		"",           // empty
		"-rm",        // leading dash — would be parsed as a flag
		"foo\nbar",   // newline
		"foo bar",    // space
		"foo=bar",    // contains equals
		"weird/char", // slash
	} {
		t.Run(key, func(t *testing.T) {
			_, err := buildExifToolWriteArgs(map[string]any{key: "value"})
			if !errors.Is(err, gotenberg.ErrPdfEngineMetadataValueNotSupported) {
				t.Fatalf("expected ErrPdfEngineMetadataValueNotSupported for key %q, got %v", key, err)
			}
		})
	}
}

func TestBuildExifToolWriteArgs_ControlCharValue(t *testing.T) {
	for _, val := range []string{
		"foo\nbar",
		"foo\rbar",
		"foo\x00bar",
	} {
		t.Run(val, func(t *testing.T) {
			_, err := buildExifToolWriteArgs(map[string]any{"Title": val})
			if !errors.Is(err, gotenberg.ErrPdfEngineMetadataValueNotSupported) {
				t.Fatalf("expected ErrPdfEngineMetadataValueNotSupported for value %q, got %v", val, err)
			}
		})
	}
}

func TestBuildExifToolWriteArgs_DangerousTagsStripped(t *testing.T) {
	// Dangerous tag keys are silently dropped; legitimate keys still pass.
	args, err := buildExifToolWriteArgs(map[string]any{
		"Author":          "legit",
		"FileName":        "stolen.pdf",
		"System:FileName": "stolen.pdf",
		"Directory":       "/tmp",
		"HardLink":        "/tmp/link",
		"SymLink":         "/tmp/link",
		"FilePermissions": "777",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(args, []string{"-Author=legit"}) {
		t.Fatalf("args = %v, want [-Author=legit]", args)
	}
}

func TestBuildExifToolWriteArgs_DangerousTagsCaseInsensitive(t *testing.T) {
	// Case variations are all dropped because exiftool is case-insensitive.
	args, err := buildExifToolWriteArgs(map[string]any{
		"filename":        "x",
		"FILENAME":        "x",
		"System:Filename": "x",
		"Title":           "keep",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(args, []string{"-Title=keep"}) {
		t.Fatalf("args = %v, want [-Title=keep]", args)
	}
}

func TestBuildExifToolWriteArgs_UnsupportedType(t *testing.T) {
	_, err := buildExifToolWriteArgs(map[string]any{"K": map[string]any{"nested": "x"}})
	if !errors.Is(err, gotenberg.ErrPdfEngineMetadataValueNotSupported) {
		t.Fatalf("expected ErrPdfEngineMetadataValueNotSupported, got %v", err)
	}
}

func TestBuildExifToolWriteArgs_Empty(t *testing.T) {
	args, err := buildExifToolWriteArgs(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 0 {
		t.Fatalf("args = %v, want empty", args)
	}
}

func TestIsDangerousTag(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want bool
	}{
		{"FileName", true},
		{"filename", true},
		{"System:FileName", true},
		{"XMP:FileName", true},
		{"Directory", true},
		{"HardLink", true},
		{"SymLink", true},
		{"FilePermissions", true},
		{"Title", false},
		{"Author", false},
		{"FileNameExtra", false}, // Suffix must not match.
		{"", false},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := isDangerousTag(tc.key); got != tc.want {
				t.Fatalf("isDangerousTag(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestSafeKeyPattern(t *testing.T) {
	// Rejects leading dash to prevent argv-level flag injection.
	if safeKeyPattern.MatchString("-injected") {
		t.Fatalf("leading-dash key must be rejected")
	}
	// Accepts common legitimate forms.
	for _, k := range []string{"Title", "System:Title", "XMP-pdf:Title", "My_Tag.1"} {
		if !safeKeyPattern.MatchString(k) {
			t.Fatalf("key %q must be accepted", k)
		}
	}
	// Rejects control characters.
	for _, k := range []string{"a\nb", "a\rb", "a\x00b", "a b"} {
		if safeKeyPattern.MatchString(k) {
			t.Fatalf("control-char key %q must be rejected", k)
		}
	}
}
