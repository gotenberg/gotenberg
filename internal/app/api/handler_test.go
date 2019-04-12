package api

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestMerge(t *testing.T) {
	opts := DefaultOptions()
	srv := New(opts)
	// OK.
	body, contentType := test.PDFTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Bad request.
	body, contentType = test.PDFTestMultipartForm(t, map[string]string{waitTimeout: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// Timeout.
	body, contentType = test.PDFTestMultipartForm(t, map[string]string{waitTimeout: "0"})
	req = httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
}

func TestHTML(t *testing.T) {
	opts := DefaultOptions()
	srv := New(opts)
	// OK.
	body, contentType := test.HTMLTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Bad request.
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{waitTimeout: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{waitDelay: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{paperWidth: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{paperHeight: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{marginTop: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{marginBottom: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{marginLeft: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{marginRight: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{landscape: "not a bool"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// Timeout.
	body, contentType = test.HTMLTestMultipartForm(t, map[string]string{waitTimeout: "0"})
	req = httptest.NewRequest(http.MethodPost, "/convert/html", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
}

func TestMarkdown(t *testing.T) {
	opts := DefaultOptions()
	srv := New(opts)
	// OK.
	body, contentType := test.MarkdownTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Bad request.
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{waitTimeout: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{waitDelay: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{paperWidth: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{paperHeight: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{marginTop: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{marginBottom: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{marginLeft: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{marginRight: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{landscape: "not a bool"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// Timeout.
	body, contentType = test.MarkdownTestMultipartForm(t, map[string]string{waitTimeout: "0"})
	req = httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
}

func TestURL(t *testing.T) {
	opts := DefaultOptions()
	srv := New(opts)
	// OK.
	body, contentType := test.URLTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Bad request.
	body, contentType = test.URLTestMultipartForm(t, map[string]string{waitTimeout: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{waitDelay: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{paperWidth: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{paperHeight: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{marginTop: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{marginBottom: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{marginLeft: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{marginRight: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, map[string]string{landscape: "not a bool"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// Timeout.
	body, contentType = test.URLTestMultipartForm(t, map[string]string{waitTimeout: "0"})
	req = httptest.NewRequest(http.MethodPost, "/convert/url", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
}

func TestOffice(t *testing.T) {
	opts := DefaultOptions()
	srv := New(opts)
	// OK.
	body, contentType := test.OfficeTestMultipartForm(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	// Bad request.
	body, contentType = test.OfficeTestMultipartForm(t, map[string]string{waitTimeout: "not a float"})
	req = httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.OfficeTestMultipartForm(t, map[string]string{landscape: "not a bool"})
	req = httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	body, contentType = test.URLTestMultipartForm(t, nil)
	req = httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusBadRequest, srv, req)
	// Timeout.
	body, contentType = test.OfficeTestMultipartForm(t, map[string]string{waitTimeout: "0"})
	req = httptest.NewRequest(http.MethodPost, "/convert/office", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusRequestTimeout, srv, req)
}

func TestConcurrent(t *testing.T) {
	opts := DefaultOptions()
	opts.DefaultWaitTimeout = 30
	srv := New(opts)
	// Merge.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.MarkdownTestMultipartForm(t, nil)
			req := httptest.NewRequest(http.MethodPost, "/convert/html", body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want %d got %d", http.StatusOK, rec.Code)
			}
			return nil
		},
		10,
	)
	// HTML.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.HTMLTestMultipartForm(t, nil)
			req := httptest.NewRequest(http.MethodPost, "/convert/html", body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want %d got %d", http.StatusOK, rec.Code)
			}
			return nil
		},
		10,
	)
	// Markdown.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.MarkdownTestMultipartForm(t, nil)
			req := httptest.NewRequest(http.MethodPost, "/convert/markdown", body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want %d got %d", http.StatusOK, rec.Code)
			}
			return nil
		},
		10,
	)
	// URL.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.URLTestMultipartForm(t, nil)
			req := httptest.NewRequest(http.MethodPost, "/convert/url", body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want %d got %d", http.StatusOK, rec.Code)
			}
			return nil
		},
		10,
	)
	// Office.
	test.AssertConcurrent(
		t,
		func() error {
			body, contentType := test.OfficeTestMultipartForm(t, nil)
			req := httptest.NewRequest(http.MethodPost, "/convert/office", body)
			req.Header.Set(echo.HeaderContentType, contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				return fmt.Errorf("wrong status code: want %d got %d", http.StatusOK, rec.Code)
			}
			return nil
		},
		10,
	)
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
	opts := DefaultOptions()
	srv := New(opts)
	body, contentType := test.PDFTestMultipartForm(t, map[string]string{webhookURL: "http://localhost:3001/foo"})
	req := httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	test.AssertStatusCode(t, http.StatusOK, srv, req)
	err := <-status
	assert.NoError(t, err)
}

func TestResultFilename(t *testing.T) {
	opts := DefaultOptions()
	srv := New(opts)
	body, contentType := test.PDFTestMultipartForm(t, map[string]string{resultFilename: "foo.pdf"})
	req := httptest.NewRequest(http.MethodPost, "/convert/merge", body)
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	assert.Equal(t, "attachment; filename=\"foo.pdf\"", rec.Header().Get("Content-Disposition"))
}
