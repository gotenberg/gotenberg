package api

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

func sumInt64Counter(rm metricdata.ResourceMetrics, name string) int64 {
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
				for _, dp := range sum.DataPoints {
					total += dp.Value
				}
			}
		}
	}
	return total
}

func TestApi_Pdf_CrashRetryCap(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	recorder := tracetest.NewSpanRecorder()
	previousTracer := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previousTracer) })

	var runCalls int
	supervisor := &gotenberg.ProcessSupervisorMock{
		RunMock: func(_ context.Context, _ *slog.Logger, _ func() error) error {
			runCalls++
			return ErrCrashed
		},
	}

	a := &Api{supervisor: supervisor}
	meter := gotenberg.Meter()
	a.pdfOutputSizeCounter, _ = meter.Int64Histogram("gotenberg.libreoffice.pdf.output.size")
	a.crashRetriesCounter, _ = meter.Int64Counter("gotenberg.libreoffice.conversion.retries")

	err := a.Pdf(context.Background(), slog.New(slog.DiscardHandler), "/nonexistent/in.docx", "/tmp/out.pdf", Options{})
	if err == nil {
		t.Fatal("expected an error after exhausting the retries")
	}
	if !errors.Is(err, ErrCrashed) {
		t.Errorf("expected ErrCrashed, got %v", err)
	}

	// 1 initial attempt + 10 retries.
	if runCalls != 11 {
		t.Errorf("supervisor.Run called %d times, want 11", runCalls)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if retries := sumInt64Counter(rm, "gotenberg.libreoffice.conversion.retries"); retries != 10 {
		t.Errorf("retries counter = %d, want 10", retries)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	if ended[0].Name() != "libreoffice.pdf" {
		t.Errorf("span name = %q, want libreoffice.pdf", ended[0].Name())
	}
	// The final failure adds an exception event after the retries.
	var retries []sdktrace.Event
	for _, event := range ended[0].Events() {
		if event.Name != "exception" {
			retries = append(retries, event)
		}
	}
	if len(retries) != 10 {
		t.Fatalf("retry events = %d, want 10", len(retries))
	}
	for i, event := range retries {
		if event.Name != "gotenberg.conversion.retry" {
			t.Errorf("event %d name = %q, want gotenberg.conversion.retry", i, event.Name)
		}
		if len(event.Attributes) != 1 || event.Attributes[0].Key != "gotenberg.conversion.retry.attempt" || event.Attributes[0].Value.AsInt64() != int64(i+1) {
			t.Errorf("event %d attributes = %v, want gotenberg.conversion.retry.attempt=%d", i, event.Attributes, i+1)
		}
	}
}
