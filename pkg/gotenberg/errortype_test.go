package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestClassifyError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"deadline", context.DeadlineExceeded, ErrorTypeTimeout},
		{"canceled", context.Canceled, ErrorTypeContextCancelled},
		{"queue size exceeded", ErrMaximumQueueSizeExceeded, ErrorTypeQueueSizeExceeded},
		{"process restarting", ErrProcessAlreadyRestarting, ErrorTypeProcessRestarting},
		{"wrapped deadline", fmt.Errorf("convert: %w", context.DeadlineExceeded), ErrorTypeTimeout},
		{"joined queue", errors.Join(errors.New("attempt"), ErrMaximumQueueSizeExceeded), ErrorTypeQueueSizeExceeded},
		{"arbitrary", errors.New("boom"), ErrorTypeUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyError(tc.err); got != tc.want {
				t.Errorf("ClassifyError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestSpanErrorType(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	_, span := otel.Tracer("test").Start(context.Background(), "engine.Op")
	SpanErrorType(span, "")               // no-op, must not add an attribute
	SpanErrorType(span, ErrorTypeTimeout) // sets error.type
	span.End()

	got := findSpan(recorder, "engine.Op")
	if got == nil {
		t.Fatal("expected the span to be recorded")
	}

	count := 0
	for _, kv := range got.Attributes() {
		if string(kv.Key) == "error.type" {
			count++
			if kv.Value.AsString() != ErrorTypeTimeout {
				t.Errorf("expected error.type=%q, got %q", ErrorTypeTimeout, kv.Value.AsString())
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one error.type attribute, got %d", count)
	}
}
