package pdf

import (
	"slices"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// TestRunSpans asserts that each written document gets a pdf.write span
// naming its stages, and each edit a child span.
func TestRunSpans(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	ctx, paths := testContext(t, "page_1.pdf", "page_2.pdf")
	stages := mustStages(t, `[{"op":"merge"},{"op":"rotate","angle":90},{"op":"set-metadata","title":"T"}]`)

	_, err := testProcessor().Run(ctx, stages, []string{paths["page_1.pdf"], paths["page_2.pdf"]}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var writes [][]string
	byID := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range recorder.Ended() {
		byID[s.SpanContext().SpanID().String()] = s
	}

	for _, s := range recorder.Ended() {
		switch s.Name() {
		case "pdf.write":
			for _, kv := range s.Attributes() {
				if kv.Key == "gotenberg.pdf.stages" {
					writes = append(writes, kv.Value.AsStringSlice())
				}
			}
		case "pdf.rotate", "pdf.set-metadata":
			parent, ok := byID[s.Parent().SpanID().String()]
			if !ok || parent.Name() != "pdf.write" {
				t.Errorf("%s: parent is not a pdf.write span", s.Name())
			}
		}
	}

	want := [][]string{{"merge"}, {"rotate", "set-metadata"}}
	if !slices.EqualFunc(writes, want, slices.Equal[[]string]) {
		t.Fatalf("pdf.write stages = %v, want %v", writes, want)
	}
}

// TestRunSpansInvalidInput asserts that a failure the request causes marks
// the spans as invalid input.
func TestRunSpansInvalidInput(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	ctx, paths := testContext(t, "page_1.pdf")

	_, err := testProcessor().Run(ctx, mustStages(t, `[{"op":"rotate","angle":90,"pages":"5"}]`), []string{paths["page_1.pdf"]}, nil)
	if err == nil {
		t.Fatal("a page past the end must fail")
	}

	wantType := attribute.String(string(semconv.ErrorTypeKey), gotenberg.ErrorTypeInvalidInput)
	for _, s := range recorder.Ended() {
		if s.Status().Code != codes.Error || !slices.Contains(s.Attributes(), wantType) {
			t.Errorf("%s: status %v, attributes %v, want an invalid input error", s.Name(), s.Status(), s.Attributes())
		}
	}

	if len(recorder.Ended()) == 0 {
		t.Fatal("no span recorded")
	}
}
