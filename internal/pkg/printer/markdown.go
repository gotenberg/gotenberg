package printer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

// Markdown facilitates Markdown to PDF conversion.
type Markdown struct {
	Context      context.Context
	TemplatePath string
	HeaderHTML   string
	FooterHTML   string
	PaperWidth   float64
	PaperHeight  float64

	html *HTML
}

// Print converts markdown to PDF.
func (md *Markdown) Print(destination string) error {
	if md.html == nil {
		md.html = &HTML{Context: md.Context}
	}
	if md.HeaderHTML != "" {
		md.html.HeaderHTML = md.HeaderHTML
	}
	if md.FooterHTML != "" {
		md.html.FooterHTML = md.FooterHTML
	}
	md.html.PaperWidth = md.PaperWidth
	md.html.PaperHeight = md.PaperHeight
	tmpl, err := template.
		New(filepath.Base(md.TemplatePath)).
		Funcs(template.FuncMap{"markdonify": markdonify}).
		ParseFiles(md.TemplatePath)
	if err != nil {
		return fmt.Errorf("%s: parsing template: %v", md.TemplatePath, err)
	}
	var data bytes.Buffer
	if err := tmpl.Execute(&data, nil); err != nil {
		return fmt.Errorf("%s: executing template: %v", md.TemplatePath, err)
	}
	dst := fmt.Sprintf("%s/markdown.html", filepath.Base(md.TemplatePath))
	if err := writeBytesToFile(dst, data.Bytes()); err != nil {
		return err
	}
	if err := md.html.WithLocalURL(dst); err != nil {
		return err
	}
	return md.html.Print(destination)
}

func markdonify(filename string) (template.HTML, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("%s: reading file: %v", filename, err)
	}
	unsafe := blackfriday.Run(b)
	contentHTML := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return template.HTML(contentHTML), nil
}

// WithHeaderFile sets header content from a file.
func (md *Markdown) WithHeaderFile(fpath string) error {
	if md.html == nil {
		md.html = &HTML{Context: md.Context}
	}
	return md.html.WithHeaderFile(fpath)
}

// WithFooterFile sets footer content from a file.
func (md *Markdown) WithFooterFile(fpath string) error {
	if md.html == nil {
		md.html = &HTML{Context: md.Context}
	}
	return md.html.WithFooterFile(fpath)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(Markdown))
)
