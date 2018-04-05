package file

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFile(t *testing.T) {
	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	// case 1: uses an empty reader.
	if _, err := NewFile(workingDir, new(bytes.Buffer)); err == nil {
		t.Error("File should not have been instantiated!")
	}

	// case 2: uses a reader from a wrong file type.
	path, _ := filepath.Abs("../../../_tests/configurations/gotenberg.yml")
	r, _ := os.Open(path)
	defer r.Close()
	if _, err := NewFile(workingDir, r); err == nil {
		t.Error("File should not have been instantiated!")
	}

	// case 3: uses a reader from a correct file type.
	path, _ = filepath.Abs("../../../_tests/file.pdf")
	r, _ = os.Open(path)
	defer r.Close()
	if _, err := NewFile(workingDir, r); err != nil {
		t.Error("File should have been instantiated!")
	}

	os.RemoveAll(workingDir)
}

func TestReworkFilePath(t *testing.T) {
	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	f := &File{
		Path: MakeFilePath(workingDir),
		Type: 999,
	}

	if _, err := reworkFilePath(workingDir, f); err == nil {
		t.Error("It should not have been able to found the file extension!")
	}

	os.RemoveAll(workingDir)
}

func TestFileTypeNotFoundError(t *testing.T) {
	err := &fileTypeNotFoundError{}
	if err.Error() != fileTypeNotFoundErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), fileTypeNotFoundErrorMessage)
	}
}

func TestFileExtNotFoundError(t *testing.T) {
	err := &fileExtNotFoundError{}
	if err.Error() != fileExtNotFoundErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), fileExtNotFoundErrorMessage)
	}
}
