package log

import (
	"fmt"
	"log/slog"
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

func levelToColor(l slog.Level) color {
	switch l {
	case slog.LevelDebug:
		return cyan
	case slog.LevelInfo:
		return blue
	case slog.LevelWarn:
		return yellow
	case slog.LevelError:
		return red
	default:
		return red
	}
}
