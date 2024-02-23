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
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
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

	buildWebhookModule := func() *Webhook {
		return &Webhook{
			allowList:      regexp2.MustCompile("", 0),
			denyList:       regexp2.MustCompile("", 0),
			errorAllowList: regexp2.MustCompile("", 0),
			errorDenyList:  regexp2.MustCompile("", 0),
			maxRetry:       0,
			retryMinWait:   0,
			retryMaxWait:   0,
			disable:        false,
		}
	}

	for _, tc := range []struct {
		scenario         string
		request          *http.Request
		mod              *Webhook
		next             echo.HandlerFunc
		noDeadline       bool
		expectError      bool
		expectHttpError  bool
		expectHttpStatus int
	}{
		{
			scenario: "no webhook URL, skip middleware",
			request:  buildMultipartFormDataRequest(),
			mod:      buildWebhookModule(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return nil
				}
			}(),
			noDeadline:      false,
			expectError:     false,
			expectHttpError: false,
		},
		{
			scenario: "no webhook error URL",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				return req
			}(),
			mod:              buildWebhookModule(),
			noDeadline:       false,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "context has no deadline",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				return req
			}(),
			mod:         buildWebhookModule(),
			noDeadline:  true,
			expectError: true,
		},
		{
			scenario: "webhook URL is not allowed",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				return req
			}(),
			mod: func() *Webhook {
				mod := buildWebhookModule()
				mod.allowList = regexp2.MustCompile("bar", 0)
				return mod
			}(),
			noDeadline:  false,
			expectError: true,
		},
		{
			scenario: "webhook URL is denied",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				return req
			}(),
			mod: func() *Webhook {
				mod := buildWebhookModule()
				mod.denyList = regexp2.MustCompile("foo", 0)
				return mod
			}(),
			noDeadline:  false,
			expectError: true,
		},
		{
			scenario: "webhook error URL is not allowed",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				return req
			}(),
			mod: func() *Webhook {
				mod := buildWebhookModule()
				mod.errorAllowList = regexp2.MustCompile("foo", 0)
				return mod
			}(),
			noDeadline:  false,
			expectError: true,
		},
		{
			scenario: "webhook error URL is denied",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				return req
			}(),
			mod: func() *Webhook {
				mod := buildWebhookModule()
				mod.errorDenyList = regexp2.MustCompile("bar", 0)
				return mod
			}(),
			noDeadline:  false,
			expectError: true,
		},
		{
			scenario: "invalid webhook method (GET)",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodGet)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				return req
			}(),
			mod:              buildWebhookModule(),
			noDeadline:       false,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid webhook error method (GET)",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)
				return req
			}(),
			mod:              buildWebhookModule(),
			noDeadline:       false,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "valid webhook method (POST) but invalid webhook error method (GET)",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodPost)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)
				return req
			}(),
			mod:              buildWebhookModule(),
			noDeadline:       false,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "valid webhook method (PATH) but invalid webhook error method (GET)",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodPatch)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)
				return req
			}(),
			mod:              buildWebhookModule(),
			noDeadline:       false,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "valid webhook method (PUT) but invalid webhook error method (GET)",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Method", http.MethodPut)
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Error-Method", http.MethodGet)
				return req
			}(),
			mod:              buildWebhookModule(),
			noDeadline:       false,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
		{
			scenario: "invalid webhook extra HTTP headers",
			request: func() *http.Request {
				req := buildMultipartFormDataRequest()
				req.Header.Set("Gotenberg-Webhook-Url", "foo")
				req.Header.Set("Gotenberg-Webhook-Error-Url", "bar")
				req.Header.Set("Gotenberg-Webhook-Extra-Http-Headers", "foo")
				return req
			}(),
			mod:              buildWebhookModule(),
			noDeadline:       false,
			expectError:      true,
			expectHttpError:  true,
			expectHttpStatus: http.StatusBadRequest,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			srv := echo.New()
			srv.HideBanner = true
			srv.HidePort = true

			c := srv.NewContext(tc.request, httptest.NewRecorder())

			if tc.noDeadline {
				ctx := &api.ContextMock{Context: &api.Context{Context: context.Background()}}
				ctx.SetEchoContext(c)
				c.Set("context", ctx.Context)
				c.Set("cancel", func() context.CancelFunc {
					return nil
				}())
			} else {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
				ctx := &api.ContextMock{Context: &api.Context{Context: timeoutCtx}}
				ctx.SetEchoContext(c)
				c.Set("context", ctx.Context)
				c.Set("cancel", cancel)
			}

			err := webhookMiddleware(tc.mod).Handler(tc.next)(c)

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none", err)
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			var httpErr api.HttpError
			isHttpError := errors.As(err, &httpErr)

			if tc.expectHttpError && !isHttpError {
				t.Errorf("expected an HTTP error but got: %v", err)
			}

			if !tc.expectHttpError && isHttpError {
				t.Errorf("expected no HTTP error but got one: %v", httpErr)
			}

			if err != nil && tc.expectHttpError && isHttpError {
				status, _ := httpErr.HttpError()
				if status != tc.expectHttpStatus {
					t.Errorf("expected %d as HTTP status code but got %d", tc.expectHttpStatus, status)
				}
			}
		})
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

	buildWebhookModule := func() *Webhook {
		return &Webhook{
			allowList:      regexp2.MustCompile("", 0),
			denyList:       regexp2.MustCompile("", 0),
			errorAllowList: regexp2.MustCompile("", 0),
			errorDenyList:  regexp2.MustCompile("", 0),
			maxRetry:       0,
			retryMinWait:   0,
			retryMaxWait:   0,
			clientTimeout:  time.Duration(30) * time.Second,
			disable:        false,
		}
	}

	for _, tc := range []struct {
		scenario                      string
		request                       *http.Request
		mod                           *Webhook
		next                          echo.HandlerFunc
		expectWebhookContentType      string
		expectWebhookMethod           string
		expectWebhookExtraHttpHeaders map[string]string
		expectWebhookFilename         string
		expectWebhookErrorStatus      int
		expectWebhookErrorMessage     string
		returnedError                 *echo.HTTPError
	}{
		{
			scenario: "next handler return an error",
			request:  buildMultipartFormDataRequest(),
			mod:      buildWebhookModule(),
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
			scenario: "next handler return an HTTP error",
			request:  buildMultipartFormDataRequest(),
			mod:      buildWebhookModule(),
			next: func() echo.HandlerFunc {
				return func(c echo.Context) error {
					return api.NewSentinelHttpError(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
				}
			}(),
			expectWebhookContentType:  echo.MIMEApplicationJSONCharsetUTF8,
			expectWebhookMethod:       http.MethodPost,
			expectWebhookErrorStatus:  http.StatusBadRequest,
			expectWebhookErrorMessage: http.StatusText(http.StatusBadRequest),
		},
		{
			scenario: "success",
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
			expectWebhookExtraHttpHeaders: map[string]string{"foo": "bar"},
		},
		{
			scenario: "success (return an error)",
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

			timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
			ctx := &api.ContextMock{Context: &api.Context{Context: timeoutCtx}}
			ctx.SetLogger(zap.NewNop())
			ctx.SetEchoContext(c)

			c.Set("context", ctx.Context)
			c.Set("cancel", cancel)

			webhook := echo.New()
			webhook.HideBanner = true
			webhook.HidePort = true
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
							t.Errorf("expected '%s' '%s' but got '%s'", echo.HeaderContentType, tc.expectWebhookContentType, contentType)
						}

						trace := c.Request().Header.Get("Gotenberg-Trace")
						if trace != "foo" {
							t.Errorf("expected '%s' '%s' but got '%s'", "Gotenberg-Trace", "foo", trace)
						}

						method := c.Request().Method
						if method != tc.expectWebhookMethod {
							t.Errorf("expected HTTP method '%s' but got '%s'", tc.expectWebhookMethod, method)
						}

						for key, expect := range tc.expectWebhookExtraHttpHeaders {
							actual := c.Request().Header.Get(key)

							if actual != expect {
								t.Errorf("expected '%s' '%s' but got '%s'", key, expect, actual)
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
								t.Errorf("expected status %d from JSON but got %d", tc.expectWebhookErrorStatus, result.Status)
							}

							if result.Message != tc.expectWebhookErrorMessage {
								t.Errorf("expected message '%s' from JSON but got '%s'", tc.expectWebhookErrorMessage, result.Message)
							}

							errChan <- nil
							return nil
						}

						contentLength := c.Request().Header.Get(echo.HeaderContentLength)
						if contentLength == "" {
							t.Errorf("expected non empty '%s'", echo.HeaderContentLength)
						}

						contentDisposition := c.Request().Header.Get(echo.HeaderContentDisposition)
						if !strings.Contains(contentDisposition, tc.expectWebhookFilename) {
							t.Errorf("expected '%s' '%s' to contain '%s'", echo.HeaderContentDisposition, contentDisposition, tc.expectWebhookFilename)
						}

						body, err := io.ReadAll(c.Request().Body)
						if err != nil {
							errChan <- err
							return nil
						}

						if body == nil || len(body) == 0 {
							t.Error("expected non nil body")
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
					t.Errorf("expected no error but got: %v", err)
				}
			}()

			defer func() {
				err := webhook.Shutdown(context.TODO())
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}()

			err := webhookMiddleware(tc.mod).Handler(tc.next)(c)
			if err != nil && !errors.Is(err, api.ErrAsyncProcess) {
				t.Errorf("expected no error but got: %v", err)
			}

			err = <-errChan
			if err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		}()
	}
}
