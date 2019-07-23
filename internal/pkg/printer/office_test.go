package printer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/printertest"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/xerrortest"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/xlogtest"
)

func TestOfficePrinter(t *testing.T) {
	var (
		logger xlog.Logger = xlogtest.DebugLogger()
		fpaths []string    = printertest.OfficeFpaths(t)
		opts   OfficePrinterOptions
		dest   string
		p      Printer
		err    error
	)
	// default options.
	opts = OfficePrinterOptions{
		WaitTimeout: 10.0,
		Landscape:   false,
	}
	p = NewOfficePrinter(logger, fpaths, opts)
	dest = printertest.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// options with landscape.
	opts = OfficePrinterOptions{
		WaitTimeout: 10.0,
		Landscape:   true,
	}
	p = NewOfficePrinter(logger, fpaths, opts)
	dest = printertest.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// should not be OK as context.Context
	// should timeout.
	opts = OfficePrinterOptions{
		WaitTimeout: 1.0,
		Landscape:   true,
	}
	p = NewOfficePrinter(logger, fpaths, opts)
	dest = printertest.GenerateDestination()
	err = p.Print(dest)
	xerrortest.AssertError(t, err)
	assert.Equal(t, xerror.TimeoutCode, xerror.Code(err))
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
}
