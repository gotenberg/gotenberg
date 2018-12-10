package gotenberg

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// MarkdownRequest facilitates Markdown conversion
// with the Gotenberg API.
type MarkdownRequest struct {
	IndexFilePath     string
	MarkdownFilePaths []string
	AssetFilePaths    []string
	Options           *MarkdownOptions
}

// MarkdownOptions gathers all options
// for Markdown conversion with the Gotenberg API.
type MarkdownOptions struct {
	WebHookURL     string
	HeaderFilePath string
	FooterFilePath string
	PaperSize      [2]float64
	PaperMargins   [4]float64
	Landscape      bool
}

func (markdown *MarkdownRequest) validate() error {
	if !fileExists(markdown.IndexFilePath) {
		return fmt.Errorf("%s: index file does not exist", markdown.IndexFilePath)
	}
	if markdown.Options.HeaderFilePath != "" && !fileExists(markdown.Options.HeaderFilePath) {
		return fmt.Errorf("%s: header file does not exist", markdown.Options.HeaderFilePath)
	}
	if markdown.Options.FooterFilePath != "" && !fileExists(markdown.Options.FooterFilePath) {
		return fmt.Errorf("%s: footer file does not exist", markdown.Options.FooterFilePath)
	}
	for _, fpath := range markdown.MarkdownFilePaths {
		if !fileExists(fpath) {
			return fmt.Errorf("%s: markdown file does not exist", fpath)
		}
	}
	return nil
}

func (markdown *MarkdownRequest) getPostURL() string {
	return "/convert/markdown"
}

func (markdown *MarkdownRequest) getFormValues() map[string]string {
	if markdown.Options == nil {
		markdown.Options = &MarkdownOptions{}
	}
	values := make(map[string]string)
	values[webhookURL] = markdown.Options.WebHookURL
	values[paperWidth] = fmt.Sprintf("%f", markdown.Options.PaperSize[0])
	values[paperHeight] = fmt.Sprintf("%f", markdown.Options.PaperSize[1])
	values[marginTop] = fmt.Sprintf("%f", markdown.Options.PaperMargins[0])
	values[marginBottom] = fmt.Sprintf("%f", markdown.Options.PaperMargins[1])
	values[marginLeft] = fmt.Sprintf("%f", markdown.Options.PaperMargins[2])
	values[marginRight] = fmt.Sprintf("%f", markdown.Options.PaperMargins[3])
	values[landscape] = strconv.FormatBool(markdown.Options.Landscape)
	return values
}

func (markdown *MarkdownRequest) getFormFiles() map[string]string {
	if markdown.Options == nil {
		markdown.Options = &MarkdownOptions{}
	}
	files := make(map[string]string)
	files["index.html"] = markdown.IndexFilePath
	files["header.html"] = markdown.Options.HeaderFilePath
	files["footer.html"] = markdown.Options.FooterFilePath
	for _, fpath := range markdown.MarkdownFilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	for _, fpath := range markdown.AssetFilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(MarkdownRequest))
)
