package printer

import (
	"context"
	"fmt"
	"strings"

	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// Printer is a type that can create a PDF file from a source.
// The source is defined in the underlying implementation.
type Printer interface {
	Print(destination string) error
}

func handleErrContext(ctx context.Context, previousErr error) error {
	const op = "printer.handleErrContext"
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
