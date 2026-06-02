package otel

import (
	"context"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
)

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
