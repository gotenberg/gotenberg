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

func TestMerge(t *testing.T) {
	c := &Client{Hostname: "http://localhost:3000"}
	req, err := NewMergeRequest([]string{
		test.PDFTestFilePath(t, "gotenberg.pdf"),
		test.PDFTestFilePath(t, "gotenberg.pdf"),
	})
	require.Nil(t, err)
	dirPath, err := rand.Get()
	require.Nil(t, err)
	dest := fmt.Sprintf("%s/foo.pdf", dirPath)
	err = c.Store(req, dest)
	assert.Nil(t, err)
	assert.FileExists(t, dest)
	err = os.RemoveAll(dirPath)
	assert.Nil(t, err)
}

func TestConcurrentMerge(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func() {
			TestMerge(t)
		}()
	}
}
