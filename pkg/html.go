package gotenberg

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// HTMLRequest facilitates HTML conversion
// with the Gotenberg API.
type HTMLRequest struct {
	indexFilePath  string
	headerFilePath string
	footerFilePath string
	assetFilePaths []string
	values         map[string]string
}

// NewHTMLRequest create HTMLRequest.
func NewHTMLRequest(indexFilePath string) (*HTMLRequest, error) {
	if !fileExists(indexFilePath) {
		return nil, fmt.Errorf("%s: index file does not exist", indexFilePath)
	}
	return &HTMLRequest{indexFilePath: indexFilePath, values: make(map[string]string)}, nil
}

// SetWebhookURL sets webhookURL form field.
func (html *HTMLRequest) SetWebhookURL(webhookURL string) {
	html.values[webhookURL] = webhookURL
}

// SetHeader sets header form file.
func (html *HTMLRequest) SetHeader(fpath string) error {
	if !fileExists(fpath) {
		return fmt.Errorf("%s: header file does not exist", fpath)
	}
	html.headerFilePath = fpath
	return nil
}

// SetFooter sets footer form file.
func (html *HTMLRequest) SetFooter(fpath string) error {
	if !fileExists(fpath) {
		return fmt.Errorf("%s: footer file does not exist", fpath)
	}
	html.footerFilePath = fpath
	return nil
}

// SetAssets sets assets form files.
func (html *HTMLRequest) SetAssets(fpaths ...string) error {
	for _, fpath := range fpaths {
		if !fileExists(fpath) {
			return fmt.Errorf("%s: file does not exist", fpath)
		}
	}
	html.assetFilePaths = fpaths
	return nil
}

// SetPaperSize sets paperWidth and paperHeight form fields.
func (html *HTMLRequest) SetPaperSize(size [2]float64) {
	html.values[paperWidth] = fmt.Sprintf("%f", size[0])
	html.values[paperHeight] = fmt.Sprintf("%f", size[1])
}

// SetMargins sets marginTop, marginBottom,
// marginLeft and marginRight form fields.
func (html *HTMLRequest) SetMargins(margins [4]float64) {
	html.values[marginTop] = fmt.Sprintf("%f", margins[0])
	html.values[marginBottom] = fmt.Sprintf("%f", margins[1])
	html.values[marginLeft] = fmt.Sprintf("%f", margins[2])
	html.values[marginRight] = fmt.Sprintf("%f", margins[3])
}

// SetLandscape sets landscape form field.
func (html *HTMLRequest) SetLandscape(isLandscape bool) {
	html.values[landscape] = strconv.FormatBool(isLandscape)
}

// SetWebFontsTimeout sets webFontsTimeout form field.
func (html *HTMLRequest) SetWebFontsTimeout(timeout int64) {
	html.values[webFontsTimeout] = strconv.FormatInt(timeout, 10)
}

func (html *HTMLRequest) getPostURL() string {
	return "/convert/html"
}

func (html *HTMLRequest) getFormValues() map[string]string {
	return html.values
}

func (html *HTMLRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	files["index.html"] = html.indexFilePath
	files["header.html"] = html.headerFilePath
	files["footer.html"] = html.footerFilePath
	for _, fpath := range html.assetFilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(HTMLRequest))
	_ = ChromeRequest(new(HTMLRequest))
)
