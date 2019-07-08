package printer

import (
	"context"
	"os/exec"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

type merge struct {
	ctx    context.Context
	fpaths []string
	opts   *MergeOptions
}

// MergeOptions helps customizing the
// merge printer behaviour.
type MergeOptions struct {
	WaitTimeout float64
}

// NewMerge returns a merge printer.
func NewMerge(fpaths []string, opts *MergeOptions) Printer {
	return &merge{
		fpaths: fpaths,
		opts:   opts,
	}
}

func (p *merge) Print(destination string) error {
	const op = "printer.merge.Print"
	if p.ctx == nil {
		// FIXME duration not working with float
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.opts.WaitTimeout)*time.Second)
		defer cancel()
		p.ctx = ctx
	}
	var cmdArgs []string
	cmdArgs = append(cmdArgs, p.fpaths...)
	cmdArgs = append(cmdArgs, "cat", "output", destination)
	cmd := exec.CommandContext(p.ctx, "pdftk", cmdArgs...)
	_, err := cmd.Output()
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(merge))
)
