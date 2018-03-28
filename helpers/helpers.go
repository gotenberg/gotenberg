// Package helpers implements simple functions used across the application.
package helpers

import (
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

// allowedContentTypes contains all allowed content types
// by the application.
var allowedContentTypes = []string{
	"application/pdf",
	"text/html",
	"application/octet-stream",
	"application/msword",
	"application/zip",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"multipart/form-data",
}

// GetMatchingContentType parses a content type and tries to find a match
// with our allowed content types. If no match found, returns an empty string.
func GetMatchingContentType(contentType string) string {
	for _, allowedContentType := range allowedContentTypes {
		if i := strings.IndexRune(contentType, ';'); i != -1 {
			contentType = contentType[0:i]
		}

		if contentType == allowedContentType {
			return allowedContentType
		}
	}

	return ""
}

// getMatchingContentTypeForFile is a simple wrapper of GetMatchingContentType
// function. It adds a condition for "multipart/form-data" content type, which
// is not an allowed content type for a file.
func getMatchingContentTypeForFile(contentType string) string {
	matchingContentType := GetMatchingContentType(contentType)
	if matchingContentType == "multipart/form-data" {
		return ""
	}

	return matchingContentType
}

// DetectMultipartFileContentType sniffs the content type of a file from
// form data.
func DetectMultipartFileContentType(file multipart.File) (string, error) {
	// only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "", err
	}

	// resets the read pointer.
	file.Seek(0, 0)

	// using n if size of buffer < 512 bytes.
	return getMatchingContentTypeForFile(http.DetectContentType(buffer[:n])), nil
}

// DetectFileContentType sniffs the content type of a file.
func DetectFileContentType(file *os.File) (string, error) {
	// only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "", err
	}

	// resets the read pointer.
	file.Seek(0, 0)

	// using n if size of buffer < 512 bytes.
	return getMatchingContentTypeForFile(http.DetectContentType(buffer[:n])), nil
}
