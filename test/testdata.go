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

// MergeFpaths return the paths of all
// files under "testdata/pdf" folder.
func MergeFpaths(t *testing.T) []string {
	return []string{
		fpath(t, "pdf", "gotenberg.pdf"),
		fpath(t, "pdf", "gotenberg_bis.pdf"),
	}
}

// HTMLFpaths return the paths of all
// files under "testdata/html" folder.
func HTMLFpaths(t *testing.T) []string {
	return []string{
		fpath(t, "html", "index.html"),
		fpath(t, "html", "header.html"),
		fpath(t, "html", "footer.html"),
		fpath(t, "html", "style.css"),
		fpath(t, "html", "img.gif"),
		fpath(t, "html", "font.woff"),
	}
}

// URLFpaths return the paths of all
// files under "testdata/url" folder.
func URLFpaths(t *testing.T) []string {
	return []string{
		fpath(t, "url", "header.html"),
		fpath(t, "url", "footer.html"),
	}
}

// MarkdownFpaths return the paths of all
// files under "testdata/markdown" folder.
func MarkdownFpaths(t *testing.T) []string {
	return []string{
		fpath(t, "markdown", "index.html"),
		fpath(t, "markdown", "header.html"),
		fpath(t, "markdown", "footer.html"),
		fpath(t, "markdown", "style.css"),
		fpath(t, "markdown", "img.gif"),
		fpath(t, "markdown", "font.woff"),
		fpath(t, "markdown", "paragraph1.md"),
		fpath(t, "markdown", "paragraph2.md"),
		fpath(t, "markdown", "paragraph3.md"),
	}
}

// OfficeFpaths return the paths of all
// files under "testdata/office" folder.
func OfficeFpaths(t *testing.T) []string {
	return []string{
		fpath(t, "office", "document.docx"),
		fpath(t, "office", "document.rtf"),
		fpath(t, "office", "document.txt"),
		fpath(t, "office", "document_with_special_éà.txt"),
	}
}

func fpath(t *testing.T, kind, filename string) string {
	require.NotEmpty(t, kind)
	require.NotEmpty(t, filename)
	fpath := fmt.Sprintf("%s/%s/%s", testdataDirectoryPath, kind, filename)
	require.FileExists(t, fpath)
	return fpath
}
