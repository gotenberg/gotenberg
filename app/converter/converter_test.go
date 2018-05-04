package converter

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/thecodingmachine/gotenberg/app/config"
)

func makeRequest(filesPaths ...string) *http.Request {
	r, w := io.Pipe()
	mpw := multipart.NewWriter(w)

	go func() {
		var part io.Writer
		defer w.Close()

		if len(filesPaths) == 0 {
			part, _ = mpw.CreateFormField("foo")
			part.Write([]byte("bar"))
		} else {
			for _, filePath := range filesPaths {
				file, _ := os.Open(filePath)
				defer file.Close()

				fileInfo, _ := file.Stat()
				part, _ = mpw.CreateFormFile("files", fileInfo.Name())
				io.Copy(part, file)
			}
		}

		mpw.Close()
	}()

	req := httptest.NewRequest(http.MethodPost, "/", r)
	req.Header.Set("Content-Type", mpw.FormDataContentType())
	return req
}

func load(configurationFilePath string) {
	config.Reset()
	path, _ := filepath.Abs(configurationFilePath)
	config.ParseFile(path)
}

func TestNoFileToConvertError(t *testing.T) {
	err := &NoFileToConvertError{}
	if err.Error() != noFileToConvertErrorMessage {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), noFileToConvertErrorMessage)
	}
}

func TestNewConverter(t *testing.T) {
	var (
		path  string
		oPath string
	)

	load("../../_tests/configurations/gotenberg.yml")

	// case 1: uses a request with a single file.
	path, _ = filepath.Abs("../../_tests/file.docx")
	if _, err := NewConverter(makeRequest(path)); err != nil {
		t.Errorf("Converter should have been instantiated with '%s'", path)
	}

	// case 2: uses a request with wrong file type.
	path, _ = filepath.Abs("../../_tests/configurations/gotenberg.yml")
	if _, err := NewConverter(makeRequest(path)); err == nil {
		t.Errorf("Converter should not have been instantiated with '%s'", path)
	}

	// case 3: uses a request with two files.
	path, _ = filepath.Abs("../../_tests/file.pdf")
	oPath, _ = filepath.Abs("../../_tests/file.docx")
	if _, err := NewConverter(makeRequest(path, oPath)); err != nil {
		t.Errorf("Converter should have been instantiated with '%s' and '%s'", path, oPath)
	}

	// case 4: uses a request with one Office file type and one wrong file type.
	path, _ = filepath.Abs("../../_tests/configurations/gotenberg.yml")
	oPath, _ = filepath.Abs("../../_tests/file.docx")
	if _, err := NewConverter(makeRequest(path, oPath)); err == nil {
		t.Errorf("Converter should not have been instantiated with '%s' and '%s'", path, oPath)
	}

	// case 5: uses a request with no file.
	if _, err := NewConverter(makeRequest()); err == nil {
		t.Error("Converter should not have been instantiated with no file")
	}
}

func TestConvert(t *testing.T) {
	var (
		path  string
		oPath string
		c     *Converter
	)

	load("../../_tests/configurations/gotenberg.yml")

	// case 1: uses a request with a single file.
	path, _ = filepath.Abs("../../_tests/file.docx")
	c, _ = NewConverter(makeRequest(path))
	if _, err := c.Convert(); err != nil {
		t.Errorf("Converter should have been able to convert '%s' to PDF", path)
	}

	// case 2: uses a request with two files.
	path, _ = filepath.Abs("../../_tests/file.pdf")
	oPath, _ = filepath.Abs("../../_tests/file.docx")
	c, _ = NewConverter(makeRequest(path, oPath))
	if _, err := c.Convert(); err != nil {
		t.Errorf("Converter should have been able to convert '%s' and '%s' to PDF", path, oPath)
	}

	load("../../_tests/configurations/timeout-gotenberg.yml")

	// case 3: uses a request with a single file and a configuration with an unsuitable timeout for the conversion commands.
	path, _ = filepath.Abs("../../_tests/file.docx")
	c, _ = NewConverter(makeRequest(path))
	if _, err := c.Convert(); err == nil {
		t.Errorf("Converter should not have been able to convert '%s' to PDF", path)
	}

	load("../../_tests/configurations/merge-timeout-gotenberg.yml")

	// case 4: uses a request with two files and a configuration with an unsuitable timeout for the merge command.
	path, _ = filepath.Abs("../../_tests/file.pdf")
	oPath, _ = filepath.Abs("../../_tests/file.docx")
	c, _ = NewConverter(makeRequest(path, oPath))
	if _, err := c.Convert(); err == nil {
		t.Errorf("Converter should not have been able to merge '%s' and '%s' into PDF", path, oPath)
	}
}

func TestClear(t *testing.T) {
	load("../../_tests/configurations/gotenberg.yml")

	path, _ := filepath.Abs("../../_tests/file.docx")
	c, _ := NewConverter(makeRequest(path))
	if err := c.Clear(); err != nil {
		t.Error("Converter should have been able to clear itself")
	}
}
