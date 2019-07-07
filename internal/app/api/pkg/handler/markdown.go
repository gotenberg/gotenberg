package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

// Markdown is the endpoint for converting
// Markdown to PDF.
func Markdown(c echo.Context) error {
	ctx := context.MustCastFromEchoContext(c)
	r := ctx.Resource()
	opts, err := r.ChromePrinterOptions()
	if err != nil {
		return err
	}
	fpath, err := r.Fpath("index.html")
	if err != nil {
		return err
	}
	p, err := printer.NewMarkdown(fpath, opts)
	if err != nil {
		return err
	}
	return convert(ctx, p)
}
