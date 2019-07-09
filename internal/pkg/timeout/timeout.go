package timeout

import (
	"context"
	"time"
)

// Context creates a context with timeout for
// given second.
func Context(seconds float64) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), Duration(seconds))
}

// Duration creates a duration from seconds.
func Duration(seconds float64) time.Duration {
	return time.Duration(1000*seconds) * time.Millisecond
}
