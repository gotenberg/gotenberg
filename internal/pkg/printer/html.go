package printer

import (
	"fmt"
)

// NewHTML returns an HTML printer.
func NewHTML(fpath string, opts *ChromeOptions) (Printer, error) {
	URL := fmt.Sprintf("file://%s", fpath)
	return newChrome(URL, opts)
}
