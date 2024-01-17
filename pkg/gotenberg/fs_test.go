package gotenberg

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestFileSystem_WorkingDir(t *testing.T) {
	fs := NewFileSystem()
	dirName := fs.WorkingDir()

	if dirName == "" {
		t.Error("expected directory name but got empty string")
	}
}

func TestFileSystem_WorkingDirPath(t *testing.T) {
	fs := NewFileSystem()
	expectedPath := fmt.Sprintf("%s/%s", os.TempDir(), fs.WorkingDir())

	if fs.WorkingDirPath() != expectedPath {
		t.Errorf("expected path '%s' but got '%s'", expectedPath, fs.WorkingDirPath())
	}
}

func TestFileSystem_NewDirPath(t *testing.T) {
	fs := NewFileSystem()
	newDir := fs.NewDirPath()
	expectedPrefix := fs.WorkingDirPath()

	if !strings.HasPrefix(newDir, expectedPrefix) {
		t.Errorf("expected new directory to start with '%s' but got '%s'", expectedPrefix, newDir)
	}
}

func TestFileSystem_MkdirAll(t *testing.T) {
	fs := NewFileSystem()

	newPath, err := fs.MkdirAll()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	_, err = os.Stat(newPath)
	if os.IsNotExist(err) {
		t.Errorf("expected directory '%s' to exist but it doesn't", newPath)
	}

	err = os.RemoveAll(fs.WorkingDirPath())
	if err != nil {
		t.Fatalf("expected no error while cleaning up but got: %v", err)
	}
}
