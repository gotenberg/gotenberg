package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
		{"filtered", fmt.Errorf("filter URL: %w", refuse(FilterRuleDenyList, "deny-list match")), ErrorTypeInvalidInput},
		{"unresolvable host", fmt.Errorf("navigate: %w", &UnresolvableHostError{Host: "gotenberg.invalid", Err: errors.New("no such host")}), ErrorTypeInvalidInput},
		{"arbitrary", errors.New("boom"), ErrorTypeOther},
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

func TestSetSpanStatus(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	_, success := otel.Tracer("test").Start(context.Background(), "engine.success")
	if err := SetSpanStatus(success, nil, ""); err != nil {
		t.Errorf("SetSpanStatus(nil) = %v, want nil", err)
	}
	success.End()

	errBoom := errors.New("boom")
	_, failure := otel.Tracer("test").Start(context.Background(), "engine.failure")
	if err := SetSpanStatus(failure, errBoom, ErrorTypeInvalidInput); !errors.Is(err, errBoom) {
		t.Errorf("SetSpanStatus(boom) = %v, want boom", err)
	}
	failure.End()

	// A success leaves the status unset, per the semconv.
	got := findSpan(recorder, "engine.success")
	if got.Status().Code != codes.Unset || len(got.Attributes()) != 0 {
		t.Errorf("success: status %v, attributes %v, want unset and none", got.Status(), got.Attributes())
	}

	got = findSpan(recorder, "engine.failure")
	if got.Status().Code != codes.Error || got.Status().Description != "boom" {
		t.Errorf("failure: status %v, want error boom", got.Status())
	}
	if want := attribute.String("error.type", ErrorTypeInvalidInput); !slices.Contains(got.Attributes(), want) {
		t.Errorf("failure: attributes %v, want %v", got.Attributes(), want)
	}
}
