package log

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Foreground colors.
// Copy pasted from go.uber.org/zap/internal/color/color.go
const (
	black color = iota + 30
	red
	green
	yellow
	blue
	magenta
	cyan
	white
)

type color uint8

func (c color) Add(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
}

// levelColor returns the color of a level as the text handler writes it,
// slog's ("INFO", "WARN+2") or GCP's ("WARNING").
func levelColor(level string) color {
	switch {
	case strings.HasPrefix(level, "DEBUG"):
		return cyan
	case strings.HasPrefix(level, "INFO"):
		return blue
	case strings.HasPrefix(level, "WARN"):
		return yellow
	default:
		return red
	}
}

// levelKey precedes the level in a line of the text handler, which writes the
// time first, then the level.
var levelKey = []byte("level=")

// levelColorWriter colors the level of each line a [slog.TextHandler] writes
// to w. The handler quotes and escapes a value holding an escape sequence, so
// the color goes in once it formatted the line. The handler writes a line per
// call.
type levelColorWriter struct {
	w io.Writer
}

func (cw levelColorWriter) Write(p []byte) (int, error) {
	start := bytes.Index(p, levelKey)
	if start < 0 {
		return cw.w.Write(p)
	}
	start += len(levelKey)

	end := bytes.IndexByte(p[start:], ' ')
	if end < 0 {
		return cw.w.Write(p)
	}
	end += start

	level := string(p[start:end])

	line := make([]byte, 0, len(p)+len("\x1b[00m\x1b[0m"))
	line = append(line, p[:start]...)
	line = append(line, levelColor(level).Add(level)...)
	line = append(line, p[end:]...)

	_, err := cw.w.Write(line)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}
