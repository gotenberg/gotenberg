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

func TestURL(t *testing.T) {
	c := &Client{Hostname: "http://localhost:3000"}
	req := NewURLRequest("http://google.com")
	err := req.SetHeader(test.URLTestFilePath(t, "header.html"))
	require.Nil(t, err)
	err = req.SetFooter(test.URLTestFilePath(t, "footer.html"))
	require.Nil(t, err)
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

func TestConcurrentURL(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func() {
			TestURL(t)
		}()
	}
}
