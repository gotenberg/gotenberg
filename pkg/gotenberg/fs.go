package gotenberg

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

// TmpPath returns the default directory to use for temporary files and
// directories. Most if not all files and directories created by the
// application and its dependencies must be based on this default directory.
func TmpPath() string {
	return os.TempDir()
}

// NewDirPath returns a random absolute path based on the temporary path.
func NewDirPath() string {
	return fmt.Sprintf("%s/%s", TmpPath(), uuid.New())
}

// MkdirAll creates a random directory based on the temporary path and
// returns its absolute path.
func MkdirAll() (string, error) {
	path := NewDirPath()

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return "", fmt.Errorf("create directory %s: %w", path, err)
	}

	return path, nil
}
