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
// meter, and logger providers. Detection is best-effort: a detector or merge
// failure is logged and the build proceeds with whatever was gathered, so a
// flaky environment never prevents telemetry from starting.
func buildResource(ctx context.Context, logger *slog.Logger, serviceName, serviceVersion string) *resource.Resource {
	base := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
	)

	// Granular detectors only. The WithProcess() bundle is deliberately omitted
	// because it adds process.command_args/process.command_line, which can carry
	// proxy credentials and host-resolver rules passed on the command line.
	detected, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithHostID(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithProcessPID(),
		resource.WithProcessExecutableName(),
		resource.WithProcessExecutablePath(),
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
		resource.WithProcessRuntimeDescription(),
	)
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("partially detect OpenTelemetry resource: %s", err))
	}
	if detected == nil {
		return base
	}

	merged, err := resource.Merge(detected, base)
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("merge OpenTelemetry resource: %s", err))
		return base
	}

	return merged
}

// InitTracerProvider initializes the OpenTelemetry tracer provider.
func InitTracerProvider(logger *slog.Logger, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	initOtelLogger(logger)

	ctx := context.Background()

	res := buildResource(ctx, logger, serviceName, serviceVersion)

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

	res := buildResource(ctx, logger, serviceName, serviceVersion)

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

	res := buildResource(ctx, logger, serviceName, serviceVersion)

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
