package log

import (
	"log/slog"
	"sync"
)

// InitLogger initializes the global logger.
func InitLogger(handler slog.Handler) {
	if logger != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Double check: ensure it wasn't initialized while we waited for the lock.
	if logger != nil {
		return
	}

	logger = slog.New(handler)
}

// Logger returns the global logger.
func Logger() *slog.Logger {
	mu.Lock()
	defer mu.Unlock()

	return logger
}

// logger is Singleton so that we instantiate our [slog.Logger] only once.
var (
	logger *slog.Logger
	mu     sync.Mutex
)
