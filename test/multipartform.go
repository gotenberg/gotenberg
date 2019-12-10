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

/*
MergeMultipartForm returns the body
for a multipart/form-data request with all
files under "testdata/pdf" folder.
*/
func MergeMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	fpaths := MergeFpaths(t)
	return multipartForm(t, "pdf", formValues, fpaths)
}

/*
HTMLMultipartForm returns the body
for a multipart/form-data request with all
files under "testdata/html" folder.
*/
func HTMLMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	fpaths := HTMLFpaths(t)
	return multipartForm(t, "html", formValues, fpaths)
}

/*
URLMultipartForm returns the body
for a multipart/form-data request with all
files under "testdata/url" folder.
*/
func URLMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	fpaths := URLFpaths(t)
	return multipartForm(t, "url", formValues, fpaths)
}

/*
MarkdownMultipartForm returns the body
for a multipart/form-data request with all
files under "testdata/markdown" folder.
*/
func MarkdownMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	fpaths := MarkdownFpaths(t)
	return multipartForm(t, "markdown", formValues, fpaths)
}

/*
OfficeMultipartForm returns the body
for a multipart/form-data request with all
files under "testdata/office" folder.
*/
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
		err := writer.WriteField("remoteURL", "https://google.com")
		require.Nil(t, err)
	}
	for k, v := range formValues {
		err := writer.WriteField(k, v)
		require.Nil(t, err)
	}
	return body, writer.FormDataContentType()
}
