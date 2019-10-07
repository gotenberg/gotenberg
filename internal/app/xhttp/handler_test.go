package xhttp

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestPingHandler(t *testing.T) {
	// should return 200.
	config := conf.DefaultConfig()
	srv := New(config)
	req := httptest.NewRequest(http.MethodGet, pingEndpoint, nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
}

func TestMergeHandler(t *testing.T) {
	config := conf.DefaultConfig()
	srv := New(config)
	// should return 200.
	body, contentType := test.MergeMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is < 0.
	body, contentType = test.MergeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is is > config.MaximumWaitTimeout().
	body, contentType = test.MergeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is invalid.
	body, contentType = test.MergeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 504.
	body, contentType = test.MergeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "0"})
	req = httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusGatewayTimeout, srv, req)
}

func TestHTMLHandler(t *testing.T) {
	config := conf.DefaultConfig()
	srv := New(config)
	endpoint := fmt.Sprintf("%s%s", convertGroupEndpoint, htmlEndpoint)
	// should return 200.
	body, contentType := test.HTMLMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is is > config.MaximumWaitTimeout().
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 504.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusGatewayTimeout, srv, req)
	// should return 400 as "waitDelay" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitDelay" form field
	// value is is > config.MaximumWaitDelay().
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitDelay" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperWidth" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.PaperWidthArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperWidth" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.PaperWidthArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperHeight" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.PaperHeightArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperHeight" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.PaperHeightArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginTop" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginTopArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginTop" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginTopArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginBottom" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginBottomArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginBottom" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginBottomArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginLeft" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginLeftArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginLeft" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginLeftArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginRight" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginRightArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginRight" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.MarginRightArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "landscape" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.LandscapeArgKey): "not a boolean"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is < 0.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is is > config.MaximumGoogleChromeRpccBufferSize().
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "104857601"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is invalid.
	body, contentType = test.HTMLMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "not an int"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
}

func TestURLHandler(t *testing.T) {
	config := conf.DefaultConfig()
	srv := New(config)
	endpoint := fmt.Sprintf("%s%s", convertGroupEndpoint, urlEndpoint)
	// should return 200.
	body, contentType := test.URLMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is is > config.MaximumWaitTimeout().
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 504.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusGatewayTimeout, srv, req)
	// should return 400 as "waitDelay" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitDelay" form field
	// value is is > config.MaximumWaitDelay().
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitDelay" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperWidth" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.PaperWidthArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperWidth" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.PaperWidthArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperHeight" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.PaperHeightArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperHeight" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.PaperHeightArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginTop" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginTopArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginTop" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginTopArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginBottom" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginBottomArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginBottom" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginBottomArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginLeft" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginLeftArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginLeft" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginLeftArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginRight" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginRightArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginRight" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.MarginRightArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "landscape" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.LandscapeArgKey): "not a boolean"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is < 0.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is is > config.MaximumGoogleChromeRpccBufferSize().
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "104857601"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is invalid.
	body, contentType = test.URLMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "not an int"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
}

func TestMarkdownHandler(t *testing.T) {
	config := conf.DefaultConfig()
	srv := New(config)
	endpoint := fmt.Sprintf("%s%s", convertGroupEndpoint, markdownEndpoint)
	// should return 200.
	body, contentType := test.MarkdownMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is is > config.MaximumWaitTimeout().
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 504.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusGatewayTimeout, srv, req)
	// should return 400 as "waitDelay" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitDelay" form field
	// value is is > config.MaximumWaitDelay().
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitDelay" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.WaitDelayArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperWidth" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.PaperWidthArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperWidth" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.PaperWidthArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperHeight" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.PaperHeightArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "paperHeight" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.PaperHeightArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginTop" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginTopArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginTop" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginTopArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginBottom" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginBottomArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginBottom" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginBottomArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginLeft" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginLeftArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginLeft" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginLeftArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginRight" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginRightArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "marginRight" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.MarginRightArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "landscape" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.LandscapeArgKey): "not a boolean"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is < 0.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is is > config.MaximumGoogleChromeRpccBufferSize().
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "104857601"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "googleChromeRpccBufferSize" form field
	// value is invalid.
	body, contentType = test.MarkdownMultipartForm(t, map[string]string{string(resource.GoogleChromeRpccBufferSizeArgKey): "not an int"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
}

func TestOfficeHandler(t *testing.T) {
	config := conf.DefaultConfig()
	srv := New(config)
	endpoint := fmt.Sprintf("%s%s", convertGroupEndpoint, officeEndpoint)
	// should return 200.
	body, contentType := test.OfficeMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is < 0.
	body, contentType = test.OfficeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "-1"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is is > config.MaximumWaitTimeout().
	body, contentType = test.OfficeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "31"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 400 as "waitTimeout" form field
	// value is invalid.
	body, contentType = test.OfficeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// should return 504.
	body, contentType = test.OfficeMultipartForm(t, map[string]string{string(resource.WaitTimeoutArgKey): "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusGatewayTimeout, srv, req)
	// should return 400 as "landscape" form field
	// value is invalid.
	body, contentType = test.OfficeMultipartForm(t, map[string]string{string(resource.LandscapeArgKey): "not a boolean"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
}

func TestWebhook(t *testing.T) {
	status := make(chan error, 2)
	rcv := echo.New()
	rcv.POST("/foo", func(c echo.Context) error {
		if c.Request().Header.Get("Content-type") != "application/pdf" {
			status <- fmt.Errorf("wrong Content-type: got %s want %s", c.Request().Header.Get("Content-type"), "application/pdf")
			return nil
		}
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			status <- err
			return nil
		}
		if body == nil || len(body) == 0 {
			status <- errors.New("empty body")
			return nil
		}
		status <- nil
		return nil
	})
	go func() {
		rcv.Start(":3001")
	}()
	config := conf.DefaultConfig()
	srv := New(config)
	// our custom server should receive the PDF.
	body, contentType := test.MergeMultipartForm(t, map[string]string{string(resource.WebhookURLArgKey): "http://localhost:3001/foo"})
	req := httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	err := <-status
	assert.NoError(t, err)
}

func TestResultFilename(t *testing.T) {
	config := conf.DefaultConfig()
	srv := New(config)
	body, contentType := test.MergeMultipartForm(t, map[string]string{string(resource.ResultFilenameArgKey): "foo.pdf"})
	req := httptest.NewRequest(http.MethodPost, mergeEndpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	assert.Equal(t, "attachment; filename=\"foo.pdf\"", rec.Header().Get("Content-Disposition"))
}
