package webhook

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

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

// TestClient_Spans pins the spans of a delivery: an internal span over the
// whole delivery, with a redacted url.full, and an HTTP client span per
// attempt under it, which the receiver sees as its parent.
func TestClient_Spans(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previousPropagator) })

	var (
		mu           sync.Mutex
		traceparents []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		traceparents = append(traceparents, r.Header.Get("traceparent"))
		mu.Unlock()
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	for _, tc := range []struct {
		name          string
		path          string
		send          func(client)
		wantSpan      string
		wantAttempts  int
		wantCode      codes.Code
		wantErrorType string
	}{
		{
			name:         "send",
			path:         "/webhook",
			send:         func(c client) { _ = c.send(context.Background(), strings.NewReader("%PDF-1.7"), map[string]string{}) },
			wantSpan:     "webhook.send",
			wantAttempts: 1,
			wantCode:     codes.Unset,
		},
		{
			name:          "send failure",
			path:          "/fail",
			send:          func(c client) { _ = c.send(context.Background(), strings.NewReader("%PDF-1.7"), map[string]string{}) },
			wantSpan:      "webhook.send",
			wantAttempts:  2,
			wantCode:      codes.Error,
			wantErrorType: "500",
		},
		{
			name: "send event",
			path: "/events",
			send: func(c client) {
				c.sendEvent(context.Background(), "Gotenberg-Trace", "id", map[string]any{"event": "test"})
			},
			wantSpan:     "webhook.send_event",
			wantAttempts: 1,
			wantCode:     codes.Unset,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder.Reset()
			mu.Lock()
			traceparents = nil
			mu.Unlock()

			httpClient := &http.Client{Transport: gotenberg.NewTracingTransport(http.DefaultTransport)}
			target := srv.URL + tc.path + "?token=secret"
			tc.send(client{
				url:             target,
				method:          http.MethodPost,
				eventsUrl:       target,
				deliveryTimeout: 10 * time.Second,
				client: &retryablehttp.Client{
					HTTPClient:   httpClient,
					RetryMax:     1,
					RetryWaitMin: time.Millisecond,
					RetryWaitMax: time.Millisecond,
					CheckRetry:   retryablehttp.DefaultRetryPolicy,
					Backoff:      retryablehttp.DefaultBackoff,
					ErrorHandler: giveUp,
				},
				logger: slog.New(slog.DiscardHandler),
			})

			var (
				delivery sdktrace.ReadOnlySpan
				attempts []sdktrace.ReadOnlySpan
			)
			for _, span := range recorder.Ended() {
				switch span.Name() {
				case tc.wantSpan:
					delivery = span
				case http.MethodPost:
					attempts = append(attempts, span)
				}
			}
			if delivery == nil || len(attempts) != tc.wantAttempts {
				t.Fatalf("got delivery span %v and %d attempt spans, want %s and %d", delivery != nil, len(attempts), tc.wantSpan, tc.wantAttempts)
			}

			if delivery.SpanKind() != trace.SpanKindInternal {
				t.Errorf("%s kind = %v, want internal", tc.wantSpan, delivery.SpanKind())
			}
			wantURL := attribute.String("url.full", srv.URL+tc.path)
			if !slices.Contains(delivery.Attributes(), wantURL) {
				t.Errorf("%s attributes = %v, want %v", tc.wantSpan, delivery.Attributes(), wantURL)
			}
			if delivery.Status().Code != tc.wantCode {
				t.Errorf("%s status = %v, want %v", tc.wantSpan, delivery.Status().Code, tc.wantCode)
			}

			mu.Lock()
			defer mu.Unlock()
			for i, attempt := range attempts {
				if attempt.SpanKind() != trace.SpanKindClient || attempt.Parent().SpanID() != delivery.SpanContext().SpanID() {
					t.Errorf("attempt %d: kind %v, parent %s, want a client span under %s", i, attempt.SpanKind(), attempt.Parent().SpanID(), tc.wantSpan)
				}
				if !slices.Contains(attempt.Attributes(), wantURL) {
					t.Errorf("attempt %d attributes = %v, want %v", i, attempt.Attributes(), wantURL)
				}
				if want := fmt.Sprintf("00-%s-%s-01", attempt.SpanContext().TraceID(), attempt.SpanContext().SpanID()); !slices.Contains(traceparents, want) {
					t.Errorf("traceparents = %v, want one of attempt %d: %s", traceparents, i, want)
				}
				attrs := attribute.NewSet(attempt.Attributes()...)
				errorType, _ := attrs.Value("error.type")
				if errorType.AsString() != tc.wantErrorType || attempt.Status().Code != tc.wantCode {
					t.Errorf("attempt %d: status %v, error.type %q, want %v and %q", i, attempt.Status().Code, errorType.AsString(), tc.wantCode, tc.wantErrorType)
				}
			}
		})
	}
}
