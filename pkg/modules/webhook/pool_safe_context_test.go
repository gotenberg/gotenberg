package webhook

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestNewDetachedContext_SurvivesUnderlyingReset(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	logger := slog.Default()
	c.Set("logger", logger)
	c.Set("correlationId", "abc-123")

	detached := newDetachedContext(c, "logger", "correlationId", "missing")

	// Simulate Echo recycling c for a concurrent request. Reset clears the
	// pooled store, which is exactly the crash scenario the detached context
	// guards against.
	c.Reset(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())

	if got, _ := detached.Get("logger").(*slog.Logger); got != logger {
		t.Fatalf("logger = %v, want snapshotted default logger", got)
	}
	if got, _ := detached.Get("correlationId").(string); got != "abc-123" {
		t.Fatalf("correlationId = %q, want %q", got, "abc-123")
	}
	if got := detached.Get("missing"); got != nil {
		t.Fatalf("missing key returned %v, want nil", got)
	}

	// Underlying c must remain clean.
	if c.Get("logger") != nil {
		t.Fatalf("underlying c.Get(\"logger\") leaked detached state after reset")
	}
}

func TestNewDetachedContext_SetDoesNotTouchUnderlying(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	detached := newDetachedContext(c)
	detached.Set("foo", "bar")

	if got, _ := detached.Get("foo").(string); got != "bar" {
		t.Fatalf("detached Get = %q, want bar", got)
	}
	if c.Get("foo") != nil {
		t.Fatalf("Set leaked %q to the underlying pooled context", "foo")
	}
}
