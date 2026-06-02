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
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// buildResource assembles the OpenTelemetry resource shared by the tracer,
// meter, and logger providers.
func buildResource(serviceName, serviceVersion string) (*resource.Resource, error) {
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

	return res, nil
}

// InitTracerProvider initializes the OpenTelemetry tracer provider.
func InitTracerProvider(logger *slog.Logger, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	initOtelLogger(logger)

	ctx := context.Background()

	res, err := buildResource(serviceName, serviceVersion)
	if err != nil {
		return nil, fmt.Errorf("build resource: %w", err)
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

	res, err := buildResource(serviceName, serviceVersion)
	if err != nil {
		return nil, fmt.Errorf("build resource: %w", err)
	}

	metricOpts := []metric.Option{
		metric.WithResource(res),
	}
	metricOpts = append(metricOpts, exemplarFilterOptions()...)

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

// exemplarFilterOptions returns the meter provider options that pin trace-based
// exemplars, so the histograms expose the trace id of a representative
// measurement. It yields no option when the operator selects a filter via
// OTEL_METRICS_EXEMPLAR_FILTER, letting the SDK's own env handling win.
func exemplarFilterOptions() []metric.Option {
	if _, ok := os.LookupEnv("OTEL_METRICS_EXEMPLAR_FILTER"); ok {
		return nil
	}
	return []metric.Option{metric.WithExemplarFilter(exemplar.TraceBasedFilter)}
}

// InitLoggerProvider initializes the OpenTelemetry logger provider.
func InitLoggerProvider(logger *slog.Logger, serviceName, serviceVersion string) (shutdown func(context.Context) error, handler slog.Handler, err error) {
	initOtelLogger(logger)

	ctx := context.Background()

	res, err := buildResource(serviceName, serviceVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("build resource: %w", err)
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
