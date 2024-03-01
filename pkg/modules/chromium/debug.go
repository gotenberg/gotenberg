package chromium

import (
	"fmt"
	"io"

	"go.uber.org/zap"
)

// debugLogger is wrapper around a [zap.Logger] which is used for debugging
// Chromium.
type debugLogger struct {
	logger *zap.Logger
}

// Write logs the bytes in a debug message.
func (debug *debugLogger) Write(p []byte) (n int, err error) {
	debug.logger.Debug(string(p))

	return len(p), nil
}

// Printf logs a debug message.
func (debug *debugLogger) Printf(format string, v ...interface{}) {
	debug.logger.Debug(fmt.Sprintf(format, v...))
}

// Interface guards.
var (
	_ io.Writer = (*debugLogger)(nil)
)
