package gotenberg

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// HTMLRequest facilitates HTML conversion
// with the Gotenberg API.
type HTMLRequest struct {
	IndexFilePath  string
	AssetFilePaths []string
	Options        *HTMLOptions
}

// HTMLOptions gathers all options
// for HTML conversion with the Gotenberg API.
type HTMLOptions struct {
	WebHookURL     string
	HeaderFilePath string
	FooterFilePath string
	PaperSize      [2]float64
	PaperMargins   [4]float64
	Landscape      bool
}

func (html *HTMLRequest) validate() error {
	if !fileExists(html.IndexFilePath) {
		return fmt.Errorf("%s: index file does not exist", html.IndexFilePath)
	}
	if html.Options.HeaderFilePath != "" && !fileExists(html.Options.HeaderFilePath) {
		return fmt.Errorf("%s: header file does not exist", html.Options.HeaderFilePath)
	}
	if html.Options.FooterFilePath != "" && !fileExists(html.Options.FooterFilePath) {
		return fmt.Errorf("%s: footer file does not exist", html.Options.FooterFilePath)
	}
	return nil
}

func (html *HTMLRequest) getPostURL() string {
	return "/convert/html"
}

func (html *HTMLRequest) getFormValues() map[string]string {
	if html.Options == nil {
		html.Options = &HTMLOptions{}
	}
	values := make(map[string]string)
	values[webhookURL] = html.Options.WebHookURL
	values[paperWidth] = fmt.Sprintf("%f", html.Options.PaperSize[0])
	values[paperHeight] = fmt.Sprintf("%f", html.Options.PaperSize[1])
	values[marginTop] = fmt.Sprintf("%f", html.Options.PaperMargins[0])
	values[marginBottom] = fmt.Sprintf("%f", html.Options.PaperMargins[1])
	values[marginLeft] = fmt.Sprintf("%f", html.Options.PaperMargins[2])
	values[marginRight] = fmt.Sprintf("%f", html.Options.PaperMargins[3])
	values[landscape] = strconv.FormatBool(html.Options.Landscape)
	return values
}

func (html *HTMLRequest) getFormFiles() map[string]string {
	if html.Options == nil {
		html.Options = &HTMLOptions{}
	}
	files := make(map[string]string)
	files["index.html"] = html.IndexFilePath
	files["header.html"] = html.Options.HeaderFilePath
	files["footer.html"] = html.Options.FooterFilePath
	for _, fpath := range html.AssetFilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(HTMLRequest))
)
