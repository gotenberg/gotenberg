package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// Markdown is the endpoint for converting
// Markdown to PDF.
func Markdown(c echo.Context) error {
	const op = "handler.Markdown"
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
	p, err := printer.NewMarkdown(fpath, opts)
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	if err := convert(ctx, p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}
