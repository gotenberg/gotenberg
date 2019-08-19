package test

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

// AssertDirectoryEmpty checks if given directory
// is empty.
func AssertDirectoryEmpty(t *testing.T, directory string) {
	f, err := os.Open(directory)
	assert.Nil(t, err)
	defer f.Close() // nolint: errcheck
	_, err = f.Readdir(1)
	if err == nil {
		return
	}
	assert.Equal(t, io.EOF, err)
}

// AssertConcurrent runs all functions simultaneously
// and wait until execution has completed
// or an error is encountered.
func AssertConcurrent(t *testing.T, fn func() error, amount int) {
	eg := errgroup.Group{}
	for i := 0; i < amount; i++ {
		eg.Go(fn)
	}
	err := eg.Wait()
	assert.NoError(t, err)
}
