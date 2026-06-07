package chromium

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestChromiumDetectVersion(t *testing.T) {
	t.Run("prefers the build-time version without executing Chromium", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "chromium"), []byte("Chromium 146.0.7680.80\n"), 0o600); err != nil {
			t.Fatalf("write version file: %v", err)
		}
		t.Setenv(gotenberg.BuildVersionsDirPathEnvVar, dir)

		// A bogus binPath would error if executed, so a correct result proves
		// the build-time file is used instead of running Chromium.
		mod := &Chromium{args: browserArguments{binPath: "/nonexistent/chromium"}}
		if got := mod.Debug()["version"]; got != "Chromium 146.0.7680.80" {
			t.Errorf("Debug()[version] = %v, want the build-time value", got)
		}
	})

	t.Run("falls back to executing Chromium when no build-time version", func(t *testing.T) {
		t.Setenv(gotenberg.BuildVersionsDirPathEnvVar, "")

		// With no build-time file and a bogus binPath, the exec fallback runs
		// and records its error rather than a build-time value.
		mod := &Chromium{args: browserArguments{binPath: "/nonexistent/chromium"}}
		if got := mod.Debug()["version"]; got == "Chromium 146.0.7680.80" {
			t.Errorf("Debug()[version] = %v, expected the exec fallback", got)
		}
	})
}
