package print

import (
	"context"

	"github.com/thecodingmachine/gotenberg/internal/pkg/process"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type mergePrint struct {
	logger xlog.Logger
	fpaths []string
}

// NewMergePrint returns a Print for
// merging PDF files.
func NewMergePrint(logger xlog.Logger, fpaths []string) Print {
	return mergePrint{
		logger: logger,
		fpaths: fpaths,
	}
}

func (p mergePrint) Print(ctx context.Context, dest string, proc process.Process) error {
	const op string = "print.mergePrint.Print"
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
