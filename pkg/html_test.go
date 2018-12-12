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

func TestHTML(t *testing.T) {
	c := &Client{Hostname: "http://localhost:3000"}
	req, err := NewHTMLRequest(test.HTMLTestFilePath(t, "index.html"))
	require.Nil(t, err)
	err = req.SetHeader(test.HTMLTestFilePath(t, "header.html"))
	require.Nil(t, err)
	err = req.SetFooter(test.HTMLTestFilePath(t, "footer.html"))
	require.Nil(t, err)
	err = req.SetAssets([]string{
		test.HTMLTestFilePath(t, "font.woff"),
		test.HTMLTestFilePath(t, "img.gif"),
		test.HTMLTestFilePath(t, "style.css"),
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
