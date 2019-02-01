package gotenberg

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// MarkdownRequest facilitates Markdown conversion
// with the Gotenberg API.
type MarkdownRequest struct {
	indexFilePath     string
	markdownFilePaths []string
	headerFilePath    string
	footerFilePath    string
	assetFilePaths    []string
	values            map[string]string
}

// NewMarkdownRequest create MarkdownRequest.
func NewMarkdownRequest(indexFilePath string, markdownFilePaths ...string) (*MarkdownRequest, error) {
	if !fileExists(indexFilePath) {
		return nil, fmt.Errorf("%s: index file does not exist", indexFilePath)
	}
	for _, fpath := range markdownFilePaths {
		if !fileExists(fpath) {
			return nil, fmt.Errorf("%s: markdown file does not exist", fpath)
		}
	}
	return &MarkdownRequest{
		indexFilePath:     indexFilePath,
		markdownFilePaths: markdownFilePaths,
		values:            make(map[string]string),
	}, nil
}

// SetWebhookURL sets webhookURL form field.
func (markdown *MarkdownRequest) SetWebhookURL(webhookURL string) {
	markdown.values[webhookURL] = webhookURL
}

// SetHeader sets header form file.
func (markdown *MarkdownRequest) SetHeader(fpath string) error {
	if !fileExists(fpath) {
		return fmt.Errorf("%s: header file does not exist", fpath)
	}
	markdown.headerFilePath = fpath
	return nil
}

// SetFooter sets footer form file.
func (markdown *MarkdownRequest) SetFooter(fpath string) error {
	if !fileExists(fpath) {
		return fmt.Errorf("%s: footer file does not exist", fpath)
	}
	markdown.footerFilePath = fpath
	return nil
}

// SetAssets sets assets form files.
func (markdown *MarkdownRequest) SetAssets(fpaths ...string) error {
	for _, fpath := range fpaths {
		if !fileExists(fpath) {
			return fmt.Errorf("%s: file does not exist", fpath)
		}
	}
	markdown.assetFilePaths = fpaths
	return nil
}

// SetPaperSize sets paperWidth and paperHeight form fields.
func (markdown *MarkdownRequest) SetPaperSize(size [2]float64) {
	markdown.values[paperWidth] = fmt.Sprintf("%f", size[0])
	markdown.values[paperHeight] = fmt.Sprintf("%f", size[1])
}

// SetMargins sets marginTop, marginBottom,
// marginLeft and marginRight form fields.
func (markdown *MarkdownRequest) SetMargins(margins [4]float64) {
	markdown.values[marginTop] = fmt.Sprintf("%f", margins[0])
	markdown.values[marginBottom] = fmt.Sprintf("%f", margins[1])
	markdown.values[marginLeft] = fmt.Sprintf("%f", margins[2])
	markdown.values[marginRight] = fmt.Sprintf("%f", margins[3])
}

// SetLandscape sets landscape form field.
func (markdown *MarkdownRequest) SetLandscape(isLandscape bool) {
	markdown.values[landscape] = strconv.FormatBool(isLandscape)
}

// SetWebFontsTimeout sets webFontsTimeout form field.
func (markdown *MarkdownRequest) SetWebFontsTimeout(timeout int64) {
	markdown.values[webFontsTimeout] = strconv.FormatInt(timeout, 10)
}

func (markdown *MarkdownRequest) getPostURL() string {
	return "/convert/markdown"
}

func (markdown *MarkdownRequest) getFormValues() map[string]string {
	return markdown.values
}

func (markdown *MarkdownRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	files["index.html"] = markdown.indexFilePath
	files["header.html"] = markdown.headerFilePath
	files["footer.html"] = markdown.footerFilePath
	for _, fpath := range markdown.markdownFilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	for _, fpath := range markdown.assetFilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(MarkdownRequest))
	_ = ChromeRequest(new(MarkdownRequest))
)
