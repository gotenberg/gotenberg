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
)

func WithContentType(r *http.Request, contentType ghttp.ContentType) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, contentTypeKey, contentType)
	r = r.WithContext(ctx)

	return r
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

func GetConverter(r *http.Request) (*converter.Converter, error) {
	c, ok := r.Context().Value(converterKey).(*converter.Converter)
	if !ok {
		return nil, &converterNotFoundError{}
	}

	return c, nil
}
