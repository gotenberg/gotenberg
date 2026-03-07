package otel

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// InitTracerProvider initializes the OpenTelemetry tracer provider.
func InitTracerProvider(logger *slog.Logger, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	initOtelLogger(logger)

	ctx := context.Background()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("get hostname: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.HostName(hostname),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("merge resource: %w", err)
	}

	traceOpts := []trace.TracerProviderOption{
		trace.WithResource(res),
	}

	traceExporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, err
	}

	if !autoexport.IsNoneSpanExporter(traceExporter) {
		traceOpts = append(traceOpts, trace.WithBatcher(traceExporter))
	}

	traceProvider := trace.NewTracerProvider(traceOpts...)
	otel.SetTracerProvider(traceProvider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return traceProvider.Shutdown, nil
}

// InitMeterProvider initializes the OpenTelemetry meter provider.
func InitMeterProvider(logger *slog.Logger, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	initOtelLogger(logger)

	ctx := context.Background()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("get hostname: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.HostName(hostname),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("merge resource: %w", err)
	}

	metricOpts := []metric.Option{
		metric.WithResource(res),
	}

	metricReader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, err
	}

	if !autoexport.IsNoneMetricReader(metricReader) {
		metricOpts = append(metricOpts, metric.WithReader(metricReader))
	}

	meterProvider := metric.NewMeterProvider(metricOpts...)
	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}

// InitLoggerProvider initializes the OpenTelemetry logger provider.
func InitLoggerProvider(logger *slog.Logger, serviceName, serviceVersion string) (shutdown func(context.Context) error, handler slog.Handler, err error) {
	initOtelLogger(logger)

	ctx := context.Background()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, fmt.Errorf("get hostname: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.HostName(hostname),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("merge resource: %w", err)
	}

	logOpts := []log.LoggerProviderOption{
		log.WithResource(res),
	}

	logExporter, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, nil, err
	}

	if !autoexport.IsNoneLogExporter(logExporter) {
		logOpts = append(logOpts, log.WithProcessor(log.NewBatchProcessor(logExporter)))
	}

	loggerProvider := log.NewLoggerProvider(logOpts...)
	otelHandler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(loggerProvider))
	global.SetLoggerProvider(loggerProvider)

	return loggerProvider.Shutdown, otelHandler, nil
}

func initOtelLogger(logger *slog.Logger) {
	if otlpLoggerInitialized.Load() {
		return
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Error(err.Error())
	}))

	otlpLoggerInitialized.Store(true)
}

var otlpLoggerInitialized atomic.Bool
