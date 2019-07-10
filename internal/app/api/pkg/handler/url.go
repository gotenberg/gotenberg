package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// URL is the endpoint for converting
// a URL to PDF.
func URL(c echo.Context) error {
	const op string = "handler.URL"
	ctx := context.MustCastFromEchoContext(c)
	ctx.StandardLogger().DebugfOp(op, "url request")
	r := ctx.Resource()
	opts, err := r.ChromePrinterOptions()
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	remoteURL, err := r.Get(resource.RemoteURLFormField)
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	p := printer.NewURL(remoteURL, opts)
	if err := convert(ctx, p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}
