package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func useTraceContextPropagator(t *testing.T) {
	t.Helper()
	previous := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previous) })
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTracingTransport_Response(t *testing.T) {
	for _, tc := range []struct {
		name          string
		status        int
		wantCode      codes.Code
		wantErrorType string
	}{
		{name: "success", status: http.StatusOK, wantCode: codes.Unset},
		{name: "client error", status: http.StatusNotFound, wantCode: codes.Error, wantErrorType: "404"},
		{name: "server error", status: http.StatusInternalServerError, wantCode: codes.Error, wantErrorType: "500"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := newTestSpanRecorder(t)
			useTraceContextPropagator(t)

			var traceparent string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				traceparent = r.Header.Get("traceparent")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("body"))
			}))
			t.Cleanup(srv.Close)

			ctx, parent := Tracer().Start(context.Background(), "parent")
			client := &http.Client{Transport: NewTracingTransport(srv.Client().Transport)}

			target := srv.URL + "/hook?token=secret#fragment"
			target = "http://user:pass@" + target[len("http://"):]
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			if req.Header.Get("traceparent") != "" {
				t.Error("the transport modified the caller's request")
			}

			// The span covers the transfer of the body.
			if findSpan(recorder, http.MethodPost) != nil {
				t.Error("the span ended before the body was closed")
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			parent.End()

			span := findSpan(recorder, http.MethodPost)
			if span == nil {
				t.Fatal("expected a POST span")
			}
			if span.SpanKind() != trace.SpanKindClient {
				t.Errorf("span kind = %v, want client", span.SpanKind())
			}
			if span.Parent().SpanID() != parent.SpanContext().SpanID() {
				t.Errorf("span parent = %s, want %s", span.Parent().SpanID(), parent.SpanContext().SpanID())
			}

			// The receiver sees the attempt span as its parent.
			want := fmt.Sprintf("00-%s-%s-01", span.SpanContext().TraceID(), span.SpanContext().SpanID())
			if traceparent != want {
				t.Errorf("traceparent = %q, want %q", traceparent, want)
			}

			u, _ := url.Parse(srv.URL)
			port, _ := strconv.Atoi(u.Port())
			for key, want := range map[string]any{
				"http.request.method":       http.MethodPost,
				"server.address":            u.Hostname(),
				"server.port":               int64(port),
				"url.full":                  srv.URL + "/hook",
				"http.response.status_code": int64(tc.status),
			} {
				got, ok := spanAttr(span, key)
				if !ok || got.AsInterface() != want {
					t.Errorf("%s = %v, want %v", key, got.AsInterface(), want)
				}
			}

			got, ok := spanAttr(span, "error.type")
			if tc.wantErrorType == "" && ok {
				t.Errorf("error.type = %q, want none", got.AsString())
			}
			if tc.wantErrorType != "" && got.AsString() != tc.wantErrorType {
				t.Errorf("error.type = %q, want %q", got.AsString(), tc.wantErrorType)
			}
			if span.Status().Code != tc.wantCode {
				t.Errorf("span status = %v, want %v", span.Status().Code, tc.wantCode)
			}
		})
	}
}

func TestTracingTransport_TransportError(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	errDial := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	client := &http.Client{Transport: NewTracingTransport(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errDial
	}))}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/file.pdf?sig=secret", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected an error")
	}

	span := findSpan(recorder, http.MethodGet)
	if span == nil {
		t.Fatal("expected a GET span")
	}
	if span.Status().Code != codes.Error || span.Status().Description != errDial.Error() {
		t.Errorf("span status = %v, want error %q", span.Status(), errDial.Error())
	}
	if got, _ := spanAttr(span, "error.type"); got.AsString() != "net.OpError" {
		t.Errorf("error.type = %q, want net.OpError", got.AsString())
	}
	if got, _ := spanAttr(span, "url.full"); got.AsString() != "https://example.com/file.pdf" {
		t.Errorf("url.full = %q, want the redacted URL", got.AsString())
	}
	if _, ok := spanAttr(span, "http.response.status_code"); ok {
		t.Error("expected no http.response.status_code without a response")
	}
}

// TestTracingTransport_Attempts pins one client span per attempt of a
// retried delivery, under the span of the delivery.
func TestTracingTransport_Attempts(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	var (
		mu       sync.Mutex
		attempts int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	httpClient := srv.Client()
	httpClient.Transport = NewTracingTransport(httpClient.Transport)
	client := &retryablehttp.Client{
		HTTPClient:   httpClient,
		RetryMax:     1,
		RetryWaitMin: time.Millisecond,
		RetryWaitMax: time.Millisecond,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}

	ctx, delivery := Tracer().Start(context.Background(), "delivery")
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	_ = resp.Body.Close()
	delivery.End()

	var statuses []int64
	for _, span := range recorder.Ended() {
		if span.Name() != http.MethodGet {
			continue
		}
		if span.Parent().SpanID() != delivery.SpanContext().SpanID() {
			t.Errorf("attempt parent = %s, want the delivery span", span.Parent().SpanID())
		}
		status, _ := spanAttr(span, "http.response.status_code")
		statuses = append(statuses, status.AsInt64())
	}
	if len(statuses) != 2 || statuses[0] != http.StatusServiceUnavailable || statuses[1] != http.StatusOK {
		t.Errorf("attempt statuses = %v, want [503 200]", statuses)
	}
}

func TestClientSpanName(t *testing.T) {
	for method, want := range map[string]string{
		"":               http.MethodGet,
		http.MethodGet:   http.MethodGet,
		http.MethodPost:  http.MethodPost,
		http.MethodPatch: http.MethodPatch,
		"PROPFIND":       "HTTP",
		"post":           "HTTP",
	} {
		if got := clientSpanName(method); got != want {
			t.Errorf("clientSpanName(%q) = %q, want %q", method, got, want)
		}
	}
}
