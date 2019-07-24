package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
)

/*
testdataDirectoryPath should be
the absolute of the testdata INSIDE
the Docker image.
*/
const testdataDirectoryPath string = "/gotenberg/tests/test/testdata"

// GenerateDestination simply generates
// a path for a resulting PDF file.
func GenerateDestination() string {
	return fmt.Sprintf("/tmp/%s.pdf", xrand.Get())
}

// MergeFpaths return the paths
// of the PDF files used in tests.
func MergeFpaths(t *testing.T) []string {
	return []string{
		fpath(t, "pdf", "gotenberg.pdf"),
		fpath(t, "pdf", "gotenberg_bis.pdf"),
	}
}

// OfficeFpaths return the paths
// of the Office documents used in tests.
func OfficeFpaths(t *testing.T) []string {
	return []string{
		fpath(t, "office", "document.docx"),
		fpath(t, "office", "document.rtf"),
		fpath(t, "office", "document.txt"),
	}
}

func fpath(t *testing.T, kind, filename string) string {
	require.NotEmpty(t, kind)
	require.NotEmpty(t, filename)
	fpath := fmt.Sprintf("%s/%s/%s", testdataDirectoryPath, kind, filename)
	require.FileExists(t, fpath)
	return fpath
}
