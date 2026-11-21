package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// TestRequestCanceled pins the client-abort discriminator: only a
// context.Canceled that stems from the request context counts, so a server
// timeout or an unrelated cancellation still surfaces as an internal failure.
// See https://github.com/gotenberg/gotenberg/issues/1627.
func TestRequestCanceled(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	timedOut, cancelTimeout := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelTimeout()

	for _, tc := range []struct {
		name   string
		reqCtx context.Context
		err    error
		want   bool
	}{
		{"client abort", canceled, context.Canceled, true},
		{"wrapped client abort", canceled, fmt.Errorf("convert to PDF: %w", context.Canceled), true},
		{"canceled error but live request", context.Background(), context.Canceled, false},
		{"canceled request but unrelated error", canceled, errors.New("boom"), false},
		{"server timeout is not a client abort", timedOut, context.DeadlineExceeded, false},
		{"no error", canceled, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(tc.reqCtx)
			c := echo.New().NewContext(req, httptest.NewRecorder())
			if got := requestCanceled(c, tc.err); got != tc.want {
				t.Fatalf("requestCanceled = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestHttpErrorHandler_ClientClosedRequest ensures a client abort is recorded
// as 499 rather than 500, and that a genuine failure keeps its status.
func TestHttpErrorHandler_ClientClosedRequest(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	for _, tc := range []struct {
		name       string
		reqCtx     context.Context
		err        error
		wantStatus int
	}{
		{"client abort", canceled, fmt.Errorf("convert to PDF: %w", context.Canceled), statusClientClosedRequest},
		{"internal failure", context.Background(), errors.New("boom"), http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(tc.reqCtx)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set("logger", slog.New(slog.DiscardHandler))

			httpErrorHandler()(c, tc.err)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

// TestOutputFilenameMiddleware pins the sanitizing of the
// "Gotenberg-Output-Filename" header. The value reaches archive entry names and
// a Content-Disposition header, so a path separator must never survive it.
// See https://github.com/gotenberg/gotenberg/issues/1227 and
// GHSA-hwc4-gmrw-5222.
func TestOutputFilenameMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		want   string
	}{
		{"no header", "", ""},
		{"plain filename", "foo", "foo"},
		{"POSIX path", "/tmp/foo", "foo"},
		{"POSIX traversal", "../../../etc/passwd", "passwd"},
		{"Windows traversal", `..\..\..\..\Windows\System32\evil`, "evil"},
		{"rooted Windows path", `C:\Windows\Temp\evil`, "evil"},
		{"mixed separators", `a/b\c`, "c"},
		{"trailing separator", "/tmp/", ""},
		{"bare dot dot", "..", ".."},
		{"control characters", "fo\x01o\x7f", "foo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := outputFilenameMiddleware()(func(c *echo.Context) error { return nil })

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.header != "" {
				req.Header.Set("Gotenberg-Output-Filename", tc.header)
			}
			c := echo.New().NewContext(req, httptest.NewRecorder())

			err := handler(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, ok := c.Get("outputFilename").(string)
			if !ok {
				t.Fatal("outputFilename is not set as a string")
			}
			if got != tc.want {
				t.Errorf("outputFilename = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestVersionHeaderMiddleware pins where the "Gotenberg-Version" header shows
// up: on authenticated responses, failed ones included, and never on a request
// that fails authentication.
// See https://github.com/gotenberg/gotenberg/issues/1693.
func TestVersionHeaderMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name       string
		handler    echo.HandlerFunc
		authorized bool
		wantStatus int
		wantHeader string
	}{
		{
			name:       "success",
			handler:    func(c *echo.Context) error { return c.NoContent(http.StatusOK) },
			authorized: true,
			wantStatus: http.StatusOK,
			wantHeader: gotenberg.Version,
		},
		{
			name:       "handler error",
			handler:    func(c *echo.Context) error { return echo.NewHTTPError(http.StatusBadRequest, "bad request") },
			authorized: true,
			wantStatus: http.StatusBadRequest,
			wantHeader: gotenberg.Version,
		},
		{
			name:       "unauthorized",
			handler:    func(c *echo.Context) error { return c.NoContent(http.StatusOK) },
			authorized: false,
			wantStatus: http.StatusUnauthorized,
			wantHeader: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newEchoServer()
			srv.Pre(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error {
					c.Set("logger", slog.New(slog.DiscardHandler))
					return next(c)
				}
			})
			srv.GET("/", tc.handler, basicAuthMiddleware("foo", "bar"), versionHeaderMiddleware())

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authorized {
				req.SetBasicAuth("foo", "bar")
			}
			rec := httptest.NewRecorder()

			srv.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := rec.Header().Get("Gotenberg-Version"); got != tc.wantHeader {
				t.Errorf("Gotenberg-Version = %q, want %q", got, tc.wantHeader)
			}
		})
	}
}

// TestTelemetryMiddleware_SpanStatus pins the status of the server span, per
// the HTTP semconv: unset on a success and on a 4xx, an error typed with its
// status code on a 5xx. The status description carries the error message,
// and no attribute does, since the error.* namespace belongs to OpenTelemetry.
func TestTelemetryMiddleware_SpanStatus(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	for _, tc := range []struct {
		name            string
		handler         echo.HandlerFunc
		wantCode        codes.Code
		wantDescription string
		wantErrorType   string
	}{
		{
			name:     "success",
			handler:  func(c *echo.Context) error { return c.NoContent(http.StatusOK) },
			wantCode: codes.Unset,
		},
		{
			name:     "client error",
			handler:  func(*echo.Context) error { return NewSentinelHttpError(http.StatusBadRequest, "Invalid form data") },
			wantCode: codes.Unset,
		},
		{
			name:            "internal error",
			handler:         func(*echo.Context) error { return fmt.Errorf("convert to PDF: %w", errors.New("boom")) },
			wantCode:        codes.Error,
			wantDescription: "convert to PDF: boom",
			wantErrorType:   "500",
		},
		{
			name:            "timeout",
			handler:         func(*echo.Context) error { return fmt.Errorf("convert to PDF: %w", context.DeadlineExceeded) },
			wantCode:        codes.Error,
			wantDescription: "convert to PDF: context deadline exceeded",
			wantErrorType:   "503",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder.Reset()

			srv := newEchoServer()
			srv.Pre(
				latencyMiddleware(),
				rootPathMiddleware("/"),
				// No primary server name, as with the default --api-bind-ip
				// (env API_BIND_IP): server.address comes from the Host.
				telemetryMiddleware(slog.New(slog.DiscardHandler), "", ":3000", "Gotenberg-Trace", nil),
			)
			srv.POST("/forms/test", tc.handler)

			srv.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "http://gotenberg.example:3000/forms/test", nil))

			ended := recorder.Ended()
			if len(ended) != 1 {
				t.Fatalf("ended spans = %d, want 1", len(ended))
			}
			span := ended[0]

			if got := span.Status(); got.Code != tc.wantCode || got.Description != tc.wantDescription {
				t.Errorf("span status = %v %q, want %v %q", got.Code, got.Description, tc.wantCode, tc.wantDescription)
			}

			attrs := make(map[attribute.Key]attribute.Value)
			for _, kv := range span.Attributes() {
				if kv.Key == "error" || kv.Key == "error.message" {
					t.Errorf("span carries the %q attribute", kv.Key)
				}
				attrs[kv.Key] = kv.Value
			}
			if got := attrs["error.type"].AsString(); got != tc.wantErrorType {
				t.Errorf("error.type = %q, want %q", got, tc.wantErrorType)
			}
			if got := attrs["server.address"].AsString(); got != "gotenberg.example" {
				t.Errorf("server.address = %q, want gotenberg.example", got)
			}
			if got := attrs["server.port"].AsInt64(); got != 3000 {
				t.Errorf("server.port = %d, want 3000", got)
			}
		})
	}
}

// TestTelemetryMiddleware_Route pins the route of the server span and of the
// metrics: the route the request matched, or none, so that a path or a Host
// a client makes up never starts a new span name or time series.
func TestTelemetryMiddleware_Route(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	previousTracer := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previousTracer) })

	for _, tc := range []struct {
		name      string
		method    string
		path      string
		wantSpan  string
		wantRoute string
	}{
		{name: "matched", method: http.MethodPost, path: "/forms/test", wantSpan: "POST /forms/test", wantRoute: "/forms/test"},
		{name: "not found", method: http.MethodPost, path: "/forms/made-up", wantSpan: "POST"},
		{name: "unknown method", method: "MADEUP", path: "/forms/made-up", wantSpan: "HTTP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder.Reset()

			reader := sdkmetric.NewManualReader()
			previousMeter := otel.GetMeterProvider()
			otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
			t.Cleanup(func() { otel.SetMeterProvider(previousMeter) })

			srv := newEchoServer()
			srv.Pre(
				latencyMiddleware(),
				rootPathMiddleware("/"),
				telemetryMiddleware(slog.New(slog.DiscardHandler), "", ":3000", "Gotenberg-Trace", nil),
			)
			srv.POST("/forms/test", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

			srv.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, "http://made-up.example:3000"+tc.path, nil))

			ended := recorder.Ended()
			if len(ended) != 1 {
				t.Fatalf("ended spans = %d, want 1", len(ended))
			}
			if got := ended[0].Name(); got != tc.wantSpan {
				t.Errorf("span name = %q, want %q", got, tc.wantSpan)
			}
			spanAttrs := attribute.NewSet(ended[0].Attributes()...)
			if got, _ := spanAttrs.Value("http.route"); got.AsString() != tc.wantRoute {
				t.Errorf("span http.route = %q, want %q", got.AsString(), tc.wantRoute)
			}

			var rm metricdata.ResourceMetrics
			err := reader.Collect(context.Background(), &rm)
			if err != nil {
				t.Fatalf("collect metrics: %v", err)
			}

			var points []metricdata.HistogramDataPoint[float64]
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if hist, ok := m.Data.(metricdata.Histogram[float64]); ok && m.Name == "http.server.request.duration" {
						points = append(points, hist.DataPoints...)
					}
				}
			}
			if len(points) != 1 {
				t.Fatalf("http.server.request.duration points = %d, want 1", len(points))
			}
			if got, _ := points[0].Attributes.Value("http.route"); got.AsString() != tc.wantRoute {
				t.Errorf("metric http.route = %q, want %q", got.AsString(), tc.wantRoute)
			}
			if got, _ := points[0].Attributes.Value("server.address"); got.AsString() != "" {
				t.Errorf("metric server.address = %q, want none from the Host", got.AsString())
			}
		})
	}
}

// newMultipartServer returns a server whose route /forms/test runs handler as
// the API runs a multipart route, under the given hard timeout, and whose route
// /other answers "other" after a while, to keep a request on a recycled
// [echo.Context] busy.
func newMultipartServer(t *testing.T, hardTimeout time.Duration, handler echo.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := newEchoServer()
	srv.Pre(
		latencyMiddleware(),
		rootPathMiddleware("/"),
		outputFilenameMiddleware(),
		telemetryMiddleware(slog.New(slog.DiscardHandler), "", ":3000", "Gotenberg-Trace", nil),
		versionHeaderMiddleware(),
	)

	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
	srv.POST(
		"/forms/test",
		handler,
		contextMiddleware(fs, time.Minute, 0, downloadFromConfig{disable: true}, archiveConfig{}),
		hardTimeoutMiddleware(hardTimeout),
	)
	srv.POST("/other", func(c *echo.Context) error {
		time.Sleep(200 * time.Millisecond)
		return c.String(http.StatusOK, "other")
	})

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	return ts
}

// postMultipart posts a multipart form with one field to url.
func postMultipart(t *testing.T, url string) *http.Response {
	t.Helper()

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	err := form.WriteField("foo", "bar")
	if err != nil {
		t.Fatalf("write field: %v", err)
	}
	err = form.Close()
	if err != nil {
		t.Fatalf("close form: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Gotenberg-Trace", "my-trace")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}

	return resp
}

// A handler that outlives its hard timeout keeps running while Echo hands its
// pooled [echo.Context] to the next request. Whatever it does then, the next
// request gets its own answer only.
func TestHardTimeoutMiddleware_LateHandlerReachesNoOtherRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		late func(c *echo.Context)
	}{
		{
			name: "writes the response",
			late: func(c *echo.Context) {
				_ = c.JSON(http.StatusOK, map[string]string{"secret": "late"})
			},
		},
		{
			name: "sets an answer",
			late: func(c *echo.Context) {
				c.Get("context").(*Context).SetAnswer(map[string]string{"secret": "late"})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan struct{})
			ts := newMultipartServer(t, 50*time.Millisecond, func(c *echo.Context) error {
				defer func() { done <- struct{}{} }()
				time.Sleep(150 * time.Millisecond)
				tc.late(c)
				return nil
			})

			// Echo's pool may hand the next request another context, and
			// sync.Pool drops items at random under -race: a few rounds make
			// the reuse all but certain.
			for range 3 {
				resp := postMultipart(t, ts.URL+"/forms/test")
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusServiceUnavailable {
					t.Fatalf("late request: status %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
				}

				resp, err := http.Post(ts.URL+"/other", "text/plain", nil)
				if err != nil {
					t.Fatalf("post /other: %v", err)
				}
				body, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()

				<-done

				if resp.StatusCode != http.StatusOK || string(body) != "other" {
					t.Fatalf("other request: got %d %q, want %d %q", resp.StatusCode, body, http.StatusOK, "other")
				}
				if got := resp.Header.Get(echo.HeaderContentType); got != echo.MIMETextPlainCharsetUTF8 {
					t.Fatalf("other request: Content-Type %q, want %q", got, echo.MIMETextPlainCharsetUTF8)
				}
			}
		})
	}
}

// The API, not the handler, sends the answer a handler sets in time, with the
// headers of any other response.
func TestContextMiddleware_SendsAnswer(t *testing.T) {
	ts := newMultipartServer(t, time.Minute, func(c *echo.Context) error {
		ctx := c.Get("context").(*Context)

		var foo string
		err := ctx.FormData().String("foo", &foo).Validate()
		if err != nil {
			return err
		}

		ctx.SetAnswer(map[string]string{"foo": foo})

		return nil
	})

	resp := postMultipart(t, ts.URL+"/forms/test")
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", resp.StatusCode, http.StatusOK, body)
	}
	if got := string(body); got != "{\"foo\":\"bar\"}\n" {
		t.Errorf("body = %q, want the answer as JSON", got)
	}
	if got := resp.Header.Get(echo.HeaderContentType); got != echo.MIMEApplicationJSON {
		t.Errorf("Content-Type = %q, want %q", got, echo.MIMEApplicationJSON)
	}
	if got := resp.Header.Get("Gotenberg-Trace"); got != "my-trace" {
		t.Errorf("Gotenberg-Trace = %q, want %q", got, "my-trace")
	}
	if got := resp.Header.Get("Gotenberg-Version"); got == "" {
		t.Error("Gotenberg-Version missing")
	}
	if got := resp.Header.Get(echo.HeaderContentDisposition); got != "" {
		t.Errorf("Content-Disposition = %q, want none on an answer", got)
	}
}

func TestHardTimeoutMiddleware_PanicReturnsErrorImmediately(t *testing.T) {
	mw := hardTimeoutMiddleware(time.Minute)
	handler := mw(func(c *echo.Context) error { panic("boom") })

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())
	c.Set("logger", slog.New(slog.DiscardHandler))

	start := time.Now()
	err := handler(c)
	if err == nil {
		t.Fatal("expected an error for a panicking handler, got nil")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("handler returned after %s, want before the hard timeout", elapsed)
	}
	if status, _ := ParseError(err); status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestOidcAuthMiddleware(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	const (
		keyID    = "test-key"
		audience = "gotenberg"
	)

	oidcServer := &oidctest.Server{
		PublicKeys: []oidctest.PublicKey{
			{PublicKey: privateKey.Public(), KeyID: keyID, Algorithm: oidc.RS256},
		},
	}
	srv := httptest.NewServer(oidcServer)
	defer srv.Close()
	oidcServer.SetIssuer(srv.URL)

	// Building through the module's own helper exercises the discovery path too.
	a := &Api{oidcIssuer: srv.URL, oidcAudience: audience}
	verifier, err := a.buildOidcVerifier()
	if err != nil {
		t.Fatalf("build verifier: %v", err)
	}

	claims := func(issuer, aud string, expiresIn time.Duration) string {
		now := time.Now()
		return fmt.Sprintf(`{"iss":%q,"aud":%q,"sub":"user","exp":%d,"iat":%d}`,
			issuer, aud, now.Add(expiresIn).Unix(), now.Unix())
	}
	sign := func(claims string) string {
		return oidctest.SignIDToken(privateKey, keyID, oidc.RS256, claims)
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}

	for _, tc := range []struct {
		scenario   string
		authHeader string
		wantStatus int
	}{
		{"valid token", "Bearer " + sign(claims(srv.URL, audience, time.Hour)), http.StatusOK},
		{"lower-case scheme", "bearer " + sign(claims(srv.URL, audience, time.Hour)), http.StatusOK},
		{"upper-case scheme", "BEARER " + sign(claims(srv.URL, audience, time.Hour)), http.StatusOK},
		{"missing header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic Zm9vOmJhcg==", http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
		{"no space after scheme", "Bearer" + sign(claims(srv.URL, audience, time.Hour)), http.StatusUnauthorized},
		{"malformed token", "Bearer not-a-jwt", http.StatusUnauthorized},
		{"wrong issuer", "Bearer " + sign(claims("https://evil.example/", audience, time.Hour)), http.StatusUnauthorized},
		{"wrong audience", "Bearer " + sign(claims(srv.URL, "someone-else", time.Hour)), http.StatusUnauthorized},
		{"expired token", "Bearer " + sign(claims(srv.URL, audience, -time.Hour)), http.StatusUnauthorized},
		{"unknown signing key", "Bearer " + oidctest.SignIDToken(otherKey, "unknown", oidc.RS256, claims(srv.URL, audience, time.Hour)), http.StatusUnauthorized},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			c := echo.New().NewContext(req, httptest.NewRecorder())

			handler := oidcAuthMiddleware(verifier)(func(c *echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			err := handler(c)

			if tc.wantStatus == http.StatusOK {
				if err != nil {
					t.Fatalf("expected the request to pass, got error: %v", err)
				}
				return
			}

			var httpErr *echo.HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("expected an *echo.HTTPError, got %T (%v)", err, err)
			}
			if httpErr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", httpErr.Code, tc.wantStatus)
			}
		})
	}
}

// TestParseError_StatusMapping pins the statuses [ParseError] derives from the
// errors Echo and Gotenberg produce.
//
// Echo v5 models the router's ErrNotFound and ErrMethodNotAllowed as an
// unexported type rather than [echo.HTTPError], so matching that type alone
// would turn every unrouted request into a 500. It also guards the ordering:
// Gotenberg's own [SentinelHttpError] carries a client-facing message and must
// not be shadowed by the generic status lookup.
func TestParseError_StatusMapping(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{"router not found", echo.ErrNotFound, http.StatusNotFound, http.StatusText(http.StatusNotFound)},
		{"router method not allowed", echo.ErrMethodNotAllowed, http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed)},
		{"explicit HTTP error", echo.NewHTTPError(http.StatusUnauthorized, "nope"), http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)},
		{"wrapped HTTP error", fmt.Errorf("authenticate request: %w", echo.NewHTTPError(http.StatusUnauthorized, "nope")), http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized)},
		{"sentinel keeps its message", NewSentinelHttpError(http.StatusBadRequest, "Invalid 'foo' form field value"), http.StatusBadRequest, "Invalid 'foo' form field value"},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			status, message := ParseError(tc.err)
			if status != tc.wantStatus {
				t.Fatalf("status = %d, want %d", status, tc.wantStatus)
			}
			if message != tc.wantMessage {
				t.Fatalf("message = %q, want %q", message, tc.wantMessage)
			}
		})
	}
}

// TestNewEchoServer_RealIP pins the client IP extraction that the access log's
// "remote_ip" field depends on.
//
// Echo v5.1.0 dropped the X-Forwarded-For and X-Real-IP fallbacks from
// Context.RealIP, so without an explicit extractor a Gotenberg behind a reverse
// proxy would log the proxy's address for every request. [newEchoServer]
// restores the previous behavior.
func TestNewEchoServer_RealIP(t *testing.T) {
	srv := newEchoServer()
	if srv.IPExtractor == nil {
		t.Fatal("no IPExtractor configured: remote_ip would report the proxy address")
	}

	for _, tc := range []struct {
		scenario string
		headers  map[string]string
		want     string
	}{
		{"x-forwarded-for keeps the client, not the proxy", map[string]string{"X-Forwarded-For": "203.0.113.7, 70.41.3.18"}, "203.0.113.7"},
		{"single x-forwarded-for", map[string]string{"X-Forwarded-For": "203.0.113.7"}, "203.0.113.7"},
		{"bracketed IPv6 is unwrapped", map[string]string{"X-Forwarded-For": "[2001:db8::1], 70.41.3.18"}, "2001:db8::1"},
		{"x-real-ip when no x-forwarded-for", map[string]string{"X-Real-IP": "203.0.113.9"}, "203.0.113.9"},
		{"no headers falls back to the remote address", nil, "192.0.2.1"},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			c := srv.NewContext(req, httptest.NewRecorder())

			if got := c.RealIP(); got != tc.want {
				t.Fatalf("RealIP = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestNewEchoServer_AttachmentServesAbsolutePath pins the filesystem that every
// conversion response is sent through.
//
// Echo v5 serves files through Echo.Filesystem, an [fs.FS] rooted at the working
// directory, and [fs.FS] rejects absolute names. Gotenberg builds every output
// file under the request's temporary directory and hands Context.Attachment an
// absolute path, so with the default filesystem every conversion route answers
// 404 while still reading the whole upload.
func TestNewEchoServer_AttachmentServesAbsolutePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.pdf")
	want := []byte("%PDF-1.7 not really a PDF")

	err := os.WriteFile(path, want, 0o600)
	if err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	srv := newEchoServer()
	rec := httptest.NewRecorder()
	c := srv.NewContext(httptest.NewRequest(http.MethodPost, "/", nil), rec)

	err = c.Attachment(path, "output.pdf")
	if err != nil {
		t.Fatalf("Attachment(%q) = %v, want nil", path, err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="output.pdf"`) {
		t.Fatalf("Content-Disposition = %q, want it to carry filename=\"output.pdf\"", got)
	}
}

// Every answer carries the correlation ID and the version: the answers to an
// unknown route, a wrong method and a route that skips telemetry included. The
// 404 and the 405 name the path and the method rather than the status text.
func TestServer_EveryAnswerCarriesTraceAndVersion(t *testing.T) {
	for _, tc := range []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantBody    string
		wantVersion bool
	}{
		{name: "matched", method: http.MethodGet, path: "/health", wantStatus: http.StatusOK, wantVersion: true},
		{name: "route without telemetry", method: http.MethodGet, path: "/version", wantStatus: http.StatusOK, wantVersion: true},
		{
			name: "not found", method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound, wantVersion: true,
			wantBody: "The route '/nope' does not exist. Check the path, and that this image includes the module that serves it",
		},
		{
			name: "method not allowed", method: http.MethodPost, path: "/health", wantStatus: http.StatusMethodNotAllowed, wantVersion: true,
			wantBody: "The route '/health' does not accept the POST method. Use one of: OPTIONS, GET",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newEchoServer()
			srv.Pre(
				latencyMiddleware(),
				rootPathMiddleware("/"),
				telemetryMiddleware(slog.New(slog.DiscardHandler), "", ":3000", "Gotenberg-Trace", []string{"version"}),
				versionHeaderMiddleware(),
			)
			srv.GET("/health", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })
			srv.GET("/version", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Gotenberg-Trace", "my-trace")
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := rec.Header().Get("Gotenberg-Trace"); got != "my-trace" {
				t.Errorf("Gotenberg-Trace = %q, want %q", got, "my-trace")
			}
			if got := rec.Header().Get("Gotenberg-Version"); (got != "") != tc.wantVersion {
				t.Errorf("Gotenberg-Version = %q, want set: %v", got, tc.wantVersion)
			}
			if tc.wantBody != "" && rec.Body.String() != tc.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}
