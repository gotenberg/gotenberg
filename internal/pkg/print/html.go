package print

import (
	"fmt"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

// NewHTMLPrint returns a Print for
// converting an HTML file to PDF.
func NewHTMLPrint(logger xlog.Logger, fpath string, opts ChromePrintOptions) Print {
	URL := fmt.Sprintf("file://%s", fpath)
	return chromePrint{
		logger: logger,
		url:    URL,
		opts:   opts,
	}
}
