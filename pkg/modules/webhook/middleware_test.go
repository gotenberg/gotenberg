package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func TestWebhookMiddlewareGuards(t *testing.T) {
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

	buildWebhookModule := func() Webhook {
		return Webhook{
			allowList:      regexp.MustCompile(""),
			denyList:       regexp.MustCompile(""),
			errorAllowList: regexp.MustCompile(""),
			errorDenyList:  regexp.MustCompile(""),
			maxRetry:       0,
			retryMinWait:   0,
			retryMaxWait:   0,
			disable:        false,
		}
	}

	for i, tc := range []struct {
		request          *http.Request
		mod              Webhook
		next             echo.HandlerFunc
		expectErr        bool
		expectHTTPErr    bool
		expectHTTPStatus int
	}{
		{
			request: buildMultipartFormDataRequest(),
			mod:     buildWebhookModule(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return nil
				}
			}(),
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")

				return req
			}(),
			mod:              buildWebhookModule(),
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
			mod: func() Webhook {
				mod := buildWebhookModule()
				mod.allowList = regexp.MustCompile("bar")

				return mod
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
			mod: func() Webhook {
				mod := buildWebhookModule()
				mod.denyList = regexp.MustCompile("foo")

				return mod
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
			mod: func() Webhook {
				mod := buildWebhookModule()
				mod.errorAllowList = regexp.MustCompile("foo")

				return mod
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
			mod: func() Webhook {
				mod := buildWebhookModule()
				mod.errorDenyList = regexp.MustCompile("bar")

				return mod
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
			mod:              buildWebhookModule(),
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
			mod:              buildWebhookModule(),
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
			mod:              buildWebhookModule(),
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
			mod:              buildWebhookModule(),
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
			mod:              buildWebhookModule(),
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
			mod:              buildWebhookModule(),
			expectErr:        true,
			expectHTTPErr:    true,
			expectHTTPStatus: http.StatusBadRequest,
		},
	} {
		srv := echo.New()
		srv.HideBanner = true
		srv.HidePort = true

		c := srv.NewContext(tc.request, httptest.NewRecorder())

		ctx := &api.ContextMock{Context: &api.Context{}}
		ctx.SetEchoContext(c)

		c.Set("context", ctx.Context)
		c.Set("cancel", func() context.CancelFunc {
			return func() {
				return
			}
		}())

		err := webhookMiddleware(tc.mod).Handler(tc.next)(c)

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}

		var httpErr api.HTTPError
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
	}
}

func TestWebhookMiddlewareAsynchronousProcess(t *testing.T) {
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

	buildWebhookModule := func() Webhook {
		return Webhook{
			allowList:      regexp.MustCompile(""),
			denyList:       regexp.MustCompile(""),
			errorAllowList: regexp.MustCompile(""),
			errorDenyList:  regexp.MustCompile(""),
			maxRetry:       0,
			retryMinWait:   0,
			retryMaxWait:   0,
			clientTimeout:  time.Duration(30) * time.Second,
			disable:        false,
		}
	}

	for i, tc := range []struct {
		request                       *http.Request
		mod                           Webhook
		next                          echo.HandlerFunc
		expectWebhookContentType      string
		expectWebhookMethod           string
		expectWebhookExtraHTTPHeaders map[string]string
		expectWebhookFilename         string
		expectWebhookErrorStatus      int
		expectWebhookErrorMessage     string
		returnedError                 *echo.HTTPError
	}{
		{
			request: buildMultipartFormDataRequest(),
			mod:     buildWebhookModule(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return errors.New("foo")
				}
			}(),
			expectWebhookContentType:  echo.MIMEApplicationJSONCharsetUTF8,
			expectWebhookMethod:       http.MethodPost,
			expectWebhookErrorStatus:  http.StatusInternalServerError,
			expectWebhookErrorMessage: http.StatusText(http.StatusInternalServerError),
		},
		{
			request: buildMultipartFormDataRequest(),
			mod:     buildWebhookModule(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return api.NewSentinelHTTPError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
				}
			}(),
			expectWebhookContentType:  echo.MIMEApplicationJSONCharsetUTF8,
			expectWebhookMethod:       http.MethodPost,
			expectWebhookErrorStatus:  http.StatusBadRequest,
			expectWebhookErrorMessage: http.StatusText(http.StatusBadRequest),
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Output-Filename", "foo")
				req.Header.Set("Gotenberg-Webhook-Extra-Http-Headers", `{ "foo": "bar" }`)

				return req
			}(),
			mod: buildWebhookModule(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					ctx := c.Get("context").(*api.Context)

					return ctx.AddOutputPaths("/tests/test/testdata/api/sample2.pdf")
				}
			}(),
			expectWebhookContentType:      "application/pdf",
			expectWebhookMethod:           http.MethodPost,
			expectWebhookFilename:         "foo",
			expectWebhookExtraHTTPHeaders: map[string]string{"foo": "bar"},
		},
		{
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Output-Filename", "foo")

				return req
			}(),
			mod: buildWebhookModule(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					ctx := c.Get("context").(*api.Context)

					return ctx.AddOutputPaths("/tests/test/testdata/api/sample1.pdf")
				}
			}(),
			returnedError:             echo.ErrInternalServerError,
			expectWebhookContentType:  echo.MIMEApplicationJSONCharsetUTF8,
			expectWebhookMethod:       http.MethodPost,
			expectWebhookErrorStatus:  http.StatusInternalServerError,
			expectWebhookErrorMessage: http.StatusText(http.StatusInternalServerError),
			expectWebhookFilename:     "foo",
		},
	} {
		func() {
			srv := echo.New()
			srv.HideBanner = true
			srv.HidePort = true

			c := srv.NewContext(tc.request, httptest.NewRecorder())
			c.Set("logger", zap.NewNop())
			c.Set("traceHeader", "Gotenberg-Trace")
			c.Set("trace", "foo")
			c.Set("startTime", time.Now())

			ctx := &api.ContextMock{Context: &api.Context{}}
			ctx.SetLogger(zap.NewNop())
			ctx.SetEchoContext(c)

			c.Set("context", ctx.Context)
			c.Set("cancel", func() context.CancelFunc {
				return func() {
					return
				}
			}())

			webhook := echo.New()
			webhook.HideBanner = true
			webhook.HidePort = true

			rand.Seed(time.Now().UnixNano())
			webhookPort := rand.Intn(65535-1025+1) + 1025

			c.Request().Header.Set("Gotenberg-Webhook-Url", fmt.Sprintf("http://localhost:%d/", webhookPort))
			c.Request().Header.Set("Gotenberg-Webhook-Error-Url", fmt.Sprintf("http://localhost:%d/", webhookPort))

			errChan := make(chan error, 1)

			webhook.POST(
				"/",
				func() echo.HandlerFunc {
					return func(c echo.Context) error {
						contentType := c.Request().Header.Get(echo.HeaderContentType)
						if contentType != tc.expectWebhookContentType {
							t.Errorf("test %d: expected '%s' '%s' but got '%s'", i, echo.HeaderContentType, tc.expectWebhookContentType, contentType)
						}

						trace := c.Request().Header.Get("Gotenberg-Trace")
						if trace != "foo" {
							t.Errorf("test %d: expected '%s' '%s' but got '%s'", i, "Gotenberg-Trace", "foo", trace)
						}

						method := c.Request().Method
						if method != tc.expectWebhookMethod {
							t.Errorf("test %d: expected HTTP method '%s' but got '%s'", i, tc.expectWebhookMethod, method)
						}

						for key, expect := range tc.expectWebhookExtraHTTPHeaders {
							actual := c.Request().Header.Get(key)

							if actual != expect {
								t.Errorf("test %d: expected '%s' '%s' but got '%s'", i, key, expect, actual)
							}
						}

						if contentType == echo.MIMEApplicationJSONCharsetUTF8 {
							body, err := io.ReadAll(c.Request().Body)
							if err != nil {
								errChan <- err
								return nil
							}

							result := struct {
								Status  int    `json:"status"`
								Message string `json:"message"`
							}{}

							err = json.Unmarshal(body, &result)
							if err != nil {
								errChan <- err
								return nil
							}

							if result.Status != tc.expectWebhookErrorStatus {
								t.Errorf("test %d: expected status %d from JSON but got %d", i, tc.expectWebhookErrorStatus, result.Status)
							}

							if result.Message != tc.expectWebhookErrorMessage {
								t.Errorf("test %d: expected message '%s' from JSON but got '%s'", i, tc.expectWebhookErrorMessage, result.Message)
							}

							errChan <- nil
							return nil
						}

						contentLength := c.Request().Header.Get(echo.HeaderContentLength)
						if contentLength == "" {
							t.Errorf("test %d: expected non empty '%s'", i, echo.HeaderContentLength)
						}

						contentDisposition := c.Request().Header.Get(echo.HeaderContentDisposition)
						if !strings.Contains(contentDisposition, tc.expectWebhookFilename) {
							t.Errorf("test %d: expected '%s' '%s' to contain '%s'", i, echo.HeaderContentDisposition, contentDisposition, tc.expectWebhookFilename)
						}

						body, err := io.ReadAll(c.Request().Body)
						if err != nil {
							errChan <- err
							return nil
						}

						if body == nil || len(body) == 0 {
							t.Errorf("test %d: expected non nil body", i)
						}

						errChan <- nil

						if tc.returnedError != nil {
							return tc.returnedError
						}

						return nil
					}
				}(),
			)

			go func() {
				err := webhook.Start(fmt.Sprintf(":%d", webhookPort))
				if !errors.Is(err, http.ErrServerClosed) {
					t.Errorf("test %d: expected no error but got: %v", i, err)
				}
			}()

			defer func() {
				err := webhook.Shutdown(context.TODO())
				if err != nil {
					t.Errorf("test %d: expected no error but got: %v", i, err)
				}
			}()

			err := webhookMiddleware(tc.mod).Handler(tc.next)(c)
			if err != nil && err != api.ErrAsyncProcess {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}

			err = <-errChan
			if err != nil {
				t.Errorf("test %d: expected no error but got: %v", i, err)
			}
		}()
	}

}
