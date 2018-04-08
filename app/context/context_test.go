package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thecodingmachine/gotenberg/app/converter"
)

func TestWithConverter(t *testing.T) {
	req := WithConverter(httptest.NewRequest(http.MethodPost, "/", nil), &converter.Converter{})
	if c, _ := req.Context().Value(converterKey).(*converter.Converter); c == nil {
		t.Errorf("Context returned a wrong converter: got %v want not nil", c)
	}
}

func TestGetConverter(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// case 1: uses a request without a converter entry in its context.
	if _, err := GetConverter(req); err == nil {
		t.Error("Context should not have a converter entry!")
	}

	// case 2: uses a request with a converter entry in its context.
	req = WithConverter(req, &converter.Converter{})
	if _, err := GetConverter(req); err != nil {
		t.Error("Context should have a converter entry!")
	}
}

func TestGetResultFilePath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// case 1: uses a request without a result file path entry in its context.
	if _, err := GetResultFilePath(req); err == nil {
		t.Error("Context should not have a result file path entry!")
	}

	// case 2: uses a request with a result file path entry in its context.
	req = WithResultFilePath(req, "file.pdf")
	if _, err := GetResultFilePath(req); err != nil {
		t.Error("Context should have a converter entry!")
	}
}

func TestWithResultFilePath(t *testing.T) {
	filePath := "file.pdf"
	req := WithResultFilePath(httptest.NewRequest(http.MethodPost, "/", nil), filePath)
	if path, _ := req.Context().Value(resultFilePathKey).(string); path != filePath {
		t.Errorf("Context returned a wrong converter: got %s want %s", path, filePath)
	}
}

func TestConverterNotFoundError(t *testing.T) {
	err := &converterNotFoundError{}
	if err.Error() != converterNotFoundErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), converterNotFoundErrorMessage)
	}
}

func TestResultFilePathNotFoundError(t *testing.T) {
	err := &resultFilePathNotFoundError{}
	if err.Error() != resultFilePathNotFoundErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), resultFilePathNotFoundErrorMessage)
	}
}
