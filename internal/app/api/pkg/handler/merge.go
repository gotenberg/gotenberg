package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

// Merge is the endpoint for
// merging PDF files.
func Merge(c echo.Context) error {
	ctx := context.MustCastFromEchoContext(c)
	r := ctx.Resource()
	opts, err := r.MergePrinterOptions()
	if err != nil {
		return err
	}
	fpaths, err := r.Fpaths(".pdf")
	if err != nil {
		return err
	}
	p := printer.NewMerge(fpaths, opts)
	return convert(ctx, p)
}
