package api

import (
	"errors"

	"github.com/labstack/echo"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

var officeExts = []string{
	".txt",
	".rtf",
	".doc",
	".docx",
	".odt",
	".xls",
	".xlsx",
	".ods",
	".ppt",
	".pptx",
	".odp",
}

func convertOffice(c echo.Context) error {
	r, err := newResource(c)
	if err != nil {
		return hijackErr(err, r)
	}
	ctx, cancel := newContext(r)
	if cancel != nil {
		defer cancel()
	}
	fpaths, err := r.filePaths(officeExts)
	if err != nil {
		return hijackErr(err, r)
	}
	if len(fpaths) == 0 {
		return hijackErr(errors.New("no suitable office documents to convert"), r)
	}
	p := &printer.Office{Context: ctx, FilePaths: fpaths}
	paperSize, err := r.paperSize()
	if err != nil {
		return hijackErr(err, r)
	}
	p.PaperWidth = paperSize[0]
	p.PaperHeight = paperSize[1]
	landscape, err := r.landscape()
	if err != nil {
		return hijackErr(err, r)
	}
	p.Landscape = landscape
	return print(c, p, r)
}
