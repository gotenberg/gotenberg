package gotenberg

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

// FileSystem provides utilities for managing temporary directories. It creates
// unique directory names based on UUIDs to ensure isolation of temporary files
// for different modules.
type FileSystem struct {
	workingDir string
}

// NewFileSystem initializes a new [FileSystem] instance with a unique working
// directory.
func NewFileSystem() *FileSystem {
	return &FileSystem{
		workingDir: uuid.NewString(),
	}
}

// WorkingDir returns the unique name of the working directory.
func (fs *FileSystem) WorkingDir() string {
	return fs.workingDir
}

// WorkingDirPath constructs and returns the full path to the working directory
// inside the system's temporary directory.
func (fs *FileSystem) WorkingDirPath() string {
	return fmt.Sprintf("%s/%s", os.TempDir(), fs.workingDir)
}

// NewDirPath generates a new unique path for a directory inside the working
// directory.
func (fs *FileSystem) NewDirPath() string {
	return fmt.Sprintf("%s/%s", fs.WorkingDirPath(), uuid.NewString())
}

// MkdirAll creates a new unique directory inside the working directory and
// returns its path. If the directory creation fails, an error is returned.
func (fs *FileSystem) MkdirAll() (string, error) {
	path := fs.NewDirPath()

	err := os.MkdirAll(path, 0o755)
	if err != nil {
		return "", fmt.Errorf("create directory %s: %w", path, err)
	}

	return path, nil
}
