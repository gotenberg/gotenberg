package chromium

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// debugLogger is wrapper around a [slog.Logger] which is used for debugging
// Chromium.
type debugLogger struct {
	logger *slog.Logger
}

// Write logs the bytes in a debug message.
func (debug *debugLogger) Write(p []byte) (n int, err error) {
	debug.logger.DebugContext(context.Background(), string(p))

	return len(p), nil
}

// Printf logs a debug message.
func (debug *debugLogger) Printf(format string, v ...any) {
	debug.logger.DebugContext(context.Background(), fmt.Sprintf(format, v...))
}

// Interface guards.
var (
	_ io.Writer = (*debugLogger)(nil)
)
