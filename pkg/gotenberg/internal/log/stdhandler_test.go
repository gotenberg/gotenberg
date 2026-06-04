package log

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestNewStdHandler_LevelCase(t *testing.T) {
	for _, tc := range []struct {
		name      string
		levelCase string
		want      string
	}{
		{"lower is the default behavior", "lower", "info"},
		{"upper keeps slog casing", "upper", "INFO"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := os.Stderr
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stderr = writer
			defer func() { os.Stderr = original }()

			handler, err := NewStdHandler(slog.LevelInfo, "json", "", false, tc.levelCase)
			if err != nil {
				t.Fatalf("create handler: %v", err)
			}

			slog.New(handler).Info("hello")

			if err := writer.Close(); err != nil {
				t.Fatalf("close writer: %v", err)
			}
			out, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}

			var record map[string]any
			if err := json.Unmarshal(out, &record); err != nil {
				t.Fatalf("parse log line %q: %v", out, err)
			}
			if record["level"] != tc.want {
				t.Errorf("level = %v, want %v", record["level"], tc.want)
			}
		})
	}
}
