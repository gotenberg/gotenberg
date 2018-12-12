package printer

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestMerge(t *testing.T) {
	dirPath := test.PDFTestDirPath(t)
	dst := fmt.Sprintf("%s/%s", dirPath, "foo.pdf")
	err := Merge(
		[]string{
			fmt.Sprintf("%s/%s", dirPath, "gotenberg.pdf"),
			fmt.Sprintf("%s/%s", dirPath, "gotenberg_bis.pdf"),
		},
		dst,
	)
	require.Nil(t, err)
	require.FileExists(t, dst)
	err = os.RemoveAll(dirPath)
	assert.Nil(t, err)
}
