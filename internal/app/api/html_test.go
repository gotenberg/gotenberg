package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestHTML(t *testing.T) {
	body, contentType := test.HTMLTestMultipartForm(t)
	req := httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	if assert.NoError(t, convertHTML(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/pdf", rec.Header().Get(echo.HeaderContentType))
	}
}
