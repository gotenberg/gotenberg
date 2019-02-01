package gotenberg

import (
	"fmt"
	"strconv"
)

// URLRequest facilitates remote URL conversion
// with the Gotenberg API.
type URLRequest struct {
	headerFilePath string
	footerFilePath string
	values         map[string]string
}

// NewURLRequest create URLRequest.
func NewURLRequest(URL string) *URLRequest {
	req := &URLRequest{values: make(map[string]string)}
	req.values[remoteURL] = URL
	return req
}

// SetWebhookURL sets webhookURL form field.
func (url *URLRequest) SetWebhookURL(webhookURL string) {
	url.values[webhookURL] = webhookURL
}

// SetHeader sets header form file.
func (url *URLRequest) SetHeader(fpath string) error {
	if !fileExists(fpath) {
		return fmt.Errorf("%s: header file does not exist", fpath)
	}
	url.headerFilePath = fpath
	return nil
}

// SetFooter sets footer form file.
func (url *URLRequest) SetFooter(fpath string) error {
	if !fileExists(fpath) {
		return fmt.Errorf("%s: footer file does not exist", fpath)
	}
	url.footerFilePath = fpath
	return nil
}

// SetPaperSize sets paperWidth and paperHeight form fields.
func (url *URLRequest) SetPaperSize(size [2]float64) {
	url.values[paperWidth] = fmt.Sprintf("%f", size[0])
	url.values[paperHeight] = fmt.Sprintf("%f", size[1])
}

// SetMargins sets marginTop, marginBottom,
// marginLeft and marginRight form fields.
func (url *URLRequest) SetMargins(margins [4]float64) {
	url.values[marginTop] = fmt.Sprintf("%f", margins[0])
	url.values[marginBottom] = fmt.Sprintf("%f", margins[1])
	url.values[marginLeft] = fmt.Sprintf("%f", margins[2])
	url.values[marginRight] = fmt.Sprintf("%f", margins[3])
}

// SetLandscape sets landscape form field.
func (url *URLRequest) SetLandscape(isLandscape bool) {
	url.values[landscape] = strconv.FormatBool(isLandscape)
}

// SetWebFontsTimeout sets webFontsTimeout form field.
func (url *URLRequest) SetWebFontsTimeout(timeout int64) {
	url.values[webFontsTimeout] = strconv.FormatInt(timeout, 10)
}

func (url *URLRequest) getPostURL() string {
	return "/convert/url"
}

func (url *URLRequest) getFormValues() map[string]string {
	return url.values
}

func (url *URLRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	files["header.html"] = url.headerFilePath
	files["footer.html"] = url.footerFilePath
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(URLRequest))
	_ = ChromeRequest(new(URLRequest))
)
