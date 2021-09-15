package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestHttpErrorHandler(t *testing.T) {
	for i, tc := range []struct {
		err           error
		webhookClient *webhookClient
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
				NewSentinelHTTPError(http.StatusBadRequest, "foo"),
			),
			expectStatus:  http.StatusBadRequest,
			expectMessage: "foo",
		},
		{
			err: echo.ErrInternalServerError,
			webhookClient: &webhookClient{
				errorURL:    "http://localhost:%d/",
				errorMethod: http.MethodPost,
				client:      retryablehttp.NewClient(),
				logger:      zap.NewNop(),
			},
			expectStatus:  http.StatusInternalServerError,
			expectMessage: http.StatusText(http.StatusInternalServerError),
		},
		{
			err: echo.ErrInternalServerError,
			webhookClient: &webhookClient{
				errorURL:    "non-existent",
				errorMethod: http.MethodPost,
				client: func() *retryablehttp.Client {
					client := retryablehttp.NewClient()
					client.RetryMax = 0

					return client
				}(),
				logger: zap.NewNop(),
			},
		},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/foo", nil)

		srv := echo.New()
		srv.HideBanner = true
		srv.HidePort = true

		c := srv.NewContext(request, recorder)
		c.Set("logger", zap.NewNop())
		c.Set("trace", "foo")

		if tc.webhookClient != nil {
			c.Set("webhookClient", tc.webhookClient)
		}

		if tc.webhookClient == nil {
			handler := httpErrorHandler("Gotenberg-Trace")
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

			continue
		}

		func() {
			rand.Seed(time.Now().UnixNano())
			webhookPort := rand.Intn(65535-1025+1) + 1025

			tc.webhookClient.errorURL = fmt.Sprintf(tc.webhookClient.errorURL, webhookPort)

			c.Set("webhookClient", tc.webhookClient)

			webhook := echo.New()
			webhook.HideBanner = true
			webhook.HidePort = true

			webhook.POST(
				"/",
				func() echo.HandlerFunc {
					return func(c echo.Context) error {
						contentType := c.Request().Header.Get(echo.HeaderContentType)
						if contentType != echo.MIMEApplicationJSONCharsetUTF8 {
							t.Errorf("test %d: expected %s '%s' but got '%s'", i, echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8, contentType)
						}

						trace := c.Request().Header.Get("Gotenberg-Trace")
						if trace != "foo" {
							t.Errorf("test %d: expected %s '%s' but got '%s'", i, "Gotenberg-Trace", "foo", trace)
						}

						body, err := ioutil.ReadAll(c.Request().Body)
						if err != nil {
							t.Fatalf("test %d: expected not error but got: %v", i, err)
						}

						result := struct {
							Status  int    `json:"status"`
							Message string `json:"message"`
						}{}

						err = json.Unmarshal(body, &result)
						if err != nil {
							t.Fatalf("test %d: expected not error but got: %v", i, err)
						}

						if result.Status != tc.expectStatus {
							t.Errorf("test %d: expected status %d from JSON but got %d", i, tc.expectStatus, result.Status)
						}

						if result.Message != tc.expectMessage {
							t.Errorf("test %d: expected message '%s' from JSON but got '%s'", i, tc.expectMessage, result.Message)
						}

						return nil
					}
				}(),
			)

			go func(server *echo.Echo, port, i int) {
				err := webhook.Start(fmt.Sprintf(":%d", port))
				if !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("test %d: expected no error but got: %v", i, err)
				}
			}(webhook, webhookPort, i)

			defer func() {
				err := webhook.Shutdown(context.TODO())
				if err != nil {
					t.Errorf("test %d: expected no error but got: %v", i, err)
				}
			}()

			handler := httpErrorHandler("Gotenberg-Trace")
			handler(tc.err, c)
		}()
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
		request                *http.Request
		next                   echo.HandlerFunc
		skipHealthRouteLogging bool
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
			skipHealthRouteLogging: true,
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

		err := loggerMiddleware(zap.NewNop(), tc.skipHealthRouteLogging)(tc.next)(c)

		if err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestContextMiddlewareWithoutWebhook(t *testing.T) {
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

		cfg := contextMiddlewareConfig{
			timeout: struct {
				process time.Duration
				write   time.Duration
			}{
				process: time.Duration(10) * time.Second,
			},
		}

		err := contextMiddleware(cfg)(tc.next)(c)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		if err != nil {
			continue
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("test %d: expected HTTP status code %d but got %d", i, http.StatusOK, recorder.Code)
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

func TestContextMiddlewareWithWebhook(t *testing.T) {
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

	buildContextMiddlewareConfig := func() contextMiddlewareConfig {
		return contextMiddlewareConfig{
			traceHeader: "Gotenberg-Trace",
			timeout: struct {
				process time.Duration
				write   time.Duration
			}{
				process: time.Duration(10) * time.Second,
				write:   time.Duration(10) * time.Second,
			},
			webhook: struct {
				allowList      *regexp.Regexp
				denyList       *regexp.Regexp
				errorAllowList *regexp.Regexp
				errorDenyList  *regexp.Regexp
				maxRetry       int
				retryMinWait   time.Duration
				retryMaxWait   time.Duration
				disable        bool
			}{
				allowList:      regexp.MustCompile(""),
				denyList:       regexp.MustCompile(""),
				errorAllowList: regexp.MustCompile(""),
				errorDenyList:  regexp.MustCompile(""),
			},
		}
	}

	for i, tc := range []struct {
		request                       *http.Request
		cfg                           contextMiddlewareConfig
		next                          echo.HandlerFunc
		autoWebhookURLs               bool
		expectErr                     bool
		expectHTTPErr                 bool
		expectHTTPStatus              int
		expectWebhookContentType      string
		expectWebhookMethod           string
		expectWebhookExtraHTTPHeaders map[string]string
		expectWebhookFilename         string
	}{
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")

				return req
			}(),
			cfg: func() contextMiddlewareConfig {
				cfg := buildContextMiddlewareConfig()
				cfg.webhook.disable = true

				return cfg
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusForbidden,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")

				return req
			}(),
			cfg:              buildContextMiddlewareConfig(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")

				return req
			}(),
			cfg: func() contextMiddlewareConfig {
				cfg := buildContextMiddlewareConfig()
				cfg.webhook.allowList = regexp.MustCompile("bar")

				return cfg
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusForbidden,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")

				return req
			}(),
			cfg: func() contextMiddlewareConfig {
				cfg := buildContextMiddlewareConfig()
				cfg.webhook.denyList = regexp.MustCompile("foo")

				return cfg
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusForbidden,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")

				return req
			}(),
			cfg: func() contextMiddlewareConfig {
				cfg := buildContextMiddlewareConfig()
				cfg.webhook.errorAllowList = regexp.MustCompile("foo")

				return cfg
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusForbidden,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")

				return req
			}(),
			cfg: func() contextMiddlewareConfig {
				cfg := buildContextMiddlewareConfig()
				cfg.webhook.errorDenyList = regexp.MustCompile("bar")

				return cfg
			}(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusForbidden,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodGet)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")

				return req
			}(),
			cfg:              buildContextMiddlewareConfig(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)

				return req
			}(),
			cfg:              buildContextMiddlewareConfig(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodPost)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)

				return req
			}(),
			cfg:              buildContextMiddlewareConfig(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodPatch)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)

				return req
			}(),
			cfg:              buildContextMiddlewareConfig(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodPut)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)

				return req
			}(),
			cfg:              buildContextMiddlewareConfig(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Extra-Http-Headers", "foo")

				return req
			}(),
			cfg:              buildContextMiddlewareConfig(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
		{
			request: buildMultipartFormDataRequest(),
			cfg:     buildContextMiddlewareConfig(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return errors.New("foo")
				}
			}(),
			autoWebhookURLs:          true,
			expectWebhookContentType: echo.MIMEApplicationJSONCharsetUTF8,
			expectWebhookMethod:      http.MethodPost,
		},
		{
			request: buildMultipartFormDataRequest(),
			cfg:     buildContextMiddlewareConfig(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return nil
				}
			}(),
			autoWebhookURLs:          true,
			expectWebhookContentType: echo.MIMEApplicationJSONCharsetUTF8,
			expectWebhookMethod:      http.MethodPost,
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Output-Filename", "foo")
				req.Header.Set("Gotenberg-Webhook-Extra-Http-Headers", `{ "foo": "bar" }`)

				return req
			}(),
			cfg: buildContextMiddlewareConfig(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					ctx := c.Get("context").(*Context)
					ctx.outputPaths = []string{
						"/tests/test/testdata/api/sample2.pdf",
					}

					return nil
				}
			}(),
			autoWebhookURLs:               true,
			expectWebhookContentType:      "application/pdf",
			expectWebhookMethod:           http.MethodPost,
			expectWebhookFilename:         "foo",
			expectWebhookExtraHTTPHeaders: map[string]string{"foo": "bar"},
		},
		{
			request: buildMultipartFormDataRequest(),
			cfg:     buildContextMiddlewareConfig(),
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
			autoWebhookURLs:          true,
			expectWebhookContentType: "application/zip",
			expectWebhookMethod:      http.MethodPost,
		},
	} {
		func() {
			recorder := httptest.NewRecorder()

			srv := echo.New()
			srv.HideBanner = true
			srv.HidePort = true
			srv.HTTPErrorHandler = httpErrorHandler(tc.cfg.traceHeader)

			c := srv.NewContext(tc.request, recorder)
			c.Set("logger", zap.NewNop())
			c.Set("trace", "foo")
			c.Set("startTime", time.Now())

			webhook := echo.New()
			webhook.HideBanner = true
			webhook.HidePort = true

			rand.Seed(time.Now().UnixNano())
			webhookPort := rand.Intn(65535-1025+1) + 1025

			if tc.autoWebhookURLs {
				c.Request().Header.Set("Gotenberg-Webhook-Url", fmt.Sprintf("http://localhost:%d/", webhookPort))
				c.Request().Header.Set("Gotenberg-Webhook-Error-Url", fmt.Sprintf("http://localhost:%d/", webhookPort))
			}

			errChan := make(chan error, 1)

			webhook.POST(
				"/",
				func() echo.HandlerFunc {
					return func(c echo.Context) error {
						contentType := c.Request().Header.Get(echo.HeaderContentType)
						if contentType != tc.expectWebhookContentType {
							t.Errorf("test %d: expected %s '%s' but got '%s'", i, echo.HeaderContentType, tc.expectWebhookContentType, contentType)
						}

						trace := c.Request().Header.Get(tc.cfg.traceHeader)
						if trace != "foo" {
							t.Errorf("test %d: expected %s '%s' but got '%s'", i, "Gotenberg-Trace", "foo", trace)
						}

						method := c.Request().Method
						if method != tc.expectWebhookMethod {
							t.Errorf("test %d: expected HTTP method '%s' but got '%s'", i, tc.expectWebhookMethod, method)
						}

						for key, expect := range tc.expectWebhookExtraHTTPHeaders {
							actual := c.Request().Header.Get(key)

							if actual != expect {
								t.Errorf("test %d: expected %s '%s' but got '%s'", i, key, expect, actual)
							}
						}

						if tc.expectWebhookContentType == echo.MIMEApplicationJSONCharsetUTF8 {
							errChan <- nil
							return nil
						}

						contentLength := c.Request().Header.Get(echo.HeaderContentLength)
						if contentLength == "" {
							t.Errorf("test %d: expected non empty %s", i, echo.HeaderContentLength)
						}

						contentDisposition := c.Request().Header.Get(echo.HeaderContentDisposition)
						if !strings.Contains(contentDisposition, tc.expectWebhookFilename) {
							t.Errorf("test %d: expected %s '%s' to contain '%s'", i, echo.HeaderContentDisposition, contentDisposition, tc.expectWebhookFilename)
						}

						body, err := ioutil.ReadAll(c.Request().Body)
						if err != nil {
							errChan <- err
							return nil
						}

						if body == nil || len(body) == 0 {
							t.Errorf("test %d: expected non nil body", i)
						}

						errChan <- nil
						return nil
					}
				}(),
			)

			go func(server *echo.Echo, port, i int) {
				err := server.Start(fmt.Sprintf(":%d", webhookPort))
				if !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("test %d: expected no error but got: %v", i, err)
				}
			}(webhook, webhookPort, i)

			defer func() {
				err := webhook.Shutdown(context.TODO())
				if err != nil {
					t.Errorf("test %d: expected no error but got: %v", i, err)
				}
			}()

			err := contextMiddleware(tc.cfg)(tc.next)(c)

			if tc.expectErr && err == nil {
				t.Errorf("test %d: expected error but got: %v", i, err)
			}

			if !tc.expectErr && err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}

			var httpErr HTTPError
			isHTTPErr := errors.As(err, &httpErr)

			if tc.expectHTTPErr && !isHTTPErr {
				t.Errorf("test %d: expected HTTP error but got: %v", i, err)
			}

			if !tc.expectHTTPErr && isHTTPErr {
				t.Errorf("test %d: expected no HTTP error but got one: %v", i, httpErr)
			}

			if err != nil && tc.expectHTTPErr && isHTTPErr {
				status, _ := httpErr.HTTPError()
				if status != tc.expectHTTPStatus {
					t.Errorf("test %d: expected %d HTTP status code but got %d", i, tc.expectHTTPStatus, status)
				}
			}

			if err != nil {
				return
			}

			if recorder.Code != http.StatusNoContent {
				t.Errorf("test %d: expected HTTP status code %d but got %d", i, http.StatusNoContent, recorder.Code)
			}

			err = <-errChan
			if err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}
}

func TestTimeoutMiddleware(t *testing.T) {
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

		err := timeoutMiddleware(tc.timeout)(tc.next)(c)

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
