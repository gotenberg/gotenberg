package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

func TestApiDebug_Versions(t *testing.T) {
	t.Run("prefers the build-time versions without executing the binaries", func(t *testing.T) {
		dir := t.TempDir()
		for name, content := range map[string]string{
			"libreoffice-api": "LibreOffice 26.8.1.1\n",
			"lok":             "lok v0.0.3\n",
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
				t.Fatalf("write version file: %v", err)
			}
		}
		t.Setenv(gotenberg.BuildVersionsDirPathEnvVar, dir)

		// Bogus paths would error if executed, so correct results prove the
		// build-time files are used.
		mod := &Api{args: libreOfficeArguments{binPath: "/nonexistent/soffice", lokBinPath: "/nonexistent/lok"}}
		debug := mod.Debug()
		if got := debug["version"]; got != "LibreOffice 26.8.1.1" {
			t.Errorf("Debug()[version] = %v, want the build-time value", got)
		}
		if got := debug["lok_version"]; got != "lok v0.0.3" {
			t.Errorf("Debug()[lok_version] = %v, want the build-time value", got)
		}
	})

	t.Run("falls back to executing lok when no build-time version", func(t *testing.T) {
		t.Setenv(gotenberg.BuildVersionsDirPathEnvVar, "")

		mod := &Api{args: libreOfficeArguments{binPath: "/nonexistent/soffice", lokBinPath: "/nonexistent/lok"}}
		if got := mod.Debug()["lok_version"]; got == "" || got == "lok v0.0.3" {
			t.Errorf("Debug()[lok_version] = %v, expected the exec error", got)
		}
	})
}
