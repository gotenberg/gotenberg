package api

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func merge(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.mergePrinterOptions()
	if err != nil {
		return err
	}
	fpaths, err := ctx.resource.fpaths(".pdf")
	if err != nil {
		return err
	}
	p := printer.NewMerge(fpaths, opts)
	return convert(ctx, p)
}
