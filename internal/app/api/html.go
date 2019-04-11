package api

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func convertHTML(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.chromePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	fpath, err := ctx.resource.fpath("index.html")
	if err != nil {
		return &errBadRequest{err}
	}
	p := printer.NewHTML(fpath, opts)
	return convert(ctx, p)
}
