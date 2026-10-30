package log

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestLevelColorWriter(t *testing.T) {
	for _, tc := range []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, `level=` + "\x1b[36mDEBUG\x1b[0m" + ` msg="level=x"`},
		{slog.LevelInfo, `level=` + "\x1b[34mINFO\x1b[0m" + ` msg="level=x"`},
		{slog.LevelWarn + 2, `level=` + "\x1b[33mWARN+2\x1b[0m" + ` msg="level=x"`},
		{slog.LevelError, `level=` + "\x1b[31mERROR\x1b[0m" + ` msg="level=x"`},
	} {
		t.Run(tc.level.String(), func(t *testing.T) {
			var buf bytes.Buffer
			opts := &slog.HandlerOptions{
				Level: slog.LevelDebug,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}

			slog.New(slog.NewTextHandler(levelColorWriter{w: &buf}, opts)).Log(t.Context(), tc.level, "level=x")

			if got := buf.String(); got != tc.want+"\n" {
				t.Errorf("got %q, want %q", got, tc.want+"\n")
			}
		})
	}
}
