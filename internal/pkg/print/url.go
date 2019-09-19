package print

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

// NewURLPrint returns a Print for
// converting a URL to PDF.
func NewURLPrint(logger xlog.Logger, url string, opts ChromePrintOptions) Print {
	return chromePrint{
		logger: logger,
		url:    url,
		opts:   opts,
	}
}
