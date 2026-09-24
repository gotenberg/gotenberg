package gotenberg

import (
	"os"
	"path/filepath"
	"strings"
)

// BuildVersionsDirPathEnvVar names the environment variable holding the
// absolute path to a directory of build-time version files. The Gotenberg image
// writes one file per module there, named by module ID and holding the version
// string of that module's backing binary, captured right after the binary is
// installed. The running process reads these files instead of executing the
// binaries, which keeps startup and the first request cheap.
const BuildVersionsDirPathEnvVar = "GOTENBERG_VERSIONS_DIR_PATH"

// BuildVersion returns the build-time version captured for the module with the
// given ID. The boolean is false when no version was captured, which is the
// case for local or non-Docker builds where the directory is absent. A module
// uses it to avoid spawning its backing binary just to report a version.
//
// It is defensive: an unset variable, a missing or unreadable file, or an empty
// value all yield ("", false), so the caller falls back to detecting the
// version at runtime. See [BuildVersionsDirPathEnvVar].
func BuildVersion(moduleID string) (string, bool) {
	dir := os.Getenv(BuildVersionsDirPathEnvVar)
	if dir == "" {
		return "", false
	}

	// Module IDs are fixed internal constants, never paths. Guard anyway so a
	// stray separator can't escape the versions directory.
	if moduleID != filepath.Base(moduleID) {
		return "", false
	}

	// The directory comes from a trusted operator-set environment variable,
	// mirroring how engines exec their env-configured binaries.
	b, err := os.ReadFile(filepath.Join(dir, moduleID)) //nolint:gosec
	if err != nil {
		return "", false
	}

	version := strings.TrimSpace(string(b))
	if version == "" {
		return "", false
	}

	return version, true
}
