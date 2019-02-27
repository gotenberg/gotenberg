package api

import (
	"errors"

	"github.com/labstack/echo"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func repairPDF(c echo.Context) error {
	r, err := newResource(c)
	if err != nil {
		return hijackErr(err, r)
	}
	ctx, cancel := newContext(r)
	if cancel != nil {
		defer cancel()
	}
	fpaths, err := r.filePaths([]string{".pdf"})
	if err != nil {
		return hijackErr(err, r)
	}
	if len(fpaths) == 0 {
		return hijackErr(errors.New("no suitable PDF file to repair"), r)
	}
	p := &printer.Repair{Context: ctx, FilePaths: fpaths}
	return print(c, p, r)
}
