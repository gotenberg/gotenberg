package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

// URL is the endpoint for converting
// a URL to PDF.
func URL(c echo.Context) error {
	ctx := context.MustCastFromEchoContext(c)
	r := ctx.Resource()
	opts, err := r.ChromePrinterOptions()
	if err != nil {
		return err
	}
	remoteURL, err := r.Get(resource.RemoteURLFormField)
	if err != nil {
		return err
	}
	p := printer.NewURL(remoteURL, opts)
	return convert(ctx, p)
}
