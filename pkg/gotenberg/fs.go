package gotenberg

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

// MkdirAll defines the method signature for create a directory. Implement this
// interface if you don't want to rely on [os.MkdirAll], notably for testing
// purpose.
type MkdirAll interface {
	// MkdirAll uses the same signature as [os.MkdirAll].
	MkdirAll(path string, perm os.FileMode) error
}

// OsMkdirAll implements the [MkdirAll] interface with [os.MkdirAll].
type OsMkdirAll struct{}

// MkdirAll is a wrapper around [os.MkdirAll].
func (o *OsMkdirAll) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

// PathRename defines the method signature for renaming files. Implement this
// interface if you don't want to rely on [os.Rename], notably for testing
// purposes.
type PathRename interface {
	// Rename uses the same signature as [os.Rename].
	Rename(oldpath, newpath string) error
}

// OsPathRename implements the [PathRename] interface with [os.Rename].
type OsPathRename struct{}

// Rename is a wrapper around [os.Rename].
func (o *OsPathRename) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// FileSystem provides utilities for managing temporary directories. It creates
// unique directory names based on UUIDs to ensure isolation of temporary files
// for different modules.
type FileSystem struct {
	workingDir string
	mkdirAll   MkdirAll
}

// NewFileSystem initializes a new [FileSystem] instance with a unique working
// directory.
func NewFileSystem(mkdirAll MkdirAll) *FileSystem {
	return &FileSystem{
		workingDir: uuid.NewString(),
		mkdirAll:   mkdirAll,
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

	err := fs.mkdirAll.MkdirAll(path, 0o755)
	if err != nil {
		return "", fmt.Errorf("create directory %s: %w", path, err)
	}

	return path, nil
}

// Interface guards.
var (
	_ MkdirAll   = (*OsMkdirAll)(nil)
	_ PathRename = (*OsPathRename)(nil)
)
