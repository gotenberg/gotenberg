package logging

import (
	"fmt"

	"go.uber.org/zap/zapcore"
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

func levelToColor(l zapcore.Level) color {
	switch l {
	case zapcore.DebugLevel:
		return cyan
	case zapcore.InfoLevel:
		return blue
	case zapcore.WarnLevel:
		return yellow
	case zapcore.ErrorLevel:
		return red
	case zapcore.DPanicLevel:
		return red
	case zapcore.PanicLevel:
		return red
	case zapcore.FatalLevel:
		return red
	case zapcore.InvalidLevel:
		return red
	default:
		return red
	}
}
