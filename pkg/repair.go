package gotenberg

import (
	"fmt"
	"path/filepath"
)

// PdfRepairRequest try to fix damaged PDF
// with the Gotenberg API.
type PdfRepairRequest struct {
	filePaths []string
	values    map[string]string
}

// NewPdfRepairRequest create PdfRepairRequest.
func NewPdfRepairRequest(fpaths string) (*PdfRepairRequest, error) {
	if !fileExists(fpath) {
		return nil, fmt.Errorf("%s: file does not exist", fpath)
	}

	return &PdfRepairRequest{filePaths: fpath, values: make(map[string]string)}, nil
}

// SetWebhookURL sets webhookURL form field.
func (r *PdfRepairRequest) SetWebhookURL(webhookURL string) {
	r.values[webhookURL] = webhookURL
}

func (r *PdfRepairRequest) getPostURL() string {
	return "/repair"
}

func (r *PdfRepairRequest) getFormValues() map[string]string {
	return r.values
}

func (r *PdfRepairRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	for _, fpath := range r.filePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(PdfRepairRequest))
)
