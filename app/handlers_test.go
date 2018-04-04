package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gulien/gotenberg/app/context"
	ghttp "github.com/gulien/gotenberg/app/http"

	"github.com/justinas/alice"
)

func fakeSuccessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func makeBody(filesPaths ...string) io.Reader {
	if len(filesPaths) == 1 {
		file, _ := os.Open(filesPaths[0])
		defer file.Close()
		return file
	}

	// TODO
	return nil
}

func TestEnforceContentLengthHandler(t *testing.T) {
	h := alice.New(enforceContentLengthHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends an empty request.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// case 2: sends a body.
	path, _ := filepath.Abs("../../_tests/file.docx")
	req = httptest.NewRequest(http.MethodPost, "/", makeBody(path))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestEnforceContentTypeHandler(t *testing.T) {
	h := alice.New(enforceContentTypeHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends a wrong content type.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", string(ghttp.PDFContentType))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnsupportedMediaType {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnsupportedMediaType)
	}

	// case 2: sends a good content type.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", string(ghttp.OctetStreamContentType))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestConvertHandler(t *testing.T) {
	h := alice.New(convertHandler).ThenFunc(fakeSuccessHandler)

	// case 1: sends a request without a content type in its context.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// case 2: sends a request as without body.
	req = context.WithContentType(httptest.NewRequest(http.MethodPost, "/", nil), ghttp.OctetStreamContentType)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// case 3: sends a request as "multipart/form-data" without "files" key.
	/*req = context.WithContentType(httptest.NewRequest(http.MethodPost, "/", nil), ghttp.MultipartFormDataContentType)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}*/
}
