package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// Merge is the endpoint for
// merging PDF files.
func Merge(c echo.Context) error {
	const op = "handler.Merge"
	ctx := context.MustCastFromEchoContext(c)
	ctx.StandardLogger().DebugfOp(op, "merge request")
	r := ctx.Resource()
	opts, err := r.MergePrinterOptions()
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	fpaths, err := r.Fpaths(".pdf")
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	p := printer.NewMerge(fpaths, opts)
	if err := convert(ctx, p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}
