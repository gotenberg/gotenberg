package printer

// NewURL returns a URL printer.
func NewURL(url string, opts *ChromeOptions) Printer {
	return &chrome{
		url:  url,
		opts: opts,
	}
}
