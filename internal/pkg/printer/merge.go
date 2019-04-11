package printer

import (
	"context"
	"fmt"
	"os/exec"
	"time"
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
	if p.ctx == nil {
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
		return fmt.Errorf("pdtk: %v", err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(merge))
)
