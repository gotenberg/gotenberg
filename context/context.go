// Package context implements a solution for accessing and setting a request's context values.
package context

import (
	"context"

	"github.com/gulien/gotenberg/converters"
	"github.com/gulien/gotenberg/logger"
)

// transactionIDCtxKeyType is a basic type for transactionIDCtxKey.
type transactionIDCtxKeyType string

// transactionIDCtxKey is the transactionID accessing key.
const transactionIDCtxKey transactionIDCtxKeyType = "transactionID"

// WithTransactionID populates a context ctx with a transaction ID v.
func WithTransactionID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, transactionIDCtxKey, v)
}

// GetTransactionID returns the transaction ID from the context ctx.
func GetTransactionID(ctx context.Context) string {
	v, ok := ctx.Value(transactionIDCtxKey).(string)
	if !ok {
		logger.Log.Warn("Unable to retrieve the transaction ID from request context")
		return ""
	}
	return v
}

// contentTypeCtxKeyType is a basic type for contentTypeCtxKey.
type contentTypeCtxKeyType string

// contentTypeCtxKey is the contentType accessing key.
const contentTypeCtxKey contentTypeCtxKeyType = "contentType"

// WithContentType populates a context ctx with a content type v.
func WithContentType(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, contentTypeCtxKey, v)
}

// GetContentType returns the content type from the context ctx.
func GetContentType(ctx context.Context) string {
	v, ok := ctx.Value(contentTypeCtxKey).(string)
	if !ok {
		logger.Log.Error("Unable to retrieve the content type from request context")
		return ""
	}
	return v
}

// resultFilePathCtxKeyType is a basic type for resultFilePathCtxKey.
type resultFilePathCtxKeyType string

// resultFilePathCtxKey is the resultFilePath accessing key.
const resultFilePathCtxKey resultFilePathCtxKeyType = "resultFilePath"

// WithResultFilePath populates a context ctx with a result file path v.
func WithResultFilePath(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, resultFilePathCtxKey, v)
}

// GetResultFilePath returns the result file path from the context ctx.
func GetResultFilePath(ctx context.Context) string {
	v, ok := ctx.Value(resultFilePathCtxKey).(string)
	if !ok {
		logger.Log.Error("Unable to retrieve the result file path from request context")
		return ""
	}
	return v
}

// converterCtxKeyType is a basic type for converterCtxKey.
type converterCtxKeyType string

// converterCtxKey is the converter accessing key.
const converterCtxKey contentTypeCtxKeyType = "converter"

// WithConverter populates a context ctx with a converter v.
func WithConverter(ctx context.Context, v converters.Converter) context.Context {
	return context.WithValue(ctx, converterCtxKey, v)
}

// GetConverter returns the converter from the context ctx.
func GetConverter(ctx context.Context) converters.Converter {
	v, ok := ctx.Value(converterCtxKey).(converters.Converter)
	if !ok {
		logger.Log.Warn("Unable to retrieve the converter from request context")
		return nil
	}
	return v
}
