package api

import (
	"github.com/labstack/echo"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

func convertMarkdown(c echo.Context) error {
	r, err := newResource(c)
	if err != nil {
		return err
	}
	defer r.removeAll()
	ctx, cancel := newContext(r)
	if cancel != nil {
		defer cancel()
	}
	indexPath, err := r.filePath("index.html")
	if err != nil {
		return err
	}
	p := &printer.Markdown{Context: ctx, TemplatePath: indexPath}
	headerPath, _ := r.filePath("header.html")
	if err := p.WithHeaderFile(headerPath); err != nil {
		return err
	}
	footerPath, _ := r.filePath("footer.html")
	if err := p.WithFooterFile(footerPath); err != nil {
		return err
	}
	paperSize, err := r.paperSize()
	if err != nil {
		return err
	}
	p.PaperWidth = paperSize[0]
	p.PaperHeight = paperSize[1]
	paperMargins, err := r.paperMargins()
	if err != nil {
		return err
	}
	p.MarginTop = paperMargins[0]
	p.MarginBottom = paperMargins[1]
	p.MarginLeft = paperMargins[2]
	p.MarginRight = paperMargins[3]
	landscape, err := r.landscape()
	if err != nil {
		return err
	}
	p.Landscape = landscape
	return print(c, p, r)
}
