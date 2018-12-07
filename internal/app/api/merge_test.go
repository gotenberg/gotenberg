package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestMerge(t *testing.T) {
	body, contentType := test.PDFMultipartForm(t)
	req := httptest.NewRequest(http.MethodPost, "/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	assert.NoError(t, merge(c))
}
