// Package test contains useful functions used across tests.
package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// HTMLMultipartForm returns the body
// for a multipate/form-data request with all
// files under "html" folder.
func HTMLMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "html")
}

// MarkdownMultipartForm returns the body
// for a multipate/form-data request with all
// files under "markdown" folder.
func MarkdownMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "markdown")
}

// OfficeMultipartForm returns the body
// for a multipate/form-data request with all
// files under "office" folder.
func OfficeMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "office")
}

// PDFMultipartForm returns the body
// for a multipate/form-data request with all
// files under "pdf" folder.
func PDFMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "pdf")
}

// HTMLTestFilePath returns the absolute
// file path of a file under "html" folder
// in test/testdata
func HTMLTestFilePath(t *testing.T, filename string) string {
	return abs(t, "html", filename)
}

// MarkdownTestFilePath returns the absolute
// file path of a file under "markdown" folder
// in test/testdata
func MarkdownTestFilePath(t *testing.T, filename string) string {
	return abs(t, "markdown", filename)
}

// OfficeTestFilePath returns the absolute
// file path of a file under "office" folder
// in test/testdata
func OfficeTestFilePath(t *testing.T, filename string) string {
	return abs(t, "office", filename)
}

// PDFTestFilePath returns the absolute
// file path of a file under "pdf" folder
// in test/testdata
func PDFTestFilePath(t *testing.T, filename string) string {
	return abs(t, "pdf", filename)
}

func multipartForm(t *testing.T, kind string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer writer.Close()
	dirPath := abs(t, kind, "")
	fpaths := make(map[string]string)
	filepath.Walk(dirPath, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		fpaths[info.Name()] = abs(t, kind, info.Name())
		return nil
	})
	for filename, fpath := range fpaths {
		file, err := os.Open(fpath)
		require.Nil(t, err)
		part, err := writer.CreateFormFile("foo", filename)
		require.Nil(t, err)
		_, err = io.Copy(part, file)
		require.Nil(t, err)
	}
	return body, writer.FormDataContentType()
}

func abs(t *testing.T, kind, filename string) string {
	_, gofilename, _, ok := runtime.Caller(0)
	require.Equal(t, ok, true, "got no caller information")
	if filename == "" {
		path, err := filepath.Abs(fmt.Sprintf("%s/testdata/%s", path.Dir(gofilename), kind))
		require.Nil(t, err, `getting the absolute path of "%s"`, kind)
		return path
	}
	path, err := filepath.Abs(fmt.Sprintf("%s/testdata/%s/%s", path.Dir(gofilename), kind, filename))
	require.Nil(t, err, `getting the absolute path of "%s"`, filename)
	return path
}
