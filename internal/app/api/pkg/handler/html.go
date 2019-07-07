package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

// HTML is the endpoint for converting
// HTML to PDF.
func HTML(c echo.Context) error {
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
	p := printer.NewHTML(fpath, opts)
	return convert(ctx, p)
}
