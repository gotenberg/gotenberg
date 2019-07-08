package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/api/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

// Office is the endpoint for converting
// Office files to PDF.
func Office(c echo.Context) error {
	const op = "handler.Office"
	ctx := context.MustCastFromEchoContext(c)
	r := ctx.Resource()
	opts, err := r.OfficePrinterOptions()
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
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
		return &standarderror.Error{Op: op, Err: err}
	}
	p := printer.NewOffice(fpaths, opts)
	if err := convert(ctx, p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}
