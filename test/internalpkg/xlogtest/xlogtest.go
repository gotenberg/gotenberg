package xlogtest

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

// DebugLogger creates a xlog.Logger
// with xlog.DebugLevel for our tests.
func DebugLogger() xlog.Logger {
	return xlog.New(xlog.DebugLevel, "tests")
}

// InfoLogger creates a xlog.Logger
// with xlog.InfoLevel for our tests.
func InfoLogger() xlog.Logger {
	return xlog.New(xlog.DebugLevel, "tests")
}

// ErrorLogger creates a xlog.Logger
// with xlog.ErrorLevel for our tests.
func ErrorLogger() xlog.Logger {
	return xlog.New(xlog.ErrorLevel, "tests")
}
