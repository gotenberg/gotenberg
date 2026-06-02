package pdfengines

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func newFallbackRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })
	return recorder
}

func findFallbackSpan(recorder *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	for _, s := range recorder.Ended() {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

func wrapTest(err error) error { return fmt.Errorf("test op with multi PDF engines: %w", err) }

func TestRunWithFallback_FirstSucceeds(t *testing.T) {
	recorder := newFallbackRecorder(t)
	engines := []gotenberg.PdfEngine{&gotenberg.PdfEngineMock{}, &gotenberg.PdfEngineMock{}}

	calls := 0
	got, err := runWithFallback(context.Background(), "pdfengines.Test", engines,
		func(_ context.Context, _ gotenberg.PdfEngine) (string, error) {
			calls++
			return "ok", nil
		}, wrapTest)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Errorf("got %q, want ok", got)
	}
	if calls != 1 {
		t.Errorf("expected 1 engine call, got %d", calls)
	}

	span := findFallbackSpan(recorder, "pdfengines.Test")
	if span.Status().Code != codes.Ok {
		t.Errorf("status = %v, want Ok", span.Status().Code)
	}
	attrs := map[string]string{}
	for _, kv := range span.Attributes() {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}
	if attrs["gotenberg.pdf_engine.attempts"] != "1" {
		t.Errorf("attempts = %q, want 1", attrs["gotenberg.pdf_engine.attempts"])
	}
	if attrs["gotenberg.pdf_engine.selected"] == "" {
		t.Error("expected a selected engine attribute")
	}
}

func TestRunWithFallback_SecondSucceeds(t *testing.T) {
	recorder := newFallbackRecorder(t)
	engines := []gotenberg.PdfEngine{&gotenberg.PdfEngineMock{}, &gotenberg.PdfEngineMock{}}

	calls := 0
	got, err := runWithFallback(context.Background(), "pdfengines.Test", engines,
		func(_ context.Context, _ gotenberg.PdfEngine) (string, error) {
			calls++
			if calls == 1 {
				return "", errors.New("first engine failed")
			}
			return "ok", nil
		}, wrapTest)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" || calls != 2 {
		t.Errorf("got %q after %d calls, want ok after 2", got, calls)
	}

	span := findFallbackSpan(recorder, "pdfengines.Test")
	var failedEvents int
	for _, e := range span.Events() {
		if e.Name == "pdf_engine.attempt_failed" {
			failedEvents++
		}
	}
	if failedEvents != 1 {
		t.Errorf("expected 1 attempt_failed event, got %d", failedEvents)
	}
	for _, kv := range span.Attributes() {
		if string(kv.Key) == "gotenberg.pdf_engine.attempts" && kv.Value.Emit() != "2" {
			t.Errorf("attempts = %q, want 2", kv.Value.Emit())
		}
	}
}

func TestRunWithFallback_AllFail(t *testing.T) {
	recorder := newFallbackRecorder(t)
	engines := []gotenberg.PdfEngine{&gotenberg.PdfEngineMock{}, &gotenberg.PdfEngineMock{}}

	sentinel := errors.New("engine failed")
	_, err := runWithFallback(context.Background(), "pdfengines.Test", engines,
		func(_ context.Context, _ gotenberg.PdfEngine) (string, error) {
			return "", sentinel
		}, wrapTest)

	if err == nil {
		t.Fatal("expected an error when all engines fail")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected the joined engine error to be wrapped, got %v", err)
	}

	span := findFallbackSpan(recorder, "pdfengines.Test")
	if span.Status().Code != codes.Error {
		t.Errorf("status = %v, want Error", span.Status().Code)
	}
}

func TestRunWithFallback_ZeroEngines(t *testing.T) {
	newFallbackRecorder(t)
	_, err := runWithFallback(context.Background(), "pdfengines.Test", nil,
		func(_ context.Context, _ gotenberg.PdfEngine) (string, error) {
			return "ok", nil
		}, wrapTest)
	if err == nil {
		t.Error("expected an error with no engines")
	}
}

func TestRunWithFallback_ContextDone(t *testing.T) {
	recorder := newFallbackRecorder(t)
	engines := []gotenberg.PdfEngine{&gotenberg.PdfEngineMock{}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	_, err := runWithFallback(ctx, "pdfengines.Test", engines,
		func(_ context.Context, _ gotenberg.PdfEngine) (string, error) {
			<-release // never returns during the call, forcing the ctx.Done branch
			return "", nil
		}, wrapTest)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	span := findFallbackSpan(recorder, "pdfengines.Test")
	if span.Status().Code != codes.Error {
		t.Errorf("status = %v, want Error (the cancellation must mark the span)", span.Status().Code)
	}
}
