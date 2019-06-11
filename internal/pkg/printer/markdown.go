package printer

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

// NewMarkdown returns a Markdown printer.
func NewMarkdown(fpath string, opts *ChromeOptions) (Printer, error) {
	tmpl, err := template.
		New(filepath.Base(fpath)).
		Funcs(template.FuncMap{"toHTML": markdownToHTML}).
		ParseFiles(fpath)
	if err != nil {
		return nil, fmt.Errorf("%s: parsing template: %v", fpath, err)
	}
	dirPath := filepath.Dir(fpath)
	data := &templateData{DirPath: dirPath}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("%s: executing template: %v", fpath, err)
	}
	baseFilename, err := rand.Get()
	if err != nil {
		return nil, err
	}
	dst := fmt.Sprintf("%s/%s.html", dirPath, baseFilename)
	if err := ioutil.WriteFile(dst, buffer.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("%s: writing file: %v", dst, err)
	}
	URL := fmt.Sprintf("file://%s", dst)
	return &chrome{
		url:  URL,
		opts: opts,
	}, nil
}

type templateData struct {
	DirPath string
}

func markdownToHTML(dirPath, filename string) (template.HTML, error) {
	fpath := fmt.Sprintf("%s/%s", dirPath, filename)
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", fmt.Errorf("%s: reading file: %v", fpath, err)
	}
	unsafe := blackfriday.Run(b)
	content := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	/* #nosec */
	return template.HTML(content), nil
}
