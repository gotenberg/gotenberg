package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// DummyEchoContext creates a
// echo.Context without anything.
func DummyEchoContext() echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

// EchoContextMultipart creates a
// echo.Context with form files.
func EchoContextMultipart(t *testing.T) echo.Context {
	e := echo.New()
	body, contentType := MergeMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}
