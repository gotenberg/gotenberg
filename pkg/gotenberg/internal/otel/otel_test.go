package otel

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestBuildResource(t *testing.T) {
	res := buildResource(context.Background(), slog.New(slog.DiscardHandler), "gotenberg", "v8.0.0")

	got := map[string]struct{}{}
	values := map[string]string{}
	for _, kv := range res.Attributes() {
		got[string(kv.Key)] = struct{}{}
		values[string(kv.Key)] = kv.Value.AsString()
	}

	if values[string(semconv.ServiceNameKey)] != "gotenberg" {
		t.Errorf("service.name = %q, want %q", values[string(semconv.ServiceNameKey)], "gotenberg")
	}
	if values[string(semconv.ServiceVersionKey)] != "v8.0.0" {
		t.Errorf("service.version = %q, want %q", values[string(semconv.ServiceVersionKey)], "v8.0.0")
	}

	for _, key := range []string{
		string(semconv.HostNameKey),
		string(semconv.OSTypeKey),
		string(semconv.ProcessRuntimeNameKey),
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("expected resource attribute %q to be present", key)
		}
	}

	// The command-line bundle must never be detected (it can leak credentials).
	for _, key := range []string{"process.command_args", "process.command_line"} {
		if _, ok := got[key]; ok {
			t.Errorf("did not expect sensitive resource attribute %q", key)
		}
	}
}

// TestInitTracerProvider_HonorsSamplerEnv guards the contract that the tracer
// provider keeps honoring OTEL_TRACES_SAMPLER. The SDK reads it only when no
// explicit sampler is configured, so any future WithSampler() would silently
// break operator-side sampling control.
func TestInitTracerProvider_HonorsSamplerEnv(t *testing.T) {
	for _, tc := range []struct {
		name        string
		sampler     string
		wantSampled bool
	}{
		{"always off", "always_off", false},
		{"always on", "always_on", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", "none")
			t.Setenv("OTEL_TRACES_SAMPLER", tc.sampler)

			shutdown, err := InitTracerProvider(slog.New(slog.DiscardHandler), "test", "v0.0.0")
			if err != nil {
				t.Fatalf("init tracer provider: %v", err)
			}
			t.Cleanup(func() { _ = shutdown(context.Background()) })

			_, span := otel.Tracer("test").Start(context.Background(), "span")
			span.End()

			if got := span.SpanContext().IsSampled(); got != tc.wantSampled {
				t.Errorf("OTEL_TRACES_SAMPLER=%q: IsSampled() = %v, want %v", tc.sampler, got, tc.wantSampled)
			}
		})
	}
}

func TestExemplarFilterOptions(t *testing.T) {
	t.Run("default pins trace-based", func(t *testing.T) {
		if v, ok := os.LookupEnv("OTEL_METRICS_EXEMPLAR_FILTER"); ok {
			os.Unsetenv("OTEL_METRICS_EXEMPLAR_FILTER")
			t.Cleanup(func() { os.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", v) })
		}
		if got := exemplarFilterOptions(); len(got) != 1 {
			t.Errorf("expected 1 option when env unset, got %d", len(got))
		}
	})

	t.Run("env override yields no option", func(t *testing.T) {
		t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
		if got := exemplarFilterOptions(); len(got) != 0 {
			t.Errorf("expected 0 options when env set, got %d", len(got))
		}
	})
}

// TestMeterProvider_TraceBasedExemplar guards that the trace-based filter we pin
// actually attaches a trace id to a histogram measurement recorded inside a
// sampled span.
func TestMeterProvider_TraceBasedExemplar(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithExemplarFilter(exemplar.TraceBasedFilter),
	)
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	hist, err := provider.Meter("test").Float64Histogram("conversion.duration")
	if err != nil {
		t.Fatalf("create histogram: %v", err)
	}

	tracer := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample())).Tracer("test")
	ctx, span := tracer.Start(context.Background(), "conversion")
	hist.Record(ctx, 1.0)
	traceID := span.SpanContext().TraceID()
	span.End()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	var found bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			hd, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				continue
			}
			for _, dp := range hd.DataPoints {
				for _, ex := range dp.Exemplars {
					if string(ex.TraceID) == string(traceID[:]) {
						found = true
					}
				}
			}
		}
	}

	if !found {
		t.Error("expected a trace-based exemplar carrying the span trace id")
	}
}
