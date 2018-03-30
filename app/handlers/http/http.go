package http

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type ContentType string

const (
	PDFContentType               ContentType = "application/pdf"
	HTMLContentType              ContentType = "text/html"
	OctetStreamContentType       ContentType = "application/octet-stream"
	ZipContentType               ContentType = "application/zip"
	MultipartFormDataContentType ContentType = "multipart/form-data"
)

type notAuthorizedContentTypeError struct{}

func (e *notAuthorizedContentTypeError) Error() string {
	return fmt.Sprintf("Accepted values for 'Content-Type': %s, %s, %s, %s", PDFContentType, HTMLContentType, OctetStreamContentType, MultipartFormDataContentType)
}

func FindAuthorizedContentType(h http.Header) (ContentType, error) {
	ct := findContentType(h.Get("Content-Type"), HTMLContentType, OctetStreamContentType, MultipartFormDataContentType)
	if ct == "" {
		return "", &notAuthorizedContentTypeError{}
	}

	return ct, nil
}

type notAuthorizedFileContentTypeError struct{}

func (e *notAuthorizedFileContentTypeError) Error() string {
	return fmt.Sprintf("Unable to detect a file 'Content-Type'")
}

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
