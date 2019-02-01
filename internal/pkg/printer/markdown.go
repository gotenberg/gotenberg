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
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

// Markdown facilitates Markdown to PDF conversion.
type Markdown struct {
	Context         context.Context
	TemplatePath    string
	HeaderHTML      string
	FooterHTML      string
	PaperWidth      float64
	PaperHeight     float64
	MarginTop       float64
	MarginBottom    float64
	MarginLeft      float64
	MarginRight     float64
	Landscape       bool
	WebFontsTimeout int64

	html *HTML
}

type templateData struct {
	DirPath string
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
	md.html.MarginTop = md.MarginTop
	md.html.MarginBottom = md.MarginBottom
	md.html.MarginLeft = md.MarginLeft
	md.html.MarginRight = md.MarginRight
	md.html.Landscape = md.Landscape
	md.html.WebFontsTimeout = md.WebFontsTimeout
	tmpl, err := template.
		New(filepath.Base(md.TemplatePath)).
		Funcs(template.FuncMap{"toHTML": toHTML}).
		ParseFiles(md.TemplatePath)
	if err != nil {
		return fmt.Errorf("%s: parsing template: %v", md.TemplatePath, err)
	}
	dirPath := filepath.Dir(md.TemplatePath)
	data := &templateData{DirPath: dirPath}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return fmt.Errorf("%s: executing template: %v", md.TemplatePath, err)
	}
	baseFilename, err := rand.Get()
	if err != nil {
		return err
	}

	dst := fmt.Sprintf("%s/%s.html", dirPath, baseFilename)
	if err := writeBytesToFile(dst, buffer.Bytes()); err != nil {
		return err
	}
	md.html.WithLocalURL(dst)
	return md.html.Print(destination)
}

func toHTML(dirPath, filename string) (template.HTML, error) {
	fpath := fmt.Sprintf("%s/%s", dirPath, filename)
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", fmt.Errorf("%s: reading file: %v", fpath, err)
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
