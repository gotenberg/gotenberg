package app

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/thecodingmachine/gotenberg/app/config"
	"github.com/thecodingmachine/gotenberg/app/context"
	"github.com/thecodingmachine/gotenberg/app/converter/process"
	ghttp "github.com/thecodingmachine/gotenberg/app/http"

	"github.com/justinas/alice"
)

func fakeSuccessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

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

func TestEnforceContentLengthHandler(t *testing.T) {
	h := alice.New(enforceContentLengthHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends an empty request.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// case 2: sends a body.
	path, _ := filepath.Abs("../_tests/file.docx")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, makeRequest(path))
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestEnforceContentTypeHandler(t *testing.T) {
	h := alice.New(enforceContentTypeHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends a wrong content type.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/pdf")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnsupportedMediaType {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusUnsupportedMediaType)
	}

	// case 2: sends a good content type.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", string(ghttp.MultipartFormDataContentType))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong a status code: got %v want %v", status, http.StatusOK)
	}
}

func TestConvertHandler(t *testing.T) {
	h := alice.New(convertHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends a request without body.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// case 2: sends a request with two files and using an unsuitable timeout for merge commande.
	path, _ := filepath.Abs("../_tests/configurations/merge-timeout-gotenberg.yml")
	appConfig, _ := config.NewAppConfig(path)
	process.Load(appConfig.CommandsConfig)

	oPath, _ := filepath.Abs("../_tests/file.docx")
	path, _ = filepath.Abs("../_tests/configurations/gotenberg.yml")
	req = makeRequest(oPath, path)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// case 3: sends two requests (almost) simultany.
	path, _ = filepath.Abs("../_tests/configurations/gotenberg.yml")
	appConfig, _ = config.NewAppConfig(path)
	process.Load(appConfig.CommandsConfig)

	// TODO

	// case 4: sends a request with two files.
	oPath, _ = filepath.Abs("../_tests/file.docx")
	path, _ = filepath.Abs("../_tests/file.pdf")
	req = makeRequest(oPath, path)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServeHandler(t *testing.T) {
	h := alice.New().ThenFunc(serveHandler)

	// case 1: sends a request without a result file path entry in its context.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// case 2: sends a request with a wrong result file path entry in its context.
	req = context.WithResultFilePath(httptest.NewRequest(http.MethodPost, "/", nil), "file")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// case 3: sends a request with a correct result file path entry in its context.
	path, _ := filepath.Abs("../_tests/file.pdf")
	req = context.WithResultFilePath(httptest.NewRequest(http.MethodPost, "/", nil), path)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned a wrong status code: got %v want %v", status, http.StatusOK)
	}
}
