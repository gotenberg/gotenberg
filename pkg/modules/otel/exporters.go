package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func newMetricExporter(ctx context.Context, protocol string) (metric.Exporter, error) {
	switch protocol {
	case "grpc":
		exporter, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create OTLP gRPC metric exporter: %w", err)
		}
		return exporter, nil
	default:
		return nil, fmt.Errorf("unknown OTLP metric exporter protocol: %s", protocol)
	}
}

func newSpanExporter(ctx context.Context, protocol string) (trace.SpanExporter, error) {
	switch protocol {
	case "grpc":
		exporter, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create OTLP gRPC span exporter: %w", err)
		}
		return exporter, nil
	default:
		return nil, fmt.Errorf("unknown OTLP span exporter protocol: %s", protocol)
	}
}
