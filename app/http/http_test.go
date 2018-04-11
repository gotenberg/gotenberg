package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckAuthorizedContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// case 1: uses a request without a content type entry in its header.
	if err := CheckAuthorizedContentType(req.Header); err == nil {
		t.Error("Function should not have been able to retrieve an authorized content type from request's header")
	}

	// case 2: uses a request with a content type entry in its header.
	req.Header.Set("Content-Type", string(MultipartFormDataContentType))
	if err := CheckAuthorizedContentType(req.Header); err != nil {
		t.Error("Function should have been able to retrieve an authorized content type from request's header")
	}

	// case 3: uses a request with a composed content type entry in its header.
	req.Header.Set("Content-Type", "multipart/form-data; boundary=—-WebKitFormBoundary7MA4YWxkTrZu0gW")
	if err := CheckAuthorizedContentType(req.Header); err != nil {
		t.Error("Function should have been able to retrieve an authorized content type from request's header")
	}

}

func TestNotAuthorizedContentTypeError(t *testing.T) {
	err := &notAuthorizedContentTypeError{}
	expected := fmt.Sprintf("Accepted value for 'Content-Type': %s", MultipartFormDataContentType)
	if err.Error() != expected {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), expected)
	}
}
