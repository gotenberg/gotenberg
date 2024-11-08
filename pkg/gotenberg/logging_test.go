package gotenberg

import (
	"testing"

	"go.uber.org/zap"
)

func TestLeveledLogger_Error(t *testing.T) {
	NewLeveledLogger(zap.NewNop()).Error("foo")
}

func TestLeveledLogger_Warn(t *testing.T) {
	NewLeveledLogger(zap.NewNop()).Warn("foo")
}

func TestLeveledLogger_Info(t *testing.T) {
	NewLeveledLogger(zap.NewNop()).Info("foo")
}

func TestLeveledLogger_Debug(t *testing.T) {
	NewLeveledLogger(zap.NewNop()).Debug("foo")
}
