package notify

import (
	"errors"
	"testing"
)

// dumb test to improve code coverage.
func TestNotify(t *testing.T) {
	Println("foo")
	ErrPrintln(errors.New("foo"))
}
