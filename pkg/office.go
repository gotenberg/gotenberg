package gotenberg

import (
	"fmt"
	"path/filepath"
)

// OfficeRequest facilitates Office documents
// conversion with the Gotenberg API.
type OfficeRequest struct {
	FilePaths []string
	Options   *OfficeOptions
}

// OfficeOptions gathers all options
// for Office documents conversion
// with the Gotenberg API.
type OfficeOptions struct {
	WebHookURL string
}

func (office *OfficeRequest) validate() error {
	for _, fpath := range office.FilePaths {
		if !fileExists(fpath) {
			return fmt.Errorf("%s: office file does not exist", fpath)
		}
	}
	return nil
}

func (office *OfficeRequest) getPostURL() string {
	return "/convert/office"
}

func (office *OfficeRequest) getFormValues() map[string]string {
	if office.Options == nil {
		office.Options = &OfficeOptions{}
	}
	values := make(map[string]string)
	values[webhookURL] = office.Options.WebHookURL
	return values
}

func (office *OfficeRequest) getFormFiles() map[string]string {
	files := make(map[string]string)
	for _, fpath := range office.FilePaths {
		files[filepath.Base(fpath)] = fpath
	}
	return files
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Request(new(OfficeRequest))
)
