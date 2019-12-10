package printer

import (
	"context"
	"sort"

	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xcontext"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type mergePrinter struct {
	ctx    context.Context
	logger xlog.Logger
	fpaths []string
	opts   MergePrinterOptions
}

// MergePrinterOptions helps customizing the
// merge Printer behaviour.
type MergePrinterOptions struct {
	WaitTimeout float64
}

// DefaultMergePrinterOptions returns the default
// merge Printer options.
func DefaultMergePrinterOptions(config conf.Config) MergePrinterOptions {
	return MergePrinterOptions{
		WaitTimeout: config.DefaultWaitTimeout(),
	}
}

// NewMergePrinter returns a Printer which
// is able to merge PDFs.
func NewMergePrinter(logger xlog.Logger, fpaths []string, opts MergePrinterOptions) Printer {
	return mergePrinter{
		logger: logger,
		fpaths: fpaths,
		opts:   opts,
	}
}

func (p mergePrinter) Print(destination string) error {
	const op string = "printer.mergePrinter.Print"
	/*
		context.Context may be providen from
		an officePrinter which needs to merge
		its result files.
	*/
	if p.ctx == nil {
		logOptions(p.logger, p.opts)
		ctx, cancel := xcontext.WithTimeout(p.logger, p.opts.WaitTimeout)
		defer cancel()
		p.ctx = ctx
	}
	// see https://github.com/thecodingmachine/gotenberg/issues/139.
	sort.Strings(p.fpaths)
	p.logger.DebugfOp(op, "merging '%v'...", p.fpaths)
	resolver := func() error {
		var args []string
		args = append(args, p.fpaths...)
		args = append(args, "cat", "output", destination)
		return xexec.Run(p.ctx, p.logger, "pdftk", args...)
	}
	if err := resolver(); err != nil {
		return xcontext.MustHandleError(
			p.ctx,
			xerror.New(op, err),
		)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(mergePrinter))
)
