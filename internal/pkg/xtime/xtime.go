package xtime

import (
	"time"
)

// Duration creates a time.Duration from seconds.
func Duration(seconds float64) time.Duration {
	return time.Duration(1000*seconds) * time.Millisecond
}
