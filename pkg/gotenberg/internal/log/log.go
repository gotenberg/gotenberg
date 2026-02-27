package log

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger initializes the global logger.
func InitLogger(level zapcore.Level, fieldsPrefix string, cores ...zapcore.Core) error {
	if logger != nil {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	// Double check: ensure it wasn't initialized while we waited for the lock.
	if logger != nil {
		return nil
	}

	for i, core := range cores {
		_, ok := core.(rootCore)
		if !ok {
			cores[i] = rootCore{Core: core, level: level, fieldsPrefix: fieldsPrefix}
		}
	}

	teeCore := zapcore.NewTee(cores...)
	logger = zap.New(teeCore)

	return nil
}

// Logger returns the global logger.
func Logger() *zap.Logger {
	mu.Lock()
	defer mu.Unlock()

	return logger
}

// logger is Singleton so that we instantiate our [zap.Logger] only once.
var (
	logger *zap.Logger
	mu     sync.Mutex
)
