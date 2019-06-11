// Package test contains useful functions used across tests.
package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// AssertStatusCode checks if the given request
// returns the expected status code.
func AssertStatusCode(t *testing.T, expectedStatusCode int, srv http.Handler, req *http.Request) {
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	assert.Equal(t, expectedStatusCode, rec.Code)
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

// HTMLTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "html" folder.
func HTMLTestMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	return multipartForm(t, "html", formValues)
}

// URLTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "url" folder.
func URLTestMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	return multipartForm(t, "url", formValues)
}

// MarkdownTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "markdown" folder.
func MarkdownTestMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	return multipartForm(t, "markdown", formValues)
}

// OfficeTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "office" folder.
func OfficeTestMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	return multipartForm(t, "office", formValues)
}

// PDFTestMultipartForm returns the body
// for a multipate/form-data request with all
// files under "pdf" folder.
func PDFTestMultipartForm(t *testing.T, formValues map[string]string) (*bytes.Buffer, string) {
	return multipartForm(t, "pdf", formValues)
}

func multipartForm(t *testing.T, kind string, formValues map[string]string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer writer.Close()
	dirPath := abs(t, kind, "")
	fpaths := make(map[string]string)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		fpaths[info.Name()] = abs(t, kind, info.Name())
		return nil
	})
	require.Nil(t, err)
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
	for k, v := range formValues {
		err := writer.WriteField(k, v)
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
