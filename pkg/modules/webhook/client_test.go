package webhook

import (
	"testing"

	"go.uber.org/zap"
)

func TestLeveledLogger_Error(t *testing.T) {
	leveledLogger{logger: zap.NewNop()}.Error("foo")
}

func TestLeveledLogger_Warn(t *testing.T) {
	leveledLogger{logger: zap.NewNop()}.Warn("foo")
}

func TestLeveledLogger_Info(t *testing.T) {
	leveledLogger{logger: zap.NewNop()}.Info("foo")
}

func TestLeveledLogger_Debug(t *testing.T) {
	leveledLogger{logger: zap.NewNop()}.Debug("foo")
}
