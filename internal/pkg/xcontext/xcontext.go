package xcontext

import (
	"context"
	"fmt"
	"strings"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
)

// WithTimeout creates a context.Context which
// times out after given seconds.
func WithTimeout(logger xlog.Logger, seconds float64) (context.Context, context.CancelFunc) {
	const op string = "xcontext.WithTimeout"
	logger.DebugOpf(op, "creating context with '%.2fs' of timeout...", seconds)
	return context.WithTimeout(context.Background(), xtime.Duration(seconds))
}

/*
MustHandleError checks if there is an error
in the given Context.

If no error, returns the previous error.

If context.DeadlineExceeded, wraps the previous
error inside an xerror.Error with xerror.TimeoutCode.

Otherwise wraps the previous error inside an
xerror.Error.

It panics if no previous error.
*/
func MustHandleError(ctx context.Context, previousErr error) error {
	const op string = "xcontext.MustHandleError"
	if previousErr == nil {
		panic(fmt.Sprintf("%s: previous error should not be nil", op))
	}
	err := ctx.Err()
	if err == nil {
		// we do not wrap the previous error
		// as it should be wrapped by the caller.
		return previousErr
	}
	// context has timed out
	if strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		return xerror.Timeout(op, "context has timed out", previousErr)
	}
	/*
		context has another error: we do not
		wrap the error from the Context as the previous
		error should contain it.
	*/
	return xerror.New(op, previousErr)
}
