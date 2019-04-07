package printer

// NewURL returns a URL printer.
func NewURL(URL string, opts *ChromeOptions) (Printer, error) {
	return newChrome(URL, opts)
}
