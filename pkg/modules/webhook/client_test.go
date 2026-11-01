package webhook

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// TestClient_SendFailure pins the event message of an upload that fails for
// good: what the webhook URL answered, or why it could not be reached, and
// after how many attempts, with the status the API answered it with before.
func TestClient_SendFailure(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failing.Close()

	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer missing.Close()

	// A listener closed at once: nothing accepts a connection on its port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	refused := "http://" + ln.Addr().String() + "/webhook"
	_ = ln.Close()

	for _, tc := range []struct {
		name string
		url  string
		want string
	}{
		{name: "error status", url: failing.URL, want: "The webhook URL answered 500 Internal Server Error after 3 attempts"},
		{name: "status not retried", url: missing.URL, want: "The webhook URL answered 404 Not Found"},
		{name: "unreachable", url: refused, want: "The webhook URL could not be reached after 3 attempts: the connection was refused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := client{
				url:             tc.url,
				method:          http.MethodPost,
				deliveryTimeout: 10 * time.Second,
				client: &retryablehttp.Client{
					HTTPClient:   http.DefaultClient,
					RetryMax:     2,
					RetryWaitMin: time.Millisecond,
					RetryWaitMax: time.Millisecond,
					CheckRetry:   retryablehttp.DefaultRetryPolicy,
					Backoff:      retryablehttp.DefaultBackoff,
					ErrorHandler: giveUp,
				},
				logger: slog.New(slog.DiscardHandler),
			}

			err := c.send(context.Background(), strings.NewReader("%PDF-1.7"), map[string]string{})

			status, message := api.ParseError(err)
			if status != http.StatusInternalServerError || message != tc.want {
				t.Fatalf("got %d %q, want 500 %q", status, message, tc.want)
			}
		})
	}
}

func TestUnreachableReason(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{name: "unresolvable host", err: &gotenberg.UnresolvableHostError{Host: "storage.example.invalid", Err: errors.New("no such host")}, want: "the host 'storage.example.invalid' does not resolve"},
		{name: "unknown host at dial", err: fmt.Errorf("dial: %w", &net.DNSError{Name: "storage.example.invalid", IsNotFound: true}), want: "the host 'storage.example.invalid' does not resolve"},
		{name: "policy refusal stays generic", err: fmt.Errorf("'http://10.0.0.1/' targets a non-public address: %w", gotenberg.ErrFiltered), want: ""},
		{name: "timeout", err: fmt.Errorf("post: %w", context.DeadlineExceeded), want: "the request timed out"},
		{name: "anything else", err: errors.New("EOF"), want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := unreachableReason(tc.err); got != tc.want {
				t.Errorf("unreachableReason() = %q, want %q", got, tc.want)
			}
		})
	}
}
