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
	"github.com/thecodingmachine/gotenberg/app/converter/process"
)

func makeRequest(filesPaths ...string) *http.Request {
	r, w := io.Pipe()
	mpw := multipart.NewWriter(w)

	go func() {
		var part io.Writer
		defer w.Close()

		for _, filePath := range filesPaths {
			file, _ := os.Open(filePath)
			defer file.Close()

			fileInfo, _ := file.Stat()
			part, _ = mpw.CreateFormFile("files", fileInfo.Name())
			io.Copy(part, file)
		}

		mpw.Close()
	}()

	req := httptest.NewRequest(http.MethodPost, "/", r)
	req.Header.Set("Content-Type", mpw.FormDataContentType())
	return req
}

func TestNewConverter(t *testing.T) {
	// case 1: uses a request with a single file.
	path, _ := filepath.Abs("../../_tests/file.docx")
	if _, err := NewConverter(makeRequest(path)); err != nil {
		t.Error("Converter should have been instantiated!")
	}

	// case 2: uses a request with wrong file type.
	path, _ = filepath.Abs("../../_tests/configurations/gotenberg.yml")
	if _, err := NewConverter(makeRequest(path)); err == nil {
		t.Error("Converter should not have been instantiated!")
	}

	// case 3: uses a request with two files.
	oPath, _ := filepath.Abs("../../_tests/file.docx")
	path, _ = filepath.Abs("../../_tests/file.pdf")
	if _, err := NewConverter(makeRequest(oPath, path)); err != nil {
		t.Error("Converter should have been instantiated!")
	}

	// case 4: uses a request with one Office file and one wrong file type.
	oPath, _ = filepath.Abs("../../_tests/file.docx")
	path, _ = filepath.Abs("../../_tests/configurations/gotenberg.yml")
	if _, err := NewConverter(makeRequest(oPath, path)); err == nil {
		t.Error("Converter should not have been instantiated!")
	}
}

func TestConvert(t *testing.T) {
	path, _ := filepath.Abs("../../_tests/configurations/gotenberg.yml")
	appConfig, _ := config.NewAppConfig(path)
	process.Load(appConfig.CommandsConfig)

	// case 1: uses a request with a single file.
	path, _ = filepath.Abs("../../_tests/file.docx")
	c, _ := NewConverter(makeRequest(path))
	if _, err := c.Convert(); err != nil {
		t.Error("Converter should have been able to convert an Office document to PDF!")
	}

	// case 2: uses a request with two files.
	oPath, _ := filepath.Abs("../../_tests/file.docx")
	path, _ = filepath.Abs("../../_tests/file.pdf")
	c, _ = NewConverter(makeRequest(oPath, path))
	if _, err := c.Convert(); err != nil {
		t.Error("Converter should have been able to convert two files to PDF and merge them!")
	}

	// case 3: uses a request with a single file and a configuration with an unsuitable timeout for the conversion commands.
	path, _ = filepath.Abs("../../_tests/configurations/timeout-gotenberg.yml")
	appConfig, _ = config.NewAppConfig(path)
	process.Load(appConfig.CommandsConfig)
	path, _ = filepath.Abs("../../_tests/file.docx")
	c, _ = NewConverter(makeRequest(path))
	if _, err := c.Convert(); err == nil {
		t.Error("Converter should not have been able to convert an Office document to PDF!")
	}

	// case 4: uses a request with two files and a configuration with an unsuitable timeout for the merge command.
	path, _ = filepath.Abs("../../_tests/configurations/merge-timeout-gotenberg.yml")
	appConfig, _ = config.NewAppConfig(path)
	process.Load(appConfig.CommandsConfig)
	oPath, _ = filepath.Abs("../../_tests/file.docx")
	path, _ = filepath.Abs("../../_tests/file.pdf")
	c, _ = NewConverter(makeRequest(oPath, path))
	if _, err := c.Convert(); err == nil {
		t.Error("Converter should not have been able to merge PDF!")
	}
}

func TestClear(t *testing.T) {
	path, _ := filepath.Abs("../../_tests/file.docx")
	c, _ := NewConverter(makeRequest(path))
	if err := c.Clear(); err != nil {
		t.Error("Converter should have been able to clear itself!")
	}
}
