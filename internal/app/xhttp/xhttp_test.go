package xhttp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestNonExistingEndpoint(t *testing.T) {
	config, err := conf.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// "/" endpoint should return 404.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
}

func TestDisableChromeEndpoints(t *testing.T) {
	os.Setenv(conf.DisableGoogleChromeEnvVar, "1")
	config, err := conf.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// Ping endpoint should return 200.
	req := httptest.NewRequest(http.MethodGet, pingEndpoint(config), nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge endpoint should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML endpoint should return 404.
	body, contentType = test.HTMLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, htmlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// URL endpoint should return 404.
	body, contentType = test.URLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, urlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Markdown endpoint should return 404.
	body, contentType = test.MarkdownMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, markdownEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Office endpoint should return 200.
	body, contentType = test.OfficeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, officeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// finally...
	os.Setenv(conf.DisableGoogleChromeEnvVar, "0")
}

func TestDisableUnoconvEndpoints(t *testing.T) {
	os.Setenv(conf.DisableUnoconvEnvVar, "1")
	config, err := conf.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// Ping endpoint should return 200.
	req := httptest.NewRequest(http.MethodGet, pingEndpoint(config), nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge endpoint should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML endpoint should return 200.
	body, contentType = test.HTMLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, htmlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// URL endpoint should return 200.
	body, contentType = test.URLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, urlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Markdown endpoint should return 200.
	body, contentType = test.MarkdownMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, markdownEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Office endpoint should return 404.
	body, contentType = test.OfficeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, officeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// finally...
	os.Setenv(conf.DisableUnoconvEnvVar, "0")
}
func TestDisableChromeAndUnoconvEndpoints(t *testing.T) {
	os.Setenv(conf.DisableGoogleChromeEnvVar, "1")
	os.Setenv(conf.DisableUnoconvEnvVar, "1")
	config, err := conf.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// Ping endpoint should return 200.
	req := httptest.NewRequest(http.MethodGet, pingEndpoint(config), nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge endpoint should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML endpoint should return 404.
	body, contentType = test.HTMLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, htmlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// URL endpoint should return 404.
	body, contentType = test.URLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, urlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Markdown endpoint should return 404.
	body, contentType = test.MarkdownMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, markdownEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Office endpoint should return 404.
	body, contentType = test.OfficeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, officeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// finally...
	os.Setenv(conf.DisableGoogleChromeEnvVar, "0")
	os.Setenv(conf.DisableUnoconvEnvVar, "0")
}

func TestCustomRootPath(t *testing.T) {
	os.Setenv(conf.RootPathEnvVar, "/foo/")
	config, err := conf.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// Ping endpoint should return 200.
	req := httptest.NewRequest(http.MethodGet, pingEndpoint(config), nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge endpoint should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML endpoint should return 200.
	body, contentType = test.HTMLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, htmlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// URL endpoint should return 200.
	body, contentType = test.URLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, urlEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Markdown endpoint should return 200.
	body, contentType = test.MarkdownMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, markdownEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Office endpoint should return 200.
	body, contentType = test.OfficeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, officeEndpoint(config), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// finally...
	os.Setenv(conf.RootPathEnvVar, "/")
}
