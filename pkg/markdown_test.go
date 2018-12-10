package gotenberg

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestMarkdown(t *testing.T) {
	c := &Client{Hostname: "http://localhost:3000"}
	req := &MarkdownRequest{
		IndexFilePath: test.MarkdownTestFilePath(t, "index.html"),
		MarkdownFilePaths: []string{
			test.MarkdownTestFilePath(t, "paragraph1.md"),
			test.MarkdownTestFilePath(t, "paragraph2.md"),
			test.MarkdownTestFilePath(t, "paragraph3.md"),
		},
		AssetFilePaths: []string{
			test.HTMLTestFilePath(t, "font.woff"),
			test.HTMLTestFilePath(t, "img.gif"),
			test.HTMLTestFilePath(t, "style.css"),
		},
		Options: &MarkdownOptions{
			HeaderFilePath: test.MarkdownTestFilePath(t, "header.html"),
			FooterFilePath: test.MarkdownTestFilePath(t, "footer.html"),
			PaperSize:      A4,
			PaperMargins:   NormalMargins,
		},
	}
	dirPath, err := rand.Get()
	require.Nil(t, err)
	dest := fmt.Sprintf("%s/foo.pdf", dirPath)
	err = c.Store(req, dest)
	assert.Nil(t, err)
	assert.FileExists(t, dest)
	err = os.RemoveAll(dirPath)
	assert.Nil(t, err)
}
