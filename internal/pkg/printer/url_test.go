package printer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestURLPrinter(t *testing.T) {
	var (
		logger xlog.Logger = test.DebugLogger()
		config conf.Config = conf.DefaultConfig()
		URL                = "https://google.com"
		opts   ChromePrinterOptions
		dest   string
		p      Printer
		err    error
	)
	// default options.
	opts = DefaultChromePrinterOptions(config)
	p = NewURLPrinter(logger, URL, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// options with a wait delay.
	opts = DefaultChromePrinterOptions(config)
	opts.WaitDelay = 0.5
	p = NewURLPrinter(logger, URL, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// options with a pages ranges.
	opts = DefaultChromePrinterOptions(config)
	opts.PageRanges = "1"
	p = NewURLPrinter(logger, URL, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// should not be OK as options have
	// a wrong page ranges.
	opts = DefaultChromePrinterOptions(config)
	opts.PageRanges = "foo"
	p = NewURLPrinter(logger, URL, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	test.AssertError(t, err)
	assert.Equal(t, xerror.InvalidCode, xerror.Code(err))
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// should not be OK as context.Context
	// should timeout.
	opts = DefaultChromePrinterOptions(config)
	opts.WaitTimeout = 0.0
	p = NewURLPrinter(logger, URL, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	test.AssertError(t, err)
	assert.Equal(t, xerror.TimeoutCode, xerror.Code(err))
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
}
