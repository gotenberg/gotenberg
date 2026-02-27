package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func newTraceExporter(ctx context.Context, protocol string) (trace.SpanExporter, error) {
	switch protocol {
	case "grpc":
		exporter, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create OTLP gRPC trace exporter: %w", err)
		}
		return exporter, nil
	default:
		return nil, fmt.Errorf("unknown OTLP trace exporter protocol: %s", protocol)
	}
}

func newMetricReader(ctx context.Context, protocol string) (metric.Reader, error) {
	switch protocol {
	case "grpc":
		exporter, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create OTLP gRPC metric exporter: %w", err)
		}
		return metric.NewPeriodicReader(exporter), nil
	case "prometheus":
		reader, err := prometheus.New(
			prometheus.WithRegisterer(PrometheusRegistry()),
		)
		if err != nil {
			return nil, fmt.Errorf("create OTLP Prometheus metric reader: %w", err)
		}
		return reader, nil
	default:
		return nil, fmt.Errorf("unknown OTLP metric exporter protocol: %s", protocol)
	}
}

func newLogExporter(ctx context.Context, protocol string) (log.Exporter, error) {
	switch protocol {
	case "grpc":
		exporter, err := otlploggrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create OTLP gRPC log exporter: %w", err)
		}
		return exporter, nil
	default:
		return nil, fmt.Errorf("unknown OTLP log exporter protocol: %s", protocol)
	}
}
