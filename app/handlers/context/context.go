package context

import (
	"context"
	"net/http"

	"github.com/gulien/gotenberg/app/handlers/converter"

	ghttp "github.com/gulien/gotenberg/app/handlers/http"
)

type key uint32

const (
	contentTypeKey key = iota
	converterKey
	resultFilePathKey
)

func WithContentType(r *http.Request, contentType ghttp.ContentType) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, contentTypeKey, contentType)
	r = r.WithContext(ctx)

	return r
}

type contentTypeNotFoundError struct{}

func (e *contentTypeNotFoundError) Error() string {
	return "The 'Content-Type' was not found in request context"
}

func GetContentType(r *http.Request) (ghttp.ContentType, error) {
	ct, ok := r.Context().Value(contentTypeKey).(ghttp.ContentType)
	if !ok {
		return "", &contentTypeNotFoundError{}
	}

	return ct, nil
}

func WithConverter(r *http.Request, converter *converter.Converter) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, converterKey, converter)
	r = r.WithContext(ctx)

	return r
}

type converterNotFoundError struct{}

func (e *converterNotFoundError) Error() string {
	return "The converter was not found in request context"
}

func GetConverter(r *http.Request) (*converter.Converter, error) {
	c, ok := r.Context().Value(converterKey).(*converter.Converter)
	if !ok {
		return nil, &converterNotFoundError{}
	}

	return c, nil
}

func WithResultFilePath(r *http.Request, resultFilePath string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, resultFilePathKey, resultFilePath)
	r = r.WithContext(ctx)

	return r
}

type resultFilePathNotFoundError struct{}

func (e *resultFilePathNotFoundError) Error() string {
	return "The resultFilePath was not found in request context"
}

func GetResultFilePath(r *http.Request) (string, error) {
	path, ok := r.Context().Value(resultFilePathKey).(string)
	if !ok {
		return "", &resultFilePathNotFoundError{}
	}

	return path, nil
}
