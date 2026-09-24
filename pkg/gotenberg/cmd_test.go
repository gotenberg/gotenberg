package gotenberg

import (
	"context"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newTestSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	return recorder
}

func findSpan(recorder *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	for _, s := range recorder.Ended() {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

func spanAttr(span sdktrace.ReadOnlySpan, key string) (attribute.Value, bool) {
	for _, kv := range span.Attributes() {
		if string(kv.Key) == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestCmd_Exec_NilContext(t *testing.T) {
	cmd := Command(slog.New(slog.DiscardHandler), "true")

	code, err := cmd.Exec()
	if err == nil {
		t.Error("expected an error for a nil context")
	}
	if code != 10 {
		t.Errorf("expected code 10, got %d", code)
	}
}

func TestCmd_Exec_NoParentSpanProducesNoSpan(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	cmd, err := CommandContext(context.Background(), slog.New(slog.DiscardHandler), "sh", "-c", "exit 0")
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	code, err := cmd.Exec()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected code 0, got %d", code)
	}
	if n := len(recorder.Ended()); n != 0 {
		t.Errorf("expected no span without an active parent, got %d", n)
	}
}

func TestCmd_Exec_RecordsSpanOnSuccess(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	ctx, parent := otel.Tracer("test").Start(context.Background(), "parent")
	cmd, err := CommandContext(ctx, slog.New(slog.DiscardHandler), "sh", "-c", "exit 0")
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	code, err := cmd.Exec()
	parent.End()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected code 0, got %d", code)
	}

	span := findSpan(recorder, "process.exec")
	if span == nil {
		t.Fatal("expected a process.exec span to be recorded")
	}
	if span.Status().Code != codes.Ok {
		t.Errorf("expected status Ok, got %v", span.Status().Code)
	}
	if name, ok := spanAttr(span, "process.executable.name"); !ok || name.AsString() != "sh" {
		t.Errorf("expected process.executable.name=sh, got %q (present=%t)", name.AsString(), ok)
	}
	if exit, ok := spanAttr(span, "process.exit.code"); !ok || exit.AsInt64() != 0 {
		t.Errorf("expected process.exit.code=0, got %d (present=%t)", exit.AsInt64(), ok)
	}
}

func TestCmd_Exec_RecordsSpanOnError(t *testing.T) {
	recorder := newTestSpanRecorder(t)

	ctx, parent := otel.Tracer("test").Start(context.Background(), "parent")
	cmd, err := CommandContext(ctx, slog.New(slog.DiscardHandler), "sh", "-c", "exit 3")
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	code, err := cmd.Exec()
	parent.End()
	if err == nil {
		t.Error("expected an error for a non-zero exit code")
	}
	if code != 3 {
		t.Errorf("expected exit code 3, got %d", code)
	}

	span := findSpan(recorder, "process.exec")
	if span == nil {
		t.Fatal("expected a process.exec span to be recorded")
	}
	if span.Status().Code != codes.Error {
		t.Errorf("expected status Error, got %v", span.Status().Code)
	}
	if et, ok := spanAttr(span, "error.type"); !ok || et.AsString() != "process_error" {
		t.Errorf("expected error.type=process_error, got %q (present=%t)", et.AsString(), ok)
	}
}
