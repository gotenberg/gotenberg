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
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

// HTMLTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "html" folder.
func HTMLTestMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "html")
}

// URLTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "url" folder.
func URLTestMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "url")
}

// MarkdownTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "markdown" folder.
func MarkdownTestMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "markdown")
}

// OfficeTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "office" folder.
func OfficeTestMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "office")
}

// PDFTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "pdf" folder.
func PDFTestMultipartForm(t *testing.T) (*bytes.Buffer, string) {
	return multipartForm(t, "pdf")
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
	if kind == "url" {
		err := writer.WriteField("remoteURL", "http://google.com")
		require.Nil(t, err)
	}
	return body, writer.FormDataContentType()
}

// HTMLTestDirPath creates a copy
// of "html" folder in test/testdata.
func HTMLTestDirPath(t *testing.T) string {
	return copyDir(t, "html")
}

// URLTestDirPath creates a copy
// of "url" folder in test/testdata.
func URLTestDirPath(t *testing.T) string {
	return copyDir(t, "url")
}

// MarkdownTestDirPath creates a copy
// of "markdown" folder in test/testdata.
func MarkdownTestDirPath(t *testing.T) string {
	return copyDir(t, "markdown")
}

// OfficeTestDirPath creates a copy
// of "office" folder in test/testdata.
func OfficeTestDirPath(t *testing.T) string {
	return copyDir(t, "office")
}

// PDFTestDirPath creates a copy
// of "pdf" folder in test/testdata.
func PDFTestDirPath(t *testing.T) string {
	return copyDir(t, "pdf")
}

func copyDir(t *testing.T, kind string) string {
	tmpDirPath, err := rand.Get()
	require.Nil(t, err)
	err = os.MkdirAll(tmpDirPath, 0755)
	require.Nil(t, err)
	dirPath := abs(t, kind, "")
	filepath.Walk(dirPath, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		fpath := abs(t, kind, info.Name())
		in, err := os.Open(fpath)
		require.Nil(t, err)
		defer in.Close()
		tmpFpath := fmt.Sprintf("%s/%s", tmpDirPath, info.Name())
		out, err := os.Create(tmpFpath)
		require.Nil(t, err)
		defer out.Close()
		err = out.Chmod(0644)
		require.Nil(t, err)
		_, err = io.Copy(out, in)
		require.Nil(t, err)
		_, err = out.Seek(0, 0)
		require.Nil(t, err)
		return nil
	})
	return tmpDirPath
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
