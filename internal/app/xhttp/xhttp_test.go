package xhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestDisableChromeEndpoints(t *testing.T) {
	os.Setenv(conf.DisableGoogleChromeEnvVar, "1")
	config, err := conf.FromEnv()
	assert.Nil(t, err)
	// TODO
	srv := New(config, nil, nil)
	// Ping endpoint should return 200.
	req := httptest.NewRequest(http.MethodGet, pingEndpoint, nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge endpoint should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML endpoint should return 404.
	body, contentType = test.HTMLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, htmlEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// URL endpoint should return 404.
	body, contentType = test.URLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, urlEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Markdown endpoint should return 404.
	body, contentType = test.MarkdownMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, markdownEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Office endpoint should return 200.
	body, contentType = test.OfficeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, officeEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// finally...
	os.Setenv(conf.DisableGoogleChromeEnvVar, "0")
}

func TestDisableUnoconvEndpoints(t *testing.T) {
	os.Setenv(conf.DisableUnoconvEnvVar, "1")
	config, err := conf.FromEnv()
	assert.Nil(t, err)
	// TODO
	srv := New(config, nil, nil)
	// Ping endpoint should return 200.
	req := httptest.NewRequest(http.MethodGet, pingEndpoint, nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge endpoint should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML endpoint should return 200.
	body, contentType = test.HTMLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, htmlEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// URL endpoint should return 200.
	body, contentType = test.URLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, urlEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Markdown endpoint should return 404.
	body, contentType = test.MarkdownMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, markdownEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Office endpoint should return 404.
	body, contentType = test.OfficeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, officeEndpoint), body)
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
	// TODO
	srv := New(config, nil, nil)
	// Ping endpoint should return 200.
	req := httptest.NewRequest(http.MethodGet, pingEndpoint, nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge endpoint should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML endpoint should return 404.
	body, contentType = test.HTMLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, htmlEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// URL endpoint should return 404.
	body, contentType = test.URLMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, urlEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Markdown endpoint should return 404.
	body, contentType = test.MarkdownMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, markdownEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Office endpoint should return 404.
	body, contentType = test.OfficeMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", convertGroupEndpoint, officeEndpoint), body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// finally...
	os.Setenv(conf.DisableGoogleChromeEnvVar, "0")
	os.Setenv(conf.DisableUnoconvEnvVar, "0")
}
