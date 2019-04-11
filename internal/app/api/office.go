package api

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func convertOffice(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.officePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	fpaths, err := ctx.resource.fpaths(
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
		return &errBadRequest{err}
	}
	p := printer.NewOffice(fpaths, opts)
	return convert(ctx, p)
}
