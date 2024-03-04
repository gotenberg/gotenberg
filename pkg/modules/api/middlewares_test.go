package api

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestParseError(t *testing.T) {
	for i, tc := range []struct {
		err           error
		expectStatus  int
		expectMessage string
	}{
		{
			err:           echo.ErrInternalServerError,
			expectStatus:  http.StatusInternalServerError,
			expectMessage: http.StatusText(http.StatusInternalServerError),
		},
		{
			err:           gotenberg.ErrFiltered,
			expectStatus:  http.StatusForbidden,
			expectMessage: http.StatusText(http.StatusForbidden),
		},
		{
			err:           gotenberg.ErrMaximumQueueSizeExceeded,
			expectStatus:  http.StatusTooManyRequests,
			expectMessage: http.StatusText(http.StatusTooManyRequests),
		},
		{
			err:           gotenberg.ErrPdfFormatNotSupported,
			expectStatus:  http.StatusBadRequest,
			expectMessage: "A least one PDF engine does not handle one of the requested PDF format, while other have failed to convert for other reasons",
		},
		{
			err: WrapError(
				errors.New("foo"),
				NewSentinelHttpError(http.StatusBadRequest, "foo"),
			),
			expectStatus:  http.StatusBadRequest,
			expectMessage: "foo",
		},
	} {
		actualStatus, actualMessage := ParseError(tc.err)

		if actualStatus != tc.expectStatus {
			t.Errorf("test %d: expected HTTP status code %d but got %d", i, tc.expectStatus, actualStatus)
		}

		if actualMessage != tc.expectMessage {
			t.Errorf("test %d: expected message '%s' but got '%s'", i, tc.expectMessage, actualMessage)
		}
	}
}

func TestHttpErrorHandler(t *testing.T) {
	for i, tc := range []struct {
		err           error
		expectStatus  int
		expectMessage string
	}{
		{
			err:           echo.ErrInternalServerError,
			expectStatus:  http.StatusInternalServerError,
			expectMessage: http.StatusText(http.StatusInternalServerError),
		},
		{
			err:           context.DeadlineExceeded,
			expectStatus:  http.StatusServiceUnavailable,
			expectMessage: http.StatusText(http.StatusServiceUnavailable),
		},
		{
			err: WrapError(
				errors.New("foo"),
				NewSentinelHttpError(http.StatusBadRequest, "foo"),
			),
			expectStatus:  http.StatusBadRequest,
			expectMessage: "foo",
		},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/foo", nil)

		srv := echo.New()
		srv.HideBanner = true
		srv.HidePort = true

		c := srv.NewContext(request, recorder)
		c.Set("logger", zap.NewNop())

		handler := httpErrorHandler()
		handler(tc.err, c)

		contentType := recorder.Header().Get(echo.HeaderContentType)
		if contentType != echo.MIMETextPlainCharsetUTF8 {
			t.Errorf("test %d: expected %s '%s' but got '%s'", i, echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8, contentType)
		}

		// Note: we cannot test the trace header in the response here, as it is set in the trace middleware.

		if recorder.Code != tc.expectStatus {
			t.Errorf("test %d: expected HTTP status code %d but got %d", i, tc.expectStatus, recorder.Code)
		}

		if recorder.Body.String() != tc.expectMessage {
			t.Errorf("test %d: expected message '%s' but got '%s'", i, tc.expectMessage, recorder.Body.String())
		}
	}
}

func TestLatencyMiddleware(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/foo", nil)

	srv := echo.New()
	srv.HideBanner = true
	srv.HidePort = true

	c := srv.NewContext(request, recorder)

	err := latencyMiddleware()(
		func(c echo.Context) error {
			return nil
		},
	)(c)
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	startTime := c.Get("startTime").(time.Time)
	now := time.Now()

	if now.Before(startTime) {
		t.Errorf("expected start time %s to be < %s", startTime, now)
	}
}

func TestRootPathMiddleware(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/foo", nil)

	srv := echo.New()
	srv.HideBanner = true
	srv.HidePort = true

	c := srv.NewContext(request, recorder)

	err := rootPathMiddleware("foo")(
		func(c echo.Context) error {
			return nil
		},
	)(c)
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	rootPath := c.Get("rootPath").(string)

	if rootPath != "foo" {
		t.Errorf("expected '%s' but got '%s", "foo", rootPath)
	}
}

func TestTraceMiddleware(t *testing.T) {
	for i, tc := range []struct {
		trace string
	}{
		{
			trace: "foo",
		},
		{
			trace: "",
		},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/foo", nil)

		srv := echo.New()
		srv.HideBanner = true
		srv.HidePort = true

		c := srv.NewContext(request, recorder)

		if tc.trace != "" {
			c.Request().Header.Set("Gotenberg-Trace", tc.trace)
		}

		err := traceMiddleware("Gotenberg-Trace")(
			func(c echo.Context) error {
				return nil
			},
		)(c)
		if err != nil {
			t.Fatalf("test %d: expected no error but got: %v", i, err)
		}

		trace := c.Get("trace").(string)

		if trace == "" {
			t.Errorf("test %d: expected non empty trace in context", i)
		}

		if tc.trace != "" && trace != tc.trace {
			t.Errorf("test %d: expected context trace '%s' but got '%s'", i, tc.trace, trace)
		}

		if tc.trace == "" && trace == tc.trace {
			t.Errorf("test %d: expected context trace different from '%s' but got '%s'", i, tc.trace, trace)
		}

		responseTrace := recorder.Header().Get("Gotenberg-Trace")

		if tc.trace != "" && responseTrace != tc.trace {
			t.Errorf("test %d: expected header trace '%s' but got '%s'", i, tc.trace, responseTrace)
		}

		if tc.trace == "" && responseTrace == tc.trace {
			t.Errorf("test %d: expected header trace different from '%s' but got '%s'", i, tc.trace, responseTrace)
		}
	}
}

func TestLoggerMiddleware(t *testing.T) {
	for i, tc := range []struct {
		request     *http.Request
		next        echo.HandlerFunc
		skipLogging bool
	}{
		{
			request: httptest.NewRequest(http.MethodGet, "/", nil),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return errors.New("foo")
				}
			}(),
		},
		{
			request: httptest.NewRequest(http.MethodGet, "/health", nil),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return nil
				}
			}(),
			skipLogging: true,
		},
		{
			request: httptest.NewRequest(http.MethodGet, "/health", nil),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return nil
				}
			}(),
		},
	} {
		recorder := httptest.NewRecorder()

		srv := echo.New()
		srv.HideBanner = true
		srv.HidePort = true

		c := srv.NewContext(tc.request, recorder)
		c.Set("startTime", time.Now())
		c.Set("trace", "foo")
		c.Set("rootPath", "/")

		var disableLoggingForPaths []string
		if tc.skipLogging {
			disableLoggingForPaths = append(disableLoggingForPaths, tc.request.RequestURI)
		}

		err := loggerMiddleware(zap.NewNop(), disableLoggingForPaths)(tc.next)(c)
		if err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestContextMiddleware(t *testing.T) {
	buildMultipartFormDataRequest := func() *http.Request {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		defer func() {
			err := writer.Close()
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		}()

		err := writer.WriteField("foo", "foo")
		if err != nil {
			t.Fatalf("expected no error but got: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())

		return req
	}

	for i, tc := range []struct {
		request           *http.Request
		next              echo.HandlerFunc
		expectErr         bool
		expectStatus      int
		expectContentType string
		expectFilename    string
	}{
		{
			request:   httptest.NewRequest(http.MethodGet, "/", nil),
			expectErr: true,
		},
		{
			request: buildMultipartFormDataRequest(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return ErrAsyncProcess
				}
			}(),
			expectStatus: http.StatusNoContent,
		},
		{
			request: buildMultipartFormDataRequest(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return errors.New("foo")
				}
			}(),
			expectErr: true,
		},
		{
			request: buildMultipartFormDataRequest(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return nil
				}
			}(),
			expectErr: true,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Output-Filename", "foo")

				return req
			}(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					ctx := c.Get("context").(*Context)
					ctx.outputPaths = []string{
						"/tests/test/testdata/api/sample2.pdf",
					}

					return nil
				}
			}(),
			expectStatus:      http.StatusOK,
			expectContentType: "application/pdf",
			expectFilename:    "foo.pdf",
		},
		{
			request: buildMultipartFormDataRequest(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					ctx := c.Get("context").(*Context)
					ctx.outputPaths = []string{
						"/tests/test/testdata/api/sample1.txt",
						"/tests/test/testdata/api/sample2.pdf",
					}

					return nil
				}
			}(),
			expectStatus:      http.StatusOK,
			expectContentType: "application/zip",
		},
	} {
		recorder := httptest.NewRecorder()

		srv := echo.New()
		srv.HideBanner = true
		srv.HidePort = true

		c := srv.NewContext(tc.request, recorder)
		c.Set("logger", zap.NewNop())
		c.Set("trace", "foo")
		c.Set("startTime", time.Now())

		err := contextMiddleware(gotenberg.NewFileSystem(), time.Duration(10)*time.Second)(tc.next)(c)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if err != nil {
			continue
		}

		if recorder.Code != tc.expectStatus {
			t.Errorf("test %d: expected HTTP status code %d but got %d", i, tc.expectStatus, recorder.Code)
		}

		if tc.expectStatus == http.StatusNoContent {
			continue
		}

		contentType := recorder.Header().Get(echo.HeaderContentType)
		if contentType != tc.expectContentType {
			t.Errorf("test %d: expected %s '%s' but got '%s'", i, echo.HeaderContentType, tc.expectContentType, contentType)
		}

		contentDisposition := recorder.Header().Get(echo.HeaderContentDisposition)
		if !strings.Contains(contentDisposition, tc.expectFilename) {
			t.Errorf("test %d: expected %s '%s' to contain '%s'", i, echo.HeaderContentDisposition, contentDisposition, tc.expectFilename)
		}
	}
}

func TestHardTimeoutMiddleware(t *testing.T) {
	for i, tc := range []struct {
		next              echo.HandlerFunc
		timeout           time.Duration
		expectErr         bool
		expectHardTimeout bool
	}{
		{
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return nil
				}
			}(),
			timeout: time.Duration(100) * time.Millisecond,
		},
		{
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					panic("foo")
				}
			}(),
			timeout:           time.Duration(100) * time.Millisecond,
			expectErr:         true,
			expectHardTimeout: true,
		},
		{
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return errors.New("foo")
				}
			}(),
			timeout:   time.Duration(100) * time.Millisecond,
			expectErr: true,
		},
		{
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					time.Sleep(time.Duration(200) * time.Millisecond)

					return nil
				}
			}(),
			timeout:           time.Duration(100) * time.Millisecond,
			expectErr:         true,
			expectHardTimeout: true,
		},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/foo", nil)

		srv := echo.New()
		srv.HideBanner = true
		srv.HidePort = true

		c := srv.NewContext(request, recorder)
		c.Set("logger", zap.NewNop())

		err := hardTimeoutMiddleware(tc.timeout)(tc.next)(c)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		var isHardTimeout bool
		if err != nil {
			isHardTimeout = strings.Contains(err.Error(), "hard timeout")
		}

		if tc.expectHardTimeout && !isHardTimeout {
			t.Errorf("test %d: expected hard timeout error but got: %v", i, err)
		}

		if !tc.expectHardTimeout && isHardTimeout {
			t.Errorf("test %d: expected no hard timeout error but got one: %v", i, err)
		}
	}
}
