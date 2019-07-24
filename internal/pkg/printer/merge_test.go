package printer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestMergePrinter(t *testing.T) {
	var (
		logger xlog.Logger = test.DebugLogger()
		fpaths []string    = test.MergeFpaths(t)
		opts   MergePrinterOptions
		dest   string
		p      Printer
		err    error
	)
	// default options.
	opts = MergePrinterOptions{
		WaitTimeout: 10.0,
	}
	p = NewMergePrinter(logger, fpaths, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	assert.Nil(t, err)
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
	// should not be OK as context.Context
	// should timeout.
	opts = MergePrinterOptions{
		WaitTimeout: 0.0,
	}
	p = NewMergePrinter(logger, fpaths, opts)
	dest = test.GenerateDestination()
	err = p.Print(dest)
	test.AssertError(t, err)
	assert.Equal(t, xerror.TimeoutCode, xerror.Code(err))
	err = os.RemoveAll(dest)
	assert.Nil(t, err)
}
