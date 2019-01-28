package api

import (
	"github.com/labstack/echo"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func convertURL(c echo.Context) error {
	r, err := newResource(c)
	if err != nil {
		return hijackErr(err, r)
	}
	ctx, cancel := newContext(r)
	if cancel != nil {
		defer cancel()
	}
	p := &printer.HTML{Context: ctx}
	URL, err := r.remoteURL()
	if err != nil {
		return hijackErr(err, r)
	}
	p.URL = URL
	headerPath, _ := r.filePath("header.html")
	if err := p.WithHeaderFile(headerPath); err != nil {
		return hijackErr(err, r)
	}
	footerPath, _ := r.filePath("footer.html")
	if err := p.WithFooterFile(footerPath); err != nil {
		return hijackErr(err, r)
	}
	paperSize, err := r.paperSize()
	if err != nil {
		return hijackErr(err, r)
	}
	p.PaperWidth = paperSize[0]
	p.PaperHeight = paperSize[1]
	paperMargins, err := r.paperMargins()
	if err != nil {
		return hijackErr(err, r)
	}
	p.MarginTop = paperMargins[0]
	p.MarginBottom = paperMargins[1]
	p.MarginLeft = paperMargins[2]
	p.MarginRight = paperMargins[3]
	landscape, err := r.landscape()
	if err != nil {
		return hijackErr(err, r)
	}
	p.Landscape = landscape
	return print(c, p, r)
}
