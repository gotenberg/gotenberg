package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// HTML is the endpoint for converting
// HTML to PDF.
func HTML(c echo.Context) error {
	const op = "handler.HTML"
	ctx := context.MustCastFromEchoContext(c)
	r := ctx.Resource()
	opts, err := r.ChromePrinterOptions()
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	fpath, err := r.Fpath("index.html")
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	p := printer.NewHTML(fpath, opts)
	if err := convert(ctx, p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}
