// Package http provides functions for detecting a request content type.
package http

import (
	"fmt"
	"net/http"
	"strings"
)

// ContentType is a string which represents a content type.
type ContentType string

// MultipartFormDataContentType represents... the multipart form data content type.
const MultipartFormDataContentType ContentType = "multipart/form-data"

type notAuthorizedContentTypeError struct{}

func (e *notAuthorizedContentTypeError) Error() string {
	return fmt.Sprintf("Accepted value for 'Content-Type': %s", MultipartFormDataContentType)
}

// CheckAuthorizedContentType checks if the request header header has an authorized content type.
// If no authorized content type found, throws an error.
func CheckAuthorizedContentType(h http.Header) error {
	ct := findContentType(h.Get("Content-Type"), MultipartFormDataContentType)
	if ct == "" {
		return &notAuthorizedContentTypeError{}
	}

	return nil
}

// findContentType parses a string representing a content type and tries to find
// one of the given content types.
func findContentType(requestContentType string, contentTypes ...ContentType) ContentType {
	for _, ct := range contentTypes {
		if i := strings.IndexRune(requestContentType, ';'); i != -1 {
			requestContentType = requestContentType[0:i]
		}

		if requestContentType == string(ct) {
			return ct
		}
	}

	return ""
}
