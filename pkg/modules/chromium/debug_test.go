package chromium

import (
	"testing"

	"go.uber.org/zap"
)

func TestDebugLogger_Write(t *testing.T) {
	actual, err := (&debugLogger{logger: zap.NewNop()}).Write([]byte("foo"))
	expected := len([]byte("foo"))

	if actual != expected {
		t.Errorf("expected %d but got %d", expected, actual)
	}

	if err != nil {
		t.Errorf("expected not error but got: %v", err)
	}
}

func TestDebugLogger_Printf(t *testing.T) {
	(&debugLogger{logger: zap.NewNop()}).Printf("%s", "foo")
}
