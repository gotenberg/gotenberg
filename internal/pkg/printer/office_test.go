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

func TestOfficePrinter(t *testing.T) {
	var (
		logger xlog.Logger = test.DebugLogger()
		config conf.Config = conf.DefaultConfig()
		fpaths []string    = test.OfficeFpaths(t)
		opts   OfficePrinterOptions
		dest   string
		p      Printer
		err    error
	)
	// default options.
	opts = DefaultOfficePrinterOptions(config)
	p = NewOfficePrinter(logger, fpaths, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// using one file.
	opts = DefaultOfficePrinterOptions(config)
	p = NewOfficePrinter(logger, []string{fpaths[0]}, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// options with landscape.
	opts = DefaultOfficePrinterOptions(config)
	opts.Landscape = true
	p = NewOfficePrinter(logger, fpaths, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// options with page ranges.
	opts = DefaultOfficePrinterOptions(config)
	opts.PageRanges = "1-1"
	p = NewOfficePrinter(logger, []string{fpaths[0]}, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// should not be OK as options have
	// a wrong page ranges.
	opts = DefaultOfficePrinterOptions(config)
	opts.PageRanges = "foo"
	p = NewOfficePrinter(logger, []string{fpaths[0]}, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	test.AssertError(t, err)
	assert.Equal(t, xerror.InvalidCode, xerror.Code(err))
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// should not be OK as context.Context
	// should timeout.
	opts = DefaultOfficePrinterOptions(config)
	opts.WaitTimeout = 0.0
	p = NewOfficePrinter(logger, fpaths, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	test.AssertError(t, err)
	assert.Equal(t, xerror.TimeoutCode, xerror.Code(err))
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
}
