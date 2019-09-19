package print

import (
	"context"

	"github.com/thecodingmachine/gotenberg/internal/pkg/process"
)

// Print is a type that can create a PDF file from a source.
// The source is defined in the underlying implementation.
type Print interface {
	Print(ctx context.Context, dest string, proc process.Process) error
}
