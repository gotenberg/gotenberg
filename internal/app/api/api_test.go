package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestDefaultWaitTimeout(t *testing.T) {
	opts := DefaultOptions()
	opts.DefaultWaitTimeout = 0
	srv := New(opts)
	// testing if timeout.
	body, contentType := test.URLTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
	// testing if no timeout.
	body, contentType = test.URLTestMultipartForm(t, map[string]string{waitTimeout: "10"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
}

func TestDisableChromeEndpoints(t *testing.T) {
	opts := DefaultOptions()
	opts.EnableChromeEndpoints = false
	srv := New(opts)
	// Ping.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge.
	body, contentType := test.PDFTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML.
	body, contentType = test.HTMLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Markdown.
	body, contentType = test.MarkdownTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// URL.
	body, contentType = test.URLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Office.
	body, contentType = test.OfficeTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
}

func TestDisableUnoconvEndpoints(t *testing.T) {
	opts := DefaultOptions()
	opts.EnableUnoconvEndpoints = false
	srv := New(opts)
	// Ping.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge.
	body, contentType := test.PDFTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML.
	body, contentType = test.HTMLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Markdown.
	body, contentType = test.MarkdownTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// URL.
	body, contentType = test.URLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Office.
	body, contentType = test.OfficeTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
}
func TestDisableChromeAndUnoconvEndpoints(t *testing.T) {
	opts := DefaultOptions()
	opts.EnableChromeEndpoints = false
	opts.EnableUnoconvEndpoints = false
	srv := New(opts)
	// Ping.
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Merge.
	body, contentType := test.PDFTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// HTML.
	body, contentType = test.HTMLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Markdown.
	body, contentType = test.MarkdownTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// URL.
	body, contentType = test.URLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
	// Office.
	body, contentType = test.OfficeTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusNotFound, srv, req)
}
