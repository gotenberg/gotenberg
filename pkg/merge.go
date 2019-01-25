package gotenberg

import (
	"fmt"
	"path/filepath"
)

// MergeRequest facilitates merging PDF
// with the Gotenberg API.
type MergeRequest struct {
	filePaths []string
	values    map[string]string
}

// NewMergeRequest create MergeRequest.
func NewMergeRequest(fpaths ...string) (*MergeRequest, error) {
	for _, fpath := range fpaths {
		if !fileExists(fpath) {
			return nil, fmt.Errorf("%s: file does not exist", fpath)
		}
	}
	return &MergeRequest{filePaths: fpaths, values: make(map[string]string)}, nil
}

// SetWebhookURL sets webhookURL form field.
func (merge *MergeRequest) SetWebhookURL(webhookURL string) {
	merge.values[webhookURL] = webhookURL
}

func (merge *MergeRequest) getPostURL() string {
	return "/merge"
}

func (merge *MergeRequest) getFormValues() map[string]string {
	return merge.values
}

func (merge *MergeRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	for _, fpath := range merge.filePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(MergeRequest))
)
