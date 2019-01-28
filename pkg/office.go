package gotenberg

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// OfficeRequest facilitates Office documents
// conversion with the Gotenberg API.
type OfficeRequest struct {
	filePaths []string
	values    map[string]string
}

// NewOfficeRequest create OfficeRequest.
func NewOfficeRequest(fpaths ...string) (*OfficeRequest, error) {
	for _, fpath := range fpaths {
		if !fileExists(fpath) {
			return nil, fmt.Errorf("%s: file does not exist", fpath)
		}
	}
	return &OfficeRequest{filePaths: fpaths, values: make(map[string]string)}, nil
}

// SetWebhookURL sets webhookURL form field.
func (office *OfficeRequest) SetWebhookURL(webhookURL string) {
	office.values[webhookURL] = webhookURL
}

// SetLandscape sets landscape form field.
func (office *OfficeRequest) SetLandscape(isLandscape bool) {
	office.values[landscape] = strconv.FormatBool(isLandscape)
}

func (office *OfficeRequest) getPostURL() string {
	return "/convert/office"
}

func (office *OfficeRequest) getFormValues() map[string]string {
	return office.values
}

func (office *OfficeRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	for _, fpath := range office.filePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(OfficeRequest))
	_ = UnoconvRequest(new(OfficeRequest))
)
