package file

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/thecodingmachine/gotenberg/app/config"
)

func load(configurationFilePath string) {
	config.Reset()
	path, _ := filepath.Abs(configurationFilePath)
	config.ParseFile(path)
}

func TestNewFile(t *testing.T) {
	load("../../../_tests/configurations/gotenberg.yml")

	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	// case 1: uses a wrong file name.
	if _, err := NewFile(workingDir, new(bytes.Buffer), "file.yml"); err == nil {
		t.Error("File should not have been instantiated with an empty buffer")
	}

	// case 2: uses a file name.
	filePath, _ := filepath.Abs("../../../_tests/file.pdf")
	r, _ := os.Open(filePath)
	defer r.Close()
	if _, err := NewFile(workingDir, r, "file.pdf"); err != nil {
		t.Errorf("File should have been instantiated using a reader of '%s'", filePath)
	}

	os.RemoveAll(workingDir)
}
