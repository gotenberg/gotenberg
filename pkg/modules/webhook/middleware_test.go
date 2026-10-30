package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

func TestWebhookMiddleware_SyncModeAnswersOnce(t *testing.T) {
	for _, tc := range []struct {
		scenario          string
		next              echo.HandlerFunc
		expectEventStatus float64
	}{
		{
			scenario:          "failed conversion",
			next:              func(_ *echo.Context) error { return errors.New("conversion failed") },
			expectEventStatus: http.StatusInternalServerError,
		},
		{
			scenario: "route that writes its own response",
			next: func(c *echo.Context) error {
				err := c.JSON(http.StatusOK, map[string]string{"foo": "bar"})
				if err != nil {
					return err
				}
				return api.ErrNoOutputFile
			},
			expectEventStatus: http.StatusBadRequest,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			events := make(chan map[string]any, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var event map[string]any
				b, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(b, &event)
				events <- event
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			ctx := &api.ContextMock{Context: &api.Context{Context: reqCtx}}
			ctx.SetLogger(slog.New(slog.DiscardHandler))

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Gotenberg-Webhook-Url", srv.URL+"/webhook")
			req.Header.Set("Gotenberg-Webhook-Events-Url", srv.URL+"/events")
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set("logger", slog.New(slog.DiscardHandler))
			c.Set("context", ctx.Context)
			c.Set("cancel", context.CancelFunc(cancel))
			c.Set("startTime", time.Now())
			c.Set("correlationIdHeader", "Gotenberg-Trace")
			c.Set("correlationId", "abc-123")

			w := &Webhook{enableSyncMode: true, clientTimeout: 5 * time.Second}
			handler := webhookMiddleware(w).Handler(tc.next)

			err := handler(c)
			if !errors.Is(err, api.ErrNoOutputFile) {
				t.Fatalf("expected api.ErrNoOutputFile so that contextMiddleware sends nothing, got: %v", err)
			}

			if rec.Code != http.StatusNoContent {
				t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
			}
			if rec.Body.Len() != 0 {
				t.Errorf("expected an empty body, got %q", rec.Body.String())
			}

			select {
			case event := <-events:
				if event["event"] != "webhook.error" {
					t.Fatalf("expected a webhook.error event, got %v", event["event"])
				}
				status := event["error"].(map[string]any)["status"]
				if status != tc.expectEventStatus {
					t.Errorf("expected event status %v, got %v", tc.expectEventStatus, status)
				}
			default:
				t.Fatal("expected an event on the events URL")
			}
		})
	}
}
