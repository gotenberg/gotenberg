package resource

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestHeaderFooterContents(t *testing.T) {
	const resourceDirectoryName string = "foo"
	var expected string
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	opts := printer.DefaultChromePrinterOptions(config)
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// arguments do not exist.
	header, footer, err := HeaderFooterContents(r, config)
	assert.Nil(t, err)
	assert.Equal(t, opts.HeaderHTML, header)
	assert.Equal(t, opts.FooterHTML, footer)
	// arguments exist.
	expected = "Gutenberg"
	fpath := test.OfficeFpaths(t)[2]
	f1, err := os.Open(fpath)
	assert.Nil(t, err)
	defer f1.Close() // nolint: errcheck
	err = r.WithFile("header.html", f1)
	assert.Nil(t, err)
	f2, err := os.Open(fpath)
	assert.Nil(t, err)
	defer f2.Close() // nolint: errcheck
	err = r.WithFile("footer.html", f2)
	assert.Nil(t, err)
	header, footer, err = HeaderFooterContents(r, config)
	assert.Nil(t, err)
	assert.Contains(t, header, expected)
	assert.Contains(t, footer, expected)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}
