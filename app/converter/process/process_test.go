package process

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gulien/gotenberg/app/config"
	gfile "github.com/gulien/gotenberg/app/converter/file"
)

func TestLoad(t *testing.T) {
	path, _ := filepath.Abs("../../../_tests/configurations/gotenberg.yml")
	c, _ := config.NewAppConfig(path)
	Load(c.CommandsConfig)

	if c.CommandsConfig != commandsConfig {
		t.Error("Commands configuration should have loaded correctly!")
	}
}

func TestUnconv(t *testing.T) {
	path, _ := filepath.Abs("../../../_tests/configurations/gotenberg.yml")
	c, _ := config.NewAppConfig(path)
	Load(c.CommandsConfig)

	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	// case 1: uses an HTML file type.
	path, _ = filepath.Abs("../../../_tests/file.html")
	r, _ := os.Open(path)
	defer r.Close()
	f, _ := gfile.NewFile(workingDir, r)
	if _, err := Unconv(workingDir, f); err != nil {
		t.Error("HTML conversion to PDF should have worked!")
	}

	// case 2: uses an Office file type.
	path, _ = filepath.Abs("../../../_tests/file.docx")
	r, _ = os.Open(path)
	defer r.Close()
	f, _ = gfile.NewFile(workingDir, r)
	if _, err := Unconv(workingDir, f); err != nil {
		t.Error("Office conversion to PDF should have worked!")
	}

	// case 3: uses a PDF file type.
	path, _ = filepath.Abs("../../../_tests/file.pdf")
	r, _ = os.Open(path)
	defer r.Close()
	f, _ = gfile.NewFile(workingDir, r)
	if _, err := Unconv(workingDir, f); err == nil {
		t.Error("PDF conversion to PDF should not have worked!")
	}

	// case 4: uses a command with an unsuitable timeout.
	path, _ = filepath.Abs("../../../_tests/configurations/timeout-gotenberg.yml")
	c, _ = config.NewAppConfig(path)
	Load(c.CommandsConfig)
	path, _ = filepath.Abs("../../../_tests/file.docx")
	r, _ = os.Open(path)
	defer r.Close()
	f, _ = gfile.NewFile(workingDir, r)
	if _, err := Unconv(workingDir, f); err == nil {
		t.Error("Office conversion to PDF should have reached timeout!")
	}

	os.RemoveAll(workingDir)
}

func TestMerge(t *testing.T) {
	path, _ := filepath.Abs("../../../_tests/configurations/gotenberg.yml")
	c, _ := config.NewAppConfig(path)
	Load(c.CommandsConfig)

	workingDir := "test"
	os.Mkdir(workingDir, 0666)

	var filesPaths []string
	path, _ = filepath.Abs("../../../_tests/file.pdf")
	filesPaths = append(filesPaths, path)
	filesPaths = append(filesPaths, path)

	// case 1: simple merge.
	if _, err := Merge(workingDir, filesPaths); err != nil {
		t.Error("Merge should have worked!")
	}

	// case 2: uses a command with an unsuitable timeout.
	path, _ = filepath.Abs("../../../_tests/configurations/timeout-gotenberg.yml")
	c, _ = config.NewAppConfig(path)
	Load(c.CommandsConfig)
	if _, err := Merge(workingDir, filesPaths); err == nil {
		t.Error("Merge should have reached timeout!")
	}

	os.RemoveAll(workingDir)
}

func TestRun(t *testing.T) {
	// case 1: uses a simple command.
	if err := run("echo Hello world", 30); err != nil {
		t.Error("Command should have worked!")
	}

	// case 2: uses a simple command but with an unsuitable timeout.
	if err := run("echo Hello world", 0); err == nil {
		t.Error("Command should not have worked!")
	}

	// case 3: uses a broken command.
	if err := run("helloworld", 30); err == nil {
		t.Error("Command should not have worked!")
	}
}

func TestImpossibleConversionError(t *testing.T) {
	err := &impossibleConversionError{}
	if err.Error() != impossibleConversionErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), impossibleConversionErrorMessage)
	}
}

func TestCommandTimeoutError(t *testing.T) {
	err := &commandTimeoutError{}
	if err.Error() != commandTimeoutErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), commandTimeoutErrorMessage)
	}
}
