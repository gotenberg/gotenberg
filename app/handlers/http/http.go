// Package http provides functions for detecting a request or a file content type.
package http

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ContentType is a string which represents a content type.
type ContentType string

const (
	// PDFContentType represents... the PDF content type.
	PDFContentType ContentType = "application/pdf"
	// HTMLContentType represents... the HTML content type.
	HTMLContentType ContentType = "text/html"
	// OctetStreamContentType represents... the octet stream content type.
	OctetStreamContentType ContentType = "application/octet-stream"
	// ZipContentType represents... the zip content type.
	ZipContentType ContentType = "application/zip"
	// MultipartFormDataContentType represents... the multipart form data content type.
	MultipartFormDataContentType ContentType = "multipart/form-data"
)

type notAuthorizedContentTypeError struct{}

func (e *notAuthorizedContentTypeError) Error() string {
	return fmt.Sprintf("Accepted values for 'Content-Type': %s, %s", OctetStreamContentType, MultipartFormDataContentType)
}

// FindAuthorizedContentType tries to return a content type according to a request header.
// If no authorized content type found, throws an error.
func FindAuthorizedContentType(h http.Header) (ContentType, error) {
	ct := findContentType(h.Get("Content-Type"), HTMLContentType, OctetStreamContentType, MultipartFormDataContentType)
	if ct == "" {
		return "", &notAuthorizedContentTypeError{}
	}

	return ct, nil
}

type notAuthorizedFileContentTypeError struct{}

func (e *notAuthorizedFileContentTypeError) Error() string {
	return fmt.Sprintf("Unable to detect a file content type")
}

// SniffContentType tries to detect the content type of a file.
// If no authorized content type found, throws an error.
func SniffContentType(f *os.File) (ContentType, error) {
	// only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil {
		return "", err
	}

	// resets the read pointer.
	f.Seek(0, 0)

	// using n if size of buffer < 512 bytes.
	ct := findContentType(http.DetectContentType(buffer[:n]), PDFContentType, HTMLContentType, OctetStreamContentType, ZipContentType)
	if ct == "" {
		return "", &notAuthorizedFileContentTypeError{}
	}

	return ct, nil
}

// findContentType parses a string representing a content type and tries to find
// one of the given content types.
func findContentType(contentType string, contentTypes ...ContentType) ContentType {
	for _, ct := range contentTypes {
		if i := strings.IndexRune(contentType, ';'); i != -1 {
			contentType = contentType[0:i]
		}

		if contentType == string(ct) {
			return ct
		}
	}

	return ""
}
