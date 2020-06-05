package printer

import (
	"bytes"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
	"html/template"
	"io/ioutil"
	"path/filepath"
)

// NewMarkdownPrinter returns a Printer which
// is able to convert Markdown files to PDF.
func NewMarkdownPrinter(logger xlog.Logger, fpath string, opts ChromePrinterOptions) (Printer, error) {
	const op string = "printer.NewMarkdownPrinter"
	resolver := func() (string, error) {
		tmpl, err := template.
			New(filepath.Base(fpath)).
			Funcs(template.FuncMap{"toHTML": markdownToHTML}).
			ParseFiles(fpath)
		if err != nil {
			return "", err
		}
		dirPath := filepath.Dir(fpath)
		data := &templateData{DirPath: dirPath}
		logger.DebugOp(op, "converting Markdown files to HTML...")
		var buffer bytes.Buffer
		if err := tmpl.Execute(&buffer, data); err != nil {
			return "", err
		}
		baseFilename := xrand.Get()
		dst := fmt.Sprintf("%s/%s.html", dirPath, baseFilename)
		logger.DebugOp(op, "writing the HTML from previous conversion(s) into new file...")
		if err := ioutil.WriteFile(dst, buffer.Bytes(), 0600); err != nil {
			return "", err
		}
		return fmt.Sprintf("file://%s", dst), nil
	}
	URL, err := resolver()
	if err != nil {
		return chromePrinter{}, xerror.New(op, err)
	}
	return chromePrinter{
		logger: logger,
		url:    URL,
		opts:   opts,
	}, nil
}

type templateData struct {
	DirPath string
}

func markdownToHTML(dirPath, filename string) (template.HTML, error) {
	const op string = "printer.markdownToHTML"
	// avoid directory traversal.
	filename = filepath.Base(filename)
	fpath := fmt.Sprintf("%s/%s", dirPath, filename)
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", xerror.New(op, err)
	}
	unsafe := blackfriday.Run(b)
	content := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	/* #nosec */
	return template.HTML(content), nil
}
