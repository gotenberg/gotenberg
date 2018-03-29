package http

import "fmt"

type notAuthorizedContentTypeError struct{}

func (e *notAuthorizedContentTypeError) Error() string {
	return fmt.Sprintf("Accepted values for 'Content-Type': %s, %s, %s, %s", PDFContentType, HTMLContentType, OctetStreamContentType, MultipartFormDataContentType)
}

type notAuthorizedFileContentTypeError struct{}

func (e *notAuthorizedFileContentTypeError) Error() string {
	return fmt.Sprintf("Unable to detect a file 'Content-Type'")
}
