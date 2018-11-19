package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

func convertHTML(c echo.Context) error {
	r, err := newResource(c)
	if err != nil {
		return err
	}
	defer r.removeAll()
	ctx, cancel := newContext(r)
	defer cancel()
	p := &printer.HTML{Context: ctx}
	indexPath, err := r.filePath("index.html")
	if err != nil {
		return err
	}
	if err := p.WithLocalURL(indexPath); err != nil {
		return err
	}
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
	filename, err := rand.Get()
	if err != nil {
		return err
	}
	filename = fmt.Sprintf("%s.pdf", filename)
	fpath := fmt.Sprintf("%s/%s", r.dirPath, filename)
	if err := p.Print(fpath); err != nil {
		return err
	}
	if r.webhookURL() == "" {
		return c.Attachment(fpath, filename)
	}
	// TODO
	return c.String(http.StatusOK, "Will upload to given webhook!")
}
