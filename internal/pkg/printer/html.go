package printer

import (
	"fmt"
)

// NewHTML returns an HTML printer.
func NewHTML(fpath string, opts *ChromeOptions) Printer {
	URL := fmt.Sprintf("file://%s", fpath)
	return &chrome{
		url:  URL,
		opts: opts,
	}
}
