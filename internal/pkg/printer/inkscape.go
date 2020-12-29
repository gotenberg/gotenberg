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

type inkscapePrinter struct {
	ctx    context.Context
	logger xlog.Logger
	fpaths []string
	opts   InkscapePrinterOptions
}

// InkscapePrinterOptions helps customizing the
// inkscape Printer behaviour.
type InkscapePrinterOptions struct {
	WaitTimeout float64
}

// DefaultInkscapePrinterOptions returns the default
// inkscape Printer options.
func DefaultInkscapePrinterOptions(config conf.Config) InkscapePrinterOptions {
	return InkscapePrinterOptions{
		WaitTimeout: config.DefaultWaitTimeout(),
	}
}

// NewInkscapePrinter returns a Printer which
// is able to inkscape PDFs.
func NewInkscapePrinter(logger xlog.Logger, fpaths []string, opts InkscapePrinterOptions) Printer {
	return inkscapePrinter{
		logger: logger,
		fpaths: fpaths,
		opts:   opts,
	}
}

func (p inkscapePrinter) Print(destination string) error {
	const op string = "printer.inkscapePrinter.Print"
	/*
		context.Context may be providen from
		an officePrinter which needs to inkscape
		its result files.
	*/
	if p.ctx == nil {
		logOptions(p.logger, p.opts)
		ctx, cancel := xcontext.WithTimeout(p.logger, p.opts.WaitTimeout)
		defer cancel()
		p.ctx = ctx
	}
	// see https://github.com/thecodingmachine/gotenberg/issues/139.
	/*
			command to output a pdf :
	        	inkscape --without-gui --export-pdf={out_filename} {svg_filename}
			command to output a png :
	        	inkscape --without-gui --export-area-page --export-dpi=100 --export-png={out_filename} {svg_filename}
	*/
	sort.Strings(p.fpaths)
	p.logger.DebugOpf(op, "inkscape converting '%v'...", p.fpaths)
	resolver := func() error {
		var args []string
		args = append(args, "--without-gui", "--export-pdf="+destination)
		args = append(args, p.fpaths...)
		//args = append(args, "cat", "output", destination)
		return xexec.Run(p.ctx, p.logger, "inkscape", args...)
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
	_ = Printer(new(inkscapePrinter))
)
