package prinery

import (
	"context"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type mergePrinter struct {
	logger xlog.Logger
	fpaths []string
}

func (p mergePrinter) print(ctx context.Context, spec processSpec, dest string) error {
	const op string = "prinery.mergePrinter.print"
	p.logger.DebugfOp(op, "merging '%v'...", p.fpaths)
	resolver := func() error {
		var args []string
		args = append(args, p.fpaths...)
		args = append(args, "cat", "output", dest)
		cmd, err := xexec.CommandContext(ctx, p.logger, "pdftk", args...)
		if err != nil {
			return err
		}
		xexec.LogBeforeExecute(p.logger, cmd)
		return cmd.Run()
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = printer(new(mergePrinter))
)
