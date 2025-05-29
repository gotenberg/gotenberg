package gotenberg

import (
	"fmt"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

// LoggerProvider is an interface for a module that supplies a method for
// creating a [zap.Logger] instance for use by other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.LoggerProvider))
//		logger, _   := provider.(gotenberg.LoggerProvider).Logger(m)
//	}
type LoggerProvider interface {
	Logger(mod Module) (*zap.Logger, error)
}

// LeveledLogger is a wrapper around a [zap.Logger] so that it may be used by a
// [retryablehttp.Client].
type LeveledLogger struct {
	logger *zap.Logger
}

// NewLeveledLogger instantiates a [LeveledLogger].
func NewLeveledLogger(logger *zap.Logger) *LeveledLogger {
	return &LeveledLogger{
		logger: logger,
	}
}

// Error logs a message at the error level using the wrapped zap.Logger.
func (leveled LeveledLogger) Error(msg string, keysAndValues ...interface{}) {
	leveled.logger.Error(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Warn logs a message at the warning level using the wrapped zap.Logger.
func (leveled LeveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	leveled.logger.Warn(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Info logs a message at the info level using the wrapped zap.Logger.
func (leveled LeveledLogger) Info(msg string, keysAndValues ...interface{}) {
	leveled.logger.Info(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Debug logs a message at the debug level using the wrapped zap.Logger.
func (leveled LeveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	leveled.logger.Debug(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Interface guards.
var (
	_ retryablehttp.LeveledLogger = (*LeveledLogger)(nil)
)
