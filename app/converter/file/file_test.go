package file

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFile(t *testing.T) {
	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	// case 1: uses a wrong file name.
	if _, err := NewFile(workingDir, new(bytes.Buffer), "file.yml"); err == nil {
		t.Error("File should not have been instantiated with an empty buffer")
	}

	// case 2: uses a reader from a correct file type.
	filePath, _ := filepath.Abs("../../../_tests/file.pdf")
	r, _ := os.Open(filePath)
	defer r.Close()
	if _, err := NewFile(workingDir, r, "file.pdf"); err != nil {
		t.Errorf("File should have been instantiated using a reader from '%s'", filePath)
	}

	os.RemoveAll(workingDir)
}
func TestFileTypeNotFoundError(t *testing.T) {
	fileName := "file.wp"
	err := &fileTypeNotFoundError{fileName: fileName}
	expected := fmt.Sprintf("File type was not found for '%s'", fileName)
	if err.Error() != expected {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), expected)
	}
}
