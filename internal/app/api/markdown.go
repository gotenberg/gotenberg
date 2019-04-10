package api

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func convertMarkdown(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.chromePrinterOptions()
	if err != nil {
		return err
	}
	fpath, err := ctx.resource.fpath("index.html")
	if err != nil {
		return err
	}
	p, err := printer.NewMarkdown(fpath, opts)
	if err != nil {
		return err
	}
	return convert(ctx, p)
}
