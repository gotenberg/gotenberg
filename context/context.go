package context

import (
	"context"

	"github.com/gulien/gotenberg/converters"
	"github.com/gulien/gotenberg/logger"
)

const transactionIDKey = "TransactionID"

func WithTransactionID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, transactionIDKey, v)
}

func GetTransactionID(ctx context.Context) string {
	v, ok := ctx.Value(transactionIDKey).(string)
	if !ok {
		logger.Log.Warn("Unable to retrieve the transaction ID from request context")
		return ""
	}
	return v
}

const contentTypeKey = "ContentType"

func WithContentType(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, contentTypeKey, v)
}

func GetContentType(ctx context.Context) string {
	v, ok := ctx.Value(contentTypeKey).(string)
	if !ok {
		logger.Log.Error("Unable to retrieve the content type from request context")
		return ""
	}
	return v
}

const resultFilePathKey = "ResultFilePath"

func WithResultFilePath(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, resultFilePathKey, v)
}

func GetResultFilePath(ctx context.Context) string {
	v, ok := ctx.Value(resultFilePathKey).(string)
	if !ok {
		logger.Log.Error("Unable to retrieve the result file path from request context")
		return ""
	}
	return v
}

const converterKey = "Converter"

func WithConverter(ctx context.Context, v converters.Converter) context.Context {
	return context.WithValue(ctx, converterKey, v)
}

func GetConverter(ctx context.Context) converters.Converter {
	v, ok := ctx.Value(converterKey).(converters.Converter)
	if !ok {
		logger.Log.Warn("Unable to retrieve the converter from request context")
		return nil
	}
	return v
}
