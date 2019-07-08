package timeout

import (
	"context"
	"strings"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
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

// Err returns a standarderror.Error
// if the context has an error.
func Err(ctx context.Context) error {
	const op = "timeout.Err"
	err := ctx.Err()
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		return &standarderror.Error{
			Code:    standarderror.Timeout,
			Message: "context has timed out",
			Op:      op,
			Err:     err,
		}
	}
	return &standarderror.Error{
		Message: "context finished with an error",
		Op:      op,
		Err:     err,
	}
}
