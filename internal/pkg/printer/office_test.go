package printer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestOffice(t *testing.T) {
	dirPath := test.OfficeTestDirPath(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	office := &Office{
		Context: ctx,
		FilePaths: []string{
			fmt.Sprintf("%s/%s", dirPath, "document.docx"),
			fmt.Sprintf("%s/%s", dirPath, "document.txt"),
			fmt.Sprintf("%s/%s", dirPath, "document.rtf"),
		},
	}
	dst := fmt.Sprintf("%s/%s", dirPath, "foo.pdf")
	err := office.Print(dst)
	require.Nil(t, err)
	require.FileExists(t, dst)
	err = os.RemoveAll(dirPath)
	assert.Nil(t, err)
}
