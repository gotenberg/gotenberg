package test

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// MergeMultipartForm returns the body
// for a multipart/form-data request with all
// files under "pdf" folder.
func MergeMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	fpaths := MergeFpaths(t)
	return multipartForm(t, "pdf", formValues, fpaths)
}

// HTMLMultipartForm returns the body
// for a multipart/form-data request with all
// files under "html" folder.
func HTMLMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	// TODO
	return multipartForm(t, "html", formValues, []string{})
}

// URLMultipartForm returns the body
// for a multipart/form-data request with all
// files under "url" folder.
func URLMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	// TODO
	return multipartForm(t, "url", formValues, []string{})
}

// MarkdownMultipartForm returns the body
// for a multipart/form-data request with all
// files under "markdown" folder.
func MarkdownMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	// TODO
	return multipartForm(t, "markdown", formValues, []string{})
}

// OfficeMultipartForm returns the body
// for a multipart/form-data request with all
// files under "office" folder.
func OfficeMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	fpaths := OfficeFpaths(t)
	return multipartForm(t, "office", formValues, fpaths)
}

func multipartForm(
	t *testing.T,
	kind string,
	formValues map[string]string,
	formFilePaths []string,
) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer writer.Close()
	for _, fpath := range formFilePaths {
		file, err := os.Open(fpath)
		require.Nil(t, err)
		part, err := writer.CreateFormFile("foo", filepath.Base(fpath))
		require.Nil(t, err)
		_, err = io.Copy(part, file)
		require.Nil(t, err)
	}
	if kind == "url" {
		err := writer.WriteField("remoteURL", "http://google.com")
		require.Nil(t, err)
	}
	for k, v := range formValues {
		err := writer.WriteField(k, v)
		require.Nil(t, err)
	}
	return body, writer.FormDataContentType()
}
