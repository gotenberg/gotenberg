package qpdf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestQPdfDetectVersion(t *testing.T) {
	t.Run("prefers the build-time version without executing qpdf", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "qpdf"), []byte("qpdf version 11.9.0\n"), 0o600); err != nil {
			t.Fatalf("write version file: %v", err)
		}
		t.Setenv(gotenberg.BuildVersionsDirPathEnvVar, dir)

		// A bogus binPath would error if executed, so a correct result proves
		// the build-time file is used instead of running qpdf.
		engine := &QPdf{binPath: "/nonexistent/qpdf"}
		if got := engine.Debug()["version"]; got != "qpdf version 11.9.0" {
			t.Errorf("Debug()[version] = %v, want the build-time value", got)
		}
	})

	t.Run("falls back to executing qpdf when no build-time version", func(t *testing.T) {
		t.Setenv(gotenberg.BuildVersionsDirPathEnvVar, "")

		// With no build-time file and a bogus binPath, the exec fallback runs
		// and records its error rather than a build-time value.
		engine := &QPdf{binPath: "/nonexistent/qpdf"}
		if got := engine.Debug()["version"]; got == "qpdf version 11.9.0" {
			t.Errorf("Debug()[version] = %v, expected the exec fallback", got)
		}
	})

	t.Run("exposes the version as a span attribute", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "qpdf"), []byte("qpdf version 12.2.0"), 0o600); err != nil {
			t.Fatalf("write version file: %v", err)
		}
		t.Setenv(gotenberg.BuildVersionsDirPathEnvVar, dir)

		// spanAttrs is what gets handed to trace.WithAttributes, so its output
		// is exactly what a span carries.
		engine := &QPdf{binPath: "/nonexistent/qpdf"}
		var got string
		for _, attr := range engine.spanAttrs() {
			if attr.Key == "gotenberg.qpdf.version" {
				got = attr.Value.AsString()
			}
		}
		if got != "qpdf version 12.2.0" {
			t.Errorf("span attribute gotenberg.qpdf.version = %q, want the build-time value", got)
		}
	})
}
