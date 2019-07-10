package api

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/handler"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/middleware"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/config"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestPing(t *testing.T) {
	endpoint := handler.PingEndpoint
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// should be OK.
	req := httptest.NewRequest(http.MethodGet, endpoint, nil)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
}

func TestMerge(t *testing.T) {
	os.Setenv(middleware.TestingTraceEnvVar, "1")
	endpoint := handler.MergeEndpoint
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// should be OK.
	body, contentType := test.PDFTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// bad request.
	body, contentType = test.PDFTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// timeout.
	body, contentType = test.PDFTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
	// should have no more resources.
	test.AssertDirectoryEmpty(t, middleware.TestsTracePrefix)
	err = os.RemoveAll(middleware.TestsTracePrefix)
	assert.Nil(t, err)
	os.Unsetenv(middleware.TestingTraceEnvVar)
}

func TestHTML(t *testing.T) {
	os.Setenv(middleware.TestingTraceEnvVar, "1")
	endpoint := fmt.Sprintf("%s%s", handler.ConvertGroupEndpoint, handler.HTMLEndpoint)
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// should be OK.
	body, contentType := test.HTMLTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// bad request.
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.WaitDelayFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.PaperWidthFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.PaperHeightFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.MarginTopFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.MarginBottomFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.MarginLeftFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.MarginRightFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.LandscapeFormField: "not a bool"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// timeout.
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
	// should have no more resources.
	test.AssertDirectoryEmpty(t, middleware.TestsTracePrefix)
	err = os.RemoveAll(middleware.TestsTracePrefix)
	assert.Nil(t, err)
	os.Unsetenv(middleware.TestingTraceEnvVar)
}

func TestMarkdown(t *testing.T) {
	os.Setenv(middleware.TestingTraceEnvVar, "1")
	endpoint := fmt.Sprintf("%s%s", handler.ConvertGroupEndpoint, handler.MarkdownEndpoint)
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// should be OK.
	body, contentType := test.MarkdownTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// bad request.
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.WaitDelayFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.PaperWidthFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.PaperHeightFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.MarginTopFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.MarginBottomFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.MarginLeftFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.MarginRightFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.LandscapeFormField: "not a bool"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// timeout.
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
	// should have no more resources.
	test.AssertDirectoryEmpty(t, middleware.TestsTracePrefix)
	err = os.RemoveAll(middleware.TestsTracePrefix)
	assert.Nil(t, err)
	os.Unsetenv(middleware.TestingTraceEnvVar)
}

func TestURL(t *testing.T) {
	os.Setenv(middleware.TestingTraceEnvVar, "1")
	endpoint := fmt.Sprintf("%s%s", handler.ConvertGroupEndpoint, handler.URLEndpoint)
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// should be OK.
	body, contentType := test.URLTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// bad request.
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.WaitDelayFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.PaperWidthFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.PaperHeightFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.MarginTopFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.MarginBottomFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.MarginLeftFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.MarginRightFormField: "not a float"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.LandscapeFormField: "not a bool"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// timeout.
	body, contentType = test.URLTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "0"})
	req = httptest.NewRequest(http.MethodPost, endpoint, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
	// should have no more resources.
	test.AssertDirectoryEmpty(t, middleware.TestsTracePrefix)
	err = os.RemoveAll(middleware.TestsTracePrefix)
	assert.Nil(t, err)
	os.Unsetenv(middleware.TestingTraceEnvVar)
}

func TestConcurrent(t *testing.T) {
	const concurrentRequests int = 4
	os.Setenv(middleware.TestingTraceEnvVar, "1")
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	// Merge.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.PDFTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "120"})
			req := httptest.NewRequest(http.MethodPost, handler.MergeEndpoint, body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want '%d' got '%d'", http.StatusOK, rec.Code)
			}
			return nil
		},
		concurrentRequests,
	)
	// HTML.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.HTMLTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "120"})
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", handler.ConvertGroupEndpoint, handler.HTMLEndpoint), body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want '%d' got '%d'", http.StatusOK, rec.Code)
			}
			return nil
		},
		concurrentRequests,
	)
	// Markdown.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.MarkdownTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "120"})
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", handler.ConvertGroupEndpoint, handler.MarkdownEndpoint), body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want '%d' got '%d'", http.StatusOK, rec.Code)
			}
			return nil
		},
		concurrentRequests,
	)
	// URL.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.URLTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "120"})
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", handler.ConvertGroupEndpoint, handler.URLEndpoint), body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want '%d' got '%d'", http.StatusOK, rec.Code)
			}
			return nil
		},
		concurrentRequests,
	)
	// Office.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.OfficeTestMultipartForm(t, map[string]string{resource.WaitTimeoutFormField: "120"})
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", handler.ConvertGroupEndpoint, handler.OfficeEndpoint), body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want '%d' got '%d'", http.StatusOK, rec.Code)
			}
			return nil
		},
		concurrentRequests,
	)
	// should have no more resources.
	test.AssertDirectoryEmpty(t, middleware.TestsTracePrefix)
	err = os.RemoveAll(middleware.TestsTracePrefix)
	assert.Nil(t, err)
	os.Unsetenv(middleware.TestingTraceEnvVar)
}

func TestWebhook(t *testing.T) {
	status := make(chan error, 2)
	rcv := echo.New()
	rcv.POST("/foo", func(c echo.Context) error {
		if c.Request().Header.Get("Content-type") != "application/pdf" {
			status <- fmt.Errorf("wrong Content-type: got '%s' want '%s'", c.Request().Header.Get("Content-type"), "application/pdf")
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
	os.Setenv(middleware.TestingTraceEnvVar, "1")
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	body, contentType := test.PDFTestMultipartForm(t, map[string]string{resource.WebhookURLFormField: "http://localhost:3001/foo"})
	req := httptest.NewRequest(http.MethodPost, "/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	err = <-status
	assert.NoError(t, err)
	// should have no more resources.
	test.AssertDirectoryEmpty(t, middleware.TestsTracePrefix)
	err = os.RemoveAll(middleware.TestsTracePrefix)
	assert.Nil(t, err)
	os.Unsetenv(middleware.TestingTraceEnvVar)
}

func TestResultFilename(t *testing.T) {
	os.Setenv(middleware.TestingTraceEnvVar, "1")
	config, err := config.FromEnv()
	assert.Nil(t, err)
	srv := New(config)
	body, contentType := test.PDFTestMultipartForm(t, map[string]string{resource.ResultFilenameFormField: "foo.pdf"})
	req := httptest.NewRequest(http.MethodPost, "/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	assert.Equal(t, "attachment; filename=\"foo.pdf\"", rec.Header().Get("Content-Disposition"))
	// should have no more resources.
	test.AssertDirectoryEmpty(t, middleware.TestsTracePrefix)
	err = os.RemoveAll(middleware.TestsTracePrefix)
	assert.Nil(t, err)
	os.Unsetenv(middleware.TestingTraceEnvVar)
}
