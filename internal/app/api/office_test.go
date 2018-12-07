package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestOffice(t *testing.T) {
	p := &pm2.Unoconv{}
	err := p.Launch()
	require.Nil(t, err)
	defer p.Shutdown(false)
	body, contentType := test.OfficeMultipartForm(t)
	req := httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	assert.NoError(t, convertOffice(c))
}
