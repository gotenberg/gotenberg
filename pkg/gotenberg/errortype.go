package gotenberg

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// Engine-agnostic, low-cardinality error.type values shared by the conversion
// engines. They are safe to use both as the semconv error.type span attribute
// and as bounded metric label values.
const (
	ErrorTypeTimeout           = "timeout"
	ErrorTypeContextCancelled  = "context_cancelled"
	ErrorTypeQueueSizeExceeded = "queue_size_exceeded"
	ErrorTypeProcessRestarting = "process_restarting"
	ErrorTypeInvalidInput      = "invalid_input"
	ErrorTypeUnknown           = "unknown"
)

// ClassifyError maps err to a bounded, engine-agnostic error.type value. It
// recognizes the failure modes shared by every engine: deadline, cancellation,
// queue saturation, and process restart. It returns an empty string for a nil
// error and [ErrorTypeUnknown] for anything it does not recognize, leaving
// engine-specific refinement (such as [ErrorTypeInvalidInput]) to the caller.
func ClassifyError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorTypeTimeout
	case errors.Is(err, context.Canceled):
		return ErrorTypeContextCancelled
	case errors.Is(err, ErrMaximumQueueSizeExceeded):
		return ErrorTypeQueueSizeExceeded
	case errors.Is(err, ErrProcessAlreadyRestarting):
		return ErrorTypeProcessRestarting
	default:
		return ErrorTypeUnknown
	}
}

// SpanErrorType records errorType as the semconv error.type attribute on span.
// It is a no-op when errorType is empty.
func SpanErrorType(span trace.Span, errorType string) {
	if errorType == "" {
		return
	}
	span.SetAttributes(semconv.ErrorTypeKey.String(errorType))
}

// SetSpanStatus sets the status of span from the outcome of the operation it
// traces, with errorType as its error.type on failure, and returns err.
func SetSpanStatus(span trace.Span, err error, errorType string) error {
	if err == nil {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	SpanErrorType(span, errorType)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}
