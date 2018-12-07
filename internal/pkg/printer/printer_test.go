package printer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestMerge(t *testing.T) {
	err := Merge(
		[]string{
			test.PDFTestFilePath(t, "gotenberg.pdf"),
			test.PDFTestFilePath(t, "gotenberg.pdf"),
		},
		"foo.pdf",
	)
	require.Nil(t, err)
	require.FileExists(t, "foo.pdf")
	err = os.Remove("foo.pdf")
	assert.Nil(t, err)
}
