package webhook

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2/v2"
	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// newWebhookContext returns an echo context carrying the values the webhook
// middleware reads, for a request with the given headers.
func newWebhookContext(t *testing.T, headers map[string]string) (*echo.Context, *api.ContextMock, context.CancelFunc) {
	t.Helper()

	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	ctx := &api.ContextMock{Context: &api.Context{Context: reqCtx}}
	ctx.SetLogger(slog.New(slog.DiscardHandler))
	ctx.SetDirPath(t.TempDir())

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	c := echo.New().NewContext(req, httptest.NewRecorder())
	c.Set("logger", slog.New(slog.DiscardHandler))
	c.Set("context", ctx.Context)
	c.Set("cancel", cancel)
	c.Set("startTime", time.Now())
	c.Set("correlationIdHeader", "Gotenberg-Trace")
	c.Set("correlationId", "abc-123")

	return c, ctx, cancel
}

func TestWebhookMiddleware_AsyncDeliversTheOutput(t *testing.T) {
	type delivery struct {
		method  string
		headers http.Header
		body    string
	}
	deliveries := make(chan delivery, 1)
	events := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.URL.Path == "/events" {
			events <- string(body)
		} else {
			deliveries <- delivery{method: r.Method, headers: r.Header.Clone(), body: string(body)}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, ctx, cancel := newWebhookContext(t, map[string]string{
		"Gotenberg-Webhook-Url":                srv.URL + "/webhook",
		"Gotenberg-Webhook-Events-Url":         srv.URL + "/events",
		"Gotenberg-Webhook-Method":             "put",
		"Gotenberg-Webhook-Extra-Http-Headers": `{"X-Foo":"bar"}`,
	})
	defer cancel()

	const pdf = "%PDF-1.7\nfake document"
	next := func(_ *echo.Context) error {
		outputPath := filepath.Join(ctx.DirPath(), "out.pdf")
		err := os.WriteFile(outputPath, []byte(pdf), 0o600)
		if err != nil {
			return err
		}
		ctx.SetOutputHeader("Gotenberg-Remediation-Residual", "2")
		return ctx.AddOutputPaths(outputPath)
	}

	w := &Webhook{clientTimeout: 5 * time.Second}
	err := webhookMiddleware(w).Handler(next)(c)
	if !errors.Is(err, api.ErrAsyncProcess) {
		t.Fatalf("expected api.ErrAsyncProcess, got: %v", err)
	}

	select {
	case d := <-deliveries:
		if d.method != http.MethodPut {
			t.Errorf("method = %s, want PUT", d.method)
		}
		if d.body != pdf {
			t.Errorf("body = %q, want the output file", d.body)
		}
		for key, want := range map[string]string{
			"Content-Type":    "application/pdf",
			"X-Foo":           "bar",
			"Gotenberg-Trace": "abc-123",
			// Set by the route, such as a remediation outcome.
			"Gotenberg-Remediation-Residual": "2",
		} {
			if got := d.headers.Get(key); got != want {
				t.Errorf("header %s = %q, want %q", key, got, want)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the output never reached the webhook URL")
	}

	select {
	case event := <-events:
		if !strings.Contains(event, `"webhook.success"`) {
			t.Errorf("event = %s, want a webhook.success", event)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no event reached the events URL")
	}
}

func TestWebhookMiddleware_RejectsInvalidRequests(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		headers       map[string]string
		denyList      string
		expectStatus  int
		expectMessage string
	}{
		{
			scenario:      "missing events URL",
			headers:       map[string]string{"Gotenberg-Webhook-Url": "http://127.0.0.1/webhook"},
			expectStatus:  http.StatusBadRequest,
			expectMessage: "'Gotenberg-Webhook-Events-Url'",
		},
		{
			scenario: "unsupported method",
			headers: map[string]string{
				"Gotenberg-Webhook-Url":        "http://127.0.0.1/webhook",
				"Gotenberg-Webhook-Events-Url": "http://127.0.0.1/events",
				"Gotenberg-Webhook-Method":     "get",
			},
			expectStatus:  http.StatusBadRequest,
			expectMessage: "'Gotenberg-Webhook-Method'",
		},
		{
			scenario: "malformed extra HTTP headers",
			headers: map[string]string{
				"Gotenberg-Webhook-Url":                "http://127.0.0.1/webhook",
				"Gotenberg-Webhook-Events-Url":         "http://127.0.0.1/events",
				"Gotenberg-Webhook-Extra-Http-Headers": `{"X-Foo":`,
			},
			expectStatus:  http.StatusBadRequest,
			expectMessage: "'Gotenberg-Webhook-Extra-Http-Headers'",
		},
		{
			// A filtered URL stays generic for the client.
			scenario: "denied webhook URL",
			headers: map[string]string{
				"Gotenberg-Webhook-Url":        "http://127.0.0.1/webhook",
				"Gotenberg-Webhook-Events-Url": "http://127.0.0.1/events",
			},
			denyList:     `^http://127\.0\.0\.1/webhook`,
			expectStatus: http.StatusForbidden,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			c, _, cancel := newWebhookContext(t, tc.headers)
			defer cancel()

			w := &Webhook{clientTimeout: time.Second}
			if tc.denyList != "" {
				w.denyList = []*regexp2.Regexp{regexp2.MustCompile(tc.denyList, regexp2.None)}
			}

			next := func(*echo.Context) error {
				t.Fatal("the route ran for an invalid webhook request")
				return nil
			}

			status, message := api.ParseError(webhookMiddleware(w).Handler(next)(c))
			if status != tc.expectStatus {
				t.Fatalf("status = %d, want %d (%s)", status, tc.expectStatus, message)
			}
			if !strings.Contains(message, tc.expectMessage) {
				t.Errorf("message = %q, want it to name %s", message, tc.expectMessage)
			}
		})
	}
}

func TestWebhook_Validate(t *testing.T) {
	err := (&Webhook{}).Validate()
	if err != nil {
		t.Fatalf("expected no error without the environment proxy, got: %v", err)
	}

	t.Setenv("HTTPS_PROXY", "://%zz")

	err = (&Webhook{enableEnvironmentProxy: true}).Validate()
	if err == nil || !strings.Contains(err.Error(), "--webhook-enable-environment-proxy") || !strings.Contains(err.Error(), "HTTPS_PROXY") {
		t.Fatalf("expected an error naming the flag and HTTPS_PROXY, got: %v", err)
	}
}
