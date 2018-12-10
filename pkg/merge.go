package gotenberg

import (
	"fmt"
	"path/filepath"
)

// MergeRequest facilitates merging PDF
// with the Gotenberg API.
type MergeRequest struct {
	FilePaths []string
	Options   *MergeOptions
}

// MergeOptions gathers all options
// for merging PDF
// with the Gotenberg API.
type MergeOptions struct {
	WebHookURL string
}

func (merge *MergeRequest) validate() error {
	for _, fpath := range merge.FilePaths {
		if !fileExists(fpath) {
			return fmt.Errorf("%s: PDF file does not exist", fpath)
		}
	}
	return nil
}

func (merge *MergeRequest) getPostURL() string {
	return "/merge"
}

func (merge *MergeRequest) getFormValues() map[string]string {
	if merge.Options == nil {
		merge.Options = &MergeOptions{}
	}
	values := make(map[string]string)
	values[webhookURL] = merge.Options.WebHookURL
	return values
}

func (merge *MergeRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	for _, fpath := range merge.FilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(MergeRequest))
)
