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
	req, err := NewMarkdownRequest(
		test.MarkdownTestFilePath(t, "index.html"),
		[]string{
			test.MarkdownTestFilePath(t, "paragraph1.md"),
			test.MarkdownTestFilePath(t, "paragraph2.md"),
			test.MarkdownTestFilePath(t, "paragraph3.md"),
		},
	)
	require.Nil(t, err)
	err = req.SetHeader(test.MarkdownTestFilePath(t, "header.html"))
	require.Nil(t, err)
	err = req.SetFooter(test.MarkdownTestFilePath(t, "footer.html"))
	require.Nil(t, err)
	err = req.SetAssets([]string{
		test.MarkdownTestFilePath(t, "font.woff"),
		test.MarkdownTestFilePath(t, "img.gif"),
		test.MarkdownTestFilePath(t, "style.css"),
	})
	require.Nil(t, err)
	req.SetPaperSize(A4)
	req.SetMargins(NormalMargins)
	req.SetLandscape(false)
	dirPath, err := rand.Get()
	require.Nil(t, err)
	dest := fmt.Sprintf("%s/foo.pdf", dirPath)
	err = c.Store(req, dest)
	assert.Nil(t, err)
	assert.FileExists(t, dest)
	err = os.RemoveAll(dirPath)
	assert.Nil(t, err)
}
