package printer

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"github.com/thecodingmachine/gotenberg/internal/pkg/file"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

// NewMarkdown returns a markdown printer.
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
	if err := file.WriteBytesToFile(dst, buffer.Bytes()); err != nil {
		return nil, err
	}
	URL := fmt.Sprintf("file://%s", dst)
	return newChrome(URL, opts)
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
	contentHTML := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return template.HTML(contentHTML), nil
}
