package gotenberg

import (
	"os"
	"strings"
	"testing"
)

func TestTmpPath(t *testing.T) {
	osTempDir := os.TempDir()
	tmpPath := TmpPath()

	if tmpPath != osTempDir {
		t.Errorf("expected path '%s' but got '%s'", osTempDir, tmpPath)
	}
}

func TestNewDirPath(t *testing.T) {
	newDirPath := NewDirPath()
	tmpPath := TmpPath()

	if !strings.HasPrefix(newDirPath, tmpPath) {
		t.Fatalf("expected path '%s' to start with '%s'", newDirPath, tmpPath)
	}

	newDirPaths := make([]string, 1000)
	for i := range newDirPaths {
		newDirPaths[i] = NewDirPath()
	}

	for i, newDirPath := range newDirPaths {
		for j, comparison := range newDirPaths {
			if i == j {
				continue
			}

			if newDirPath == comparison {
				t.Fatalf("expected path '%s' (index %d) to be unique, but found an identical path on index %d", newDirPath, i, j)
			}
		}
	}
}

func TestMkdirAll(t *testing.T) {
	path, err := MkdirAll()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	tmpPath := TmpPath()
	if !strings.HasPrefix(path, tmpPath) {
		t.Fatalf("expected path '%s' to start with '%s'", path, tmpPath)
	}

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("expected path '%s' to exist but got: %v", path, err)
	}
}
