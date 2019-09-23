package prinery

import (
	"context"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type processSpec interface {
	id() string
	host() string
	port() uint
}

type process interface {
	spec() processSpec
	binary() string
	args() []string
	warmupTime() time.Duration
	viabilityFunc() func(logger xlog.Logger) bool
}

func processesToSpecs(processes []process) []processSpec {
	specs := make([]processSpec, len(processes))
	for i, p := range processes {
		specs[i] = p.spec()
	}
	return specs
}

type printer interface {
	print(ctx context.Context, spec processSpec, dest string) error
}

// ChromePrintOptions helps customizing the
// Google Chrome print result.
type ChromePrintOptions struct {
	WaitDelay    float64
	HeaderHTML   string
	FooterHTML   string
	PaperWidth   float64
	PaperHeight  float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
	Landscape    bool
}

// DefaultChromePrintOptions returns the default
// Google Chrome print options.
func DefaultChromePrintOptions() ChromePrintOptions {
	const defaultHeaderFooterHTML string = "<html><head></head><body></body></html>"
	return ChromePrintOptions{
		WaitDelay:    0.0,
		HeaderHTML:   defaultHeaderFooterHTML,
		FooterHTML:   defaultHeaderFooterHTML,
		PaperWidth:   8.27,
		PaperHeight:  11.7,
		MarginTop:    1.0,
		MarginBottom: 1.0,
		MarginLeft:   1.0,
		MarginRight:  1.0,
		Landscape:    false,
	}
}

// UnoconvPrintOptions helps customizing the
// LibreOffice print result.
type UnoconvPrintOptions struct {
	Landscape bool
}

// DefaultUnoconvPrinterOptions returns the default
// LibreOffice print options.
func DefaultUnoconvPrinterOptions() UnoconvPrintOptions {
	return UnoconvPrintOptions{
		Landscape: false,
	}
}

type Prinery interface {
	Start(emergency chan error) error
	HTML(ctx context.Context, logger xlog.Logger, dest, fpath string, opts ChromePrintOptions) error
	URL(ctx context.Context, logger xlog.Logger, dest, URL string, opts ChromePrintOptions) error
	Markdown(ctx context.Context, logger xlog.Logger, dest, fpath string, opts ChromePrintOptions) error
	Office(ctx context.Context, logger xlog.Logger, dest string, fpaths []string, opts UnoconvPrintOptions) error
	Merge(ctx context.Context, logger xlog.Logger, dest string, fpaths []string) error
}
