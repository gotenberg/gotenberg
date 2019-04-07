package printer

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"time"
)

type merge struct {
	ctx    context.Context
	fpaths []string
	opts   *MergeOptions
}

// MergeOptions helps customizing the
// Merge printer behaviour.
type MergeOptions struct {
	WaitTimeout float64
}

// NewMerge returns a merge printer.
func NewMerge(fpaths []string, opts *MergeOptions) (Printer, error) {
	if len(fpaths) <= 1 {
		return nil, fmt.Errorf("merge requires at least two files to merge: got %d", len(fpaths))
	}
	sort.Strings(fpaths)
	// if no custom timeout, set default timeout to 30 seconds.
	if opts.WaitTimeout == 0.0 {
		opts.WaitTimeout = 30.0
	}
	return &merge{
		fpaths: fpaths,
		opts:   opts,
	}, nil
}

func (m *merge) Print(destination string) error {
	if m.ctx == nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.opts.WaitTimeout)*time.Second)
		defer cancel()
		m.ctx = ctx
	}
	var cmdArgs []string
	cmdArgs = append(cmdArgs, m.fpaths...)
	cmdArgs = append(cmdArgs, "cat", "output", destination)
	cmd := exec.CommandContext(m.ctx, "pdftk", cmdArgs...)
	_, err := cmd.Output()
	if m.ctx.Err() == context.DeadlineExceeded {
		return errors.New("pdftk: command timed out")
	}
	if err != nil {
		return fmt.Errorf("pdtk: non-zero exit code: %v", err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(merge))
)
