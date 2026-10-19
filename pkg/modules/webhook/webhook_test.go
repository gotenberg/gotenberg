package webhook

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestWebhook_deliveryTimeout(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		clientTimeout time.Duration
		maxRetry      int
		retryMaxWait  time.Duration
		want          time.Duration
	}{
		{
			scenario:      "shipped defaults",
			clientTimeout: 30 * time.Second,
			maxRetry:      4,
			retryMaxWait:  30 * time.Second,
			want:          270 * time.Second,
		},
		{
			scenario:      "no retry is one client timeout",
			clientTimeout: 30 * time.Second,
			maxRetry:      0,
			retryMaxWait:  30 * time.Second,
			want:          30 * time.Second,
		},
		{
			scenario:      "a zero client timeout still gets a budget",
			clientTimeout: 0,
			maxRetry:      0,
			retryMaxWait:  0,
			want:          minDeliveryTimeout,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			w := &Webhook{
				clientTimeout: tc.clientTimeout,
				maxRetry:      tc.maxRetry,
				retryMaxWait:  tc.retryMaxWait,
			}
			if got := w.deliveryTimeout(); got != tc.want {
				t.Fatalf("deliveryTimeout() = %s, want %s", got, tc.want)
			}
			if w.deliveryTimeout() <= 0 {
				t.Fatal("deliveryTimeout() must always be positive")
			}
		})
	}
}

// A delivery must not inherit the conversion deadline. It runs after the
// handler returned, so that deadline is often already spent, which would fail
// the callback without a single attempt.
func TestClient_deliveryContext_IgnoresAnExpiredParentDeadline(t *testing.T) {
	expired, cancelExpired := context.WithTimeout(context.Background(), -1*time.Second)
	defer cancelExpired()

	if expired.Err() == nil {
		t.Fatal("expected the parent context to be expired")
	}

	c := client{deliveryTimeout: 30 * time.Second}
	ctx, cancel := c.deliveryContext(expired)
	defer cancel()

	if ctx.Err() != nil {
		t.Fatalf("delivery context inherited the expired parent: %v", ctx.Err())
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("delivery context has no deadline, so a delivery would be unbounded")
	}
	if remaining := time.Until(deadline); remaining <= 0 {
		t.Fatalf("delivery budget = %s, want a positive value", remaining)
	}
}

// The delivery context must still be bounded, so a hostile remote cannot hold
// the goroutine and its output file open indefinitely.
func TestClient_deliveryContext_IsBounded(t *testing.T) {
	c := client{deliveryTimeout: 50 * time.Millisecond}
	ctx, cancel := c.deliveryContext(context.Background())
	defer cancel()

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("delivery context never expired")
	}
}

// TestClient_VersionHeader pins the "Gotenberg-Version" header on every webhook
// delivery, and that extra HTTP headers cannot override it.
// See https://github.com/gotenberg/gotenberg/issues/1693.
func TestClient_VersionHeader(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		deliver  func(c client) error
	}{
		{
			scenario: "output file",
			deliver: func(c client) error {
				return c.send(context.Background(), strings.NewReader("%PDF-1.7"), nil, false)
			},
		},
		{
			scenario: "error details",
			deliver: func(c client) error {
				return c.send(context.Background(), strings.NewReader("{}"), nil, true)
			},
		},
		{
			scenario: "event",
			deliver: func(c client) error {
				c.sendEvent(context.Background(), "Gotenberg-Trace", "foo", map[string]any{"event": "webhook.success"})
				return nil
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			received := make(chan string, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received <- r.Header.Get("Gotenberg-Version")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			retryableClient := retryablehttp.NewClient()
			retryableClient.HTTPClient = srv.Client()
			retryableClient.RetryMax = 0
			retryableClient.Logger = nil

			c := client{
				url:              srv.URL,
				method:           http.MethodPost,
				errorUrl:         srv.URL,
				errorMethod:      http.MethodPost,
				eventsUrl:        srv.URL,
				extraHttpHeaders: map[string]string{"Gotenberg-Version": "spoofed"},
				startTime:        time.Now(),
				deliveryTimeout:  30 * time.Second,
				client:           retryableClient,
				logger:           slog.New(slog.DiscardHandler),
			}

			err := tc.deliver(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			select {
			case got := <-received:
				if got != gotenberg.Version {
					t.Errorf("Gotenberg-Version = %q, want %q", got, gotenberg.Version)
				}
			default:
				t.Fatal("the webhook server received no request")
			}
		})
	}
}
