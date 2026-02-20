package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// Propagate HTTP request context cancellation to processing modules to save resources
// https://github.com/gotenberg/gotenberg/issues/1455
func TestNewContext_Cancellation(t *testing.T) {
	e := echo.New()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	err := writer.Close()
	if err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	// Create a request with a cancellable context.
	reqCtx, cancelReq := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/", body).WithContext(reqCtx)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	logger := zap.NewNop()
	fs := gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll))
	timeout := time.Duration(10) * time.Second
	downloadFromCfg := downloadFromConfig{
		disable: true,
	}

	ctx, cancel, err := newContext(c, logger, fs, timeout, 0, downloadFromCfg, "trace", "trace")
	if err != nil {
		t.Fatalf("expected no error from newContext, got: %v", err)
	}
	defer cancel()

	// Verify initial state: context SHOULD NOT be done yet.
	select {
	case <-ctx.Done():
		t.Fatal("context should not be done immediately")
	default:
	}

	// Simulate Client Disconnect
	cancelReq()

	// Verify Propagation
	select {
	case <-ctx.Done():
		// Success! The context was cancelled.
		if ctx.Err() != context.Canceled {
			t.Errorf("expected context error to be 'context.Canceled', got: %v", ctx.Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected context to be cancelled after request context cancellation, but it timed out")
	}
}
