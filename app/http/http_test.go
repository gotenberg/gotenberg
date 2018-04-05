package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFindAuthorizedContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// case 1: uses a request without a content type entry in its header.
	if _, err := FindAuthorizedContentType(req.Header); err == nil {
		t.Error("It should not have been able to retrieve an authorized content type from header!")
	}

	// case 2: uses a request with a content type entry in its header.
	req.Header.Set("Content-Type", string(MultipartFormDataContentType))
	if _, err := FindAuthorizedContentType(req.Header); err != nil {
		t.Error("It should have been able to retrieve an authorized content type from header!")
	}
}

func TestSniffContentType(t *testing.T) {
	// case 1: uses a file with a wrong content type.
	path, _ := filepath.Abs("../../_tests/configurations/gotenberg.yml")
	f, _ := os.Open(path)
	defer f.Close()
	if _, err := SniffContentType(f); err == nil {
		t.Error("It should not have been able to retrieve an authorized content type from an YAML file!")
	}

	// case 2: uses a file with a correct content type.
	path, _ = filepath.Abs("../../_tests/file.pdf")
	f, _ = os.Open(path)
	defer f.Close()
	if _, err := SniffContentType(f); err != nil {
		t.Error("It should have been able to retrieve an authorized content type from a PDF file!")
	}
}

func TestNotAuthorizedContentTypeError(t *testing.T) {
	err := &notAuthorizedContentTypeError{}
	message := fmt.Sprintf("Accepted values for 'Content-Type': %s, %s", OctetStreamContentType, MultipartFormDataContentType)
	if err.Error() != message {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), message)
	}
}

func TestNotAuthorizedFileContentTypeError(t *testing.T) {
	err := &notAuthorizedFileContentTypeError{}
	if err.Error() != notAuthorizedFileContentTypeErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), notAuthorizedFileContentTypeErrorMessage)
	}
}
