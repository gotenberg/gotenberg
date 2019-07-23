package printertest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
)

const testdataDirectoryPath string = "/gotenberg/tests/test/testdata"

// GenerateDestination simply generates
// a path for a resulting PDF file.
func GenerateDestination() string {
	return fmt.Sprintf("/tmp/%s.pdf", xrand.Get())
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
