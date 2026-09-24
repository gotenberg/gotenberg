package api

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
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

func TestApi_Pdf_CoreDumpedRetryCap(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	var runCalls int
	supervisor := &gotenberg.ProcessSupervisorMock{
		RunMock: func(_ context.Context, _ *slog.Logger, _ func() error) error {
			runCalls++
			return ErrCoreDumped
		},
		ReqQueueSizeMock:            func() int64 { return 0 },
		ConversionsSinceRestartMock: func() int64 { return 0 },
	}

	a := &Api{supervisor: supervisor}
	meter := gotenberg.Meter()
	a.reqsCounter, _ = meter.Int64Counter("libreoffice.requests.total")
	a.errsCounter, _ = meter.Int64Counter("libreoffice.errors.total")
	a.conversionDurationCounter, _ = meter.Float64Histogram("libreoffice.conversion.duration")
	a.queueWaitDurationCounter, _ = meter.Float64Histogram("libreoffice.queue.wait.duration")
	a.pdfOutputSizeCounter, _ = meter.Int64Histogram("libreoffice.pdf.output.size")
	a.coreDumpedRetriesCounter, _ = meter.Int64Counter("libreoffice.conversion.retries.total")

	err := a.Pdf(context.Background(), slog.New(slog.DiscardHandler), "/nonexistent/in.docx", "/tmp/out.pdf", Options{})
	if err == nil {
		t.Fatal("expected an error after exhausting the retries")
	}
	if !errors.Is(err, ErrCoreDumped) {
		t.Errorf("expected ErrCoreDumped, got %v", err)
	}

	// 1 initial attempt + 10 retries.
	if runCalls != 11 {
		t.Errorf("supervisor.Run called %d times, want 11", runCalls)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if retries := sumInt64Counter(rm, "libreoffice.conversion.retries.total"); retries != 10 {
		t.Errorf("retries counter = %d, want 10", retries)
	}
	// Per-attempt request metric must be preserved: one per attempt.
	if reqs := sumInt64Counter(rm, "libreoffice.requests.total"); reqs != 11 {
		t.Errorf("requests counter = %d, want 11", reqs)
	}
}
