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

	"github.com/justinas/alice"
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

func fakeSuccessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestGetHandlersChain(t *testing.T) {
	// dumb test to improve code coverage...
	if GetHandlersChain() == nil {
		t.Errorf("Handler chains should not be nil")
	}
}

func TestRequestHasNoContentError(t *testing.T) {
	err := &requestHasNoContentError{}
	if err.Error() != requestHasNoContentErrorMessage {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), requestHasNoContentErrorMessage)
	}
}

func TestEnforceContentLengthHandler(t *testing.T) {
	var (
		req *http.Request
		rr  *httptest.ResponseRecorder
	)

	h := alice.New(enforceContentLengthHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends an empty request.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusBadRequest)
	}

	// case 2: sends a real body.
	path, _ := filepath.Abs("../_tests/file.docx")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, makeRequest(path))
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusOK)
	}
}

func TestEnforceContentTypeHandler(t *testing.T) {
	var (
		req *http.Request
		rr  *httptest.ResponseRecorder
	)

	h := alice.New(enforceContentTypeHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends a wrong content type.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/pdf")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnsupportedMediaType {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusUnsupportedMediaType)
	}

	// case 2: sends a good content type.
	path, _ := filepath.Abs("../_tests/file.docx")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, makeRequest(path))
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong a status code: got '%v' want '%v'", status, http.StatusOK)
	}
}

func TestConvertHandler(t *testing.T) {
	var (
		req   *http.Request
		rr    *httptest.ResponseRecorder
		path  string
		oPath string
	)

	h := alice.New(convertHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends a request without body.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusInternalServerError)
	}

	// case 2: sends a request with no file.
	req = makeRequest()
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusBadRequest)
	}

	load("../_tests/configurations/merge-timeout-gotenberg.yml")

	// case 3: sends a request with two files and using an unsuitable timeout for merge commande.
	path, _ = filepath.Abs("../_tests/file.pdf")
	oPath, _ = filepath.Abs("../_tests/file.docx")
	req = makeRequest(path, oPath)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusInternalServerError)
	}

	load("../_tests/configurations/gotenberg.yml")

	// case 4: sends a request with two files.
	path, _ = filepath.Abs("../_tests/file.pdf")
	oPath, _ = filepath.Abs("../_tests/file.docx")
	req = makeRequest(path, oPath)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusOK)
	}

	// case 5: sends five requests (almost) simultany.
	path, _ = filepath.Abs("../_tests/file.docx")
	filesPaths := []string{
		path,
		path,
		path,
		path,
		path,
	}

	for i := 0; i < len(filesPaths); i++ {
		go func(i int) {
			req := makeRequest(filesPaths[i])
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if status := rr.Code; status != http.StatusOK {
				t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusOK)
			}
		}(i)
	}
}

func TestServeHandler(t *testing.T) {
	var (
		req *http.Request
		rr  *httptest.ResponseRecorder
	)

	h := alice.New().ThenFunc(serveHandler)

	// case 1: sends a request without a result file path entry in its context.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusInternalServerError)
	}

	// case 2: sends a request with a wrong result file path entry in its context.
	req = context.WithResultFilePath(httptest.NewRequest(http.MethodPost, "/", nil), "file")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusInternalServerError)
	}

	// case 3: sends a request with a correct result file path entry in its context.
	path, _ := filepath.Abs("../_tests/file.pdf")
	req = context.WithResultFilePath(httptest.NewRequest(http.MethodPost, "/", nil), path)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned a wrong status code: got '%v' want '%v'", status, http.StatusOK)
	}
}
