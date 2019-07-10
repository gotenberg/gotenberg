package timeout

import (
	"context"
	"fmt"
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

// Err checks if there is an error in the given context
// and wraps the previous error inside a standarderror.Error.
func Err(ctx context.Context, previousErr error) error {
	const op string = "timeout.Err"
	if previousErr == nil {
		panic(fmt.Sprintf("%s: previous error should not be nil", op))
	}
	err := ctx.Err()
	if err == nil {
		return previousErr
	}
	if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		return &standarderror.Error{
			Code:    standarderror.Timeout,
			Message: "context has timed out",
			Op:      op,
			Err:     previousErr,
		}
	}
	return &standarderror.Error{
		Message: "context finished with an error",
		Op:      op,
		Err:     previousErr,
	}
}
