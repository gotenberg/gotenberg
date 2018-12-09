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

func TestMarkdown(t *testing.T) {
	dirPath := test.MarkdownTestDirPath(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	markdown := &Markdown{
		Context:      ctx,
		TemplatePath: fmt.Sprintf("%s/%s", dirPath, "index.html"),
		PaperWidth:   8.27,
		PaperHeight:  11.7,
		MarginTop:    1,
		MarginBottom: 1,
		MarginLeft:   1,
		MarginRight:  1,
	}
	err := markdown.WithHeaderFile(fmt.Sprintf("%s/%s", dirPath, "header.html"))
	require.Nil(t, err)
	err = markdown.WithFooterFile(fmt.Sprintf("%s/%s", dirPath, "footer.html"))
	require.Nil(t, err)
	dst := fmt.Sprintf("%s/%s", dirPath, "foo.pdf")
	err = markdown.Print(dst)
	require.Nil(t, err)
	require.FileExists(t, dst)
	err = os.RemoveAll(dirPath)
	assert.Nil(t, err)
}
