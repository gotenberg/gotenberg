package api

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func convertURL(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.chromePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	remote, err := ctx.resource.get(remoteURL)
	if err != nil {
		return &errBadRequest{err}
	}
	p := printer.NewURL(remote, opts)
	return convert(ctx, p)
}
