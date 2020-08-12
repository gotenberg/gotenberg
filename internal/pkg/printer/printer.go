package printer

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

// Printer is a type that can create a PDF file from a source.
// The source is defined in the underlying implementation.
type Printer interface {
	Print(destination string) error
}

func logOptions(logger xlog.Logger, opts interface{}) {
	const op string = "printer.logOptions"
	logger.DebugOpf(op, "options: %+v", opts)
}
