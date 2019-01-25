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

func TestOffice(t *testing.T) {
	c := &Client{Hostname: "http://localhost:3000"}
	req, err := NewOfficeRequest(test.OfficeTestFilePath(t, "document.docx"))
	require.Nil(t, err)
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

func TestConcurrentOffice(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func() {
			TestOffice(t)
		}()
	}
}
