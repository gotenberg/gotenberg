package gotenberg

import (
	"bytes"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"
)

func TestLeveledLogger_RedactsURLs(t *testing.T) {
	for _, tc := range []struct {
		name          string
		log           func(leveled *LeveledLogger)
		expectPresent []string
	}{
		{
			name: "url",
			log: func(leveled *LeveledLogger) {
				leveled.Debug("performing request", "method", "POST", "url", "https://user:pass@example.com/hook?token=secret")
			},
			expectPresent: []string{"https://example.com/hook"},
		},
		{
			name: "request",
			log: func(leveled *LeveledLogger) {
				leveled.Debug("retrying request", "request", "POST https://example.com/hook?token=secret", "timeout", "1s")
			},
			expectPresent: []string{"POST https://example.com/hook"},
		},
		{
			name: "error",
			log: func(leveled *LeveledLogger) {
				leveled.Error("request failed", "error", &url.Error{Op: "Post", URL: "https://example.com/hook?token=secret", Err: errors.New("connection refused")})
			},
			expectPresent: []string{`Post \"https://example.com/hook\": connection refused`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			tc.log(NewLeveledLogger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))))

			out := buf.String()
			if strings.Contains(out, "secret") || strings.Contains(out, "pass@") {
				t.Errorf("expected no credential in %q", out)
			}
			for _, want := range tc.expectPresent {
				if !strings.Contains(out, want) {
					t.Errorf("expected %q in %q", want, out)
				}
			}
		})
	}
}
