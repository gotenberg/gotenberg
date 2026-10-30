package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// decode returns the JSON log lines written to buf.
func decode(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var lines []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}

		var entry map[string]any
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			t.Fatalf("decode log line %q: %v", line, err)
		}
		lines = append(lines, entry)
	}

	return lines
}

func TestGotenbergHandler(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		prefix   string
		log      func(logger *slog.Logger)
		expect   map[string]any
		absent   []string
	}{
		{
			scenario: "logger names join with a dot",
			log: func(logger *slog.Logger) {
				logger.With(slog.String("logger", "api")).With(slog.String("logger", "gc")).Info("msg")
			},
			expect: map[string]any{"logger": "api.gc"},
		},
		{
			scenario: "a prefix applies to fields, not to the reserved ones",
			prefix:   "gotenberg",
			log: func(logger *slog.Logger) {
				logger.With(slog.String("logger", "api"), slog.String("correlation_id", "abc"), slog.String("host", "h")).
					Info("msg", slog.String("trace_id", "t"), slog.String("uri", "/u"))
			},
			expect: map[string]any{"logger": "api", "correlation_id": "abc", "gotenberg_host": "h", "trace_id": "t", "gotenberg_uri": "/u"},
			absent: []string{"host", "uri", "gotenberg_correlation_id", "gotenberg_trace_id"},
		},
		{
			scenario: "no prefix keeps the fields as they are",
			log: func(logger *slog.Logger) {
				logger.With(slog.String("logger", "api")).Info("msg", slog.String("uri", "/u"))
			},
			expect: map[string]any{"logger": "api", "uri": "/u"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			buf := new(bytes.Buffer)
			tc.log(slog.New(NewGotenbergHandler(slog.NewJSONHandler(buf, nil), tc.prefix)))

			lines := decode(t, buf)
			if len(lines) != 1 {
				t.Fatalf("expected 1 log line, got %d: %s", len(lines), buf)
			}

			for key, want := range tc.expect {
				if got := lines[0][key]; got != want {
					t.Errorf("%s = %v, want %v (line: %v)", key, got, want, lines[0])
				}
			}
			for _, key := range tc.absent {
				if _, ok := lines[0][key]; ok {
					t.Errorf("%s should be absent (line: %v)", key, lines[0])
				}
			}
		})
	}
}

func TestFanOut(t *testing.T) {
	debug, warn := new(bytes.Buffer), new(bytes.Buffer)
	logger := slog.New(FanOut(
		slog.NewJSONHandler(debug, &slog.HandlerOptions{Level: slog.LevelDebug}),
		slog.NewJSONHandler(warn, &slog.HandlerOptions{Level: slog.LevelWarn}),
	)).With(slog.String("k", "v")).WithGroup("g")

	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("expected the fan-out to be enabled when one handler is")
	}

	logger.Debug("debug", slog.String("a", "1"))
	logger.Warn("warn", slog.String("a", "2"))

	debugLines, warnLines := decode(t, debug), decode(t, warn)
	if len(debugLines) != 2 || len(warnLines) != 1 {
		t.Fatalf("expected 2 debug and 1 warn lines, got %d and %d", len(debugLines), len(warnLines))
	}

	for _, line := range append(debugLines, warnLines...) {
		if line["k"] != "v" {
			t.Errorf("attribute k lost through the fan-out: %v", line)
		}
		if _, ok := line["g"].(map[string]any); !ok {
			t.Errorf("group g lost through the fan-out: %v", line)
		}
	}
}

func TestLevelFilter(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := slog.New(LevelFilter(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}), slog.LevelInfo)).
		With(slog.String("k", "v")).WithGroup("g")

	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("expected debug to be filtered out")
	}

	logger.Debug("dropped")
	logger.Info("kept", slog.String("a", "1"))

	lines := decode(t, buf)
	if len(lines) != 1 || lines[0]["msg"] != "kept" || lines[0]["k"] != "v" {
		t.Fatalf("expected only the info line with its attributes, got %v", lines)
	}
}
