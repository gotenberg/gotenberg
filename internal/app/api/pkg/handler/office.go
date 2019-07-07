package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

// Office is the endpoint for converting
// Office files to PDF.
func Office(c echo.Context) error {
	ctx := context.MustCastFromEchoContext(c)
	r := ctx.Resource()
	opts, err := r.OfficePrinterOptions()
	if err != nil {
		return err
	}
	fpaths, err := r.Fpaths(
		".txt",
		".rtf",
		".fodt",
		".doc",
		".docx",
		".odt",
		".xls",
		".xlsx",
		".ods",
		".ppt",
		".pptx",
		".odp",
	)
	if err != nil {
		return err
	}
	p := printer.NewOffice(fpaths, opts)
	return convert(ctx, p)
}
