package otel

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/go-logr/zapr"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitTracerProvider initializes the OpenTelemetry tracer provider.
func InitTracerProvider(stdLogger *zap.Logger, serviceName, serviceVersion string, protocols []string) (shutdown func(context.Context) error, err error) {
	initOtelLogger(stdLogger)

	ctx := context.Background()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("get hostname: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.HostName(hostname),
	)

	traceOpts := []trace.TracerProviderOption{
		trace.WithResource(res),
	}

	for _, protocol := range protocols {
		traceExporter, err := newTraceExporter(ctx, protocol)
		if err != nil {
			return nil, err
		}

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
func InitMeterProvider(stdLogger *zap.Logger, serviceName, serviceVersion string, protocols []string) (shutdown func(context.Context) error, err error) {
	initOtelLogger(stdLogger)

	ctx := context.Background()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("get hostname: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.HostName(hostname),
	)

	metricOpts := []metric.Option{
		metric.WithResource(res),
	}

	for _, protocol := range protocols {
		metricReader, err := newMetricReader(ctx, protocol)
		if err != nil {
			return nil, err
		}

		metricOpts = append(metricOpts, metric.WithReader(metricReader))
	}

	meterProvider := metric.NewMeterProvider(metricOpts...)
	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}

// InitLoggerProvider initializes the OpenTelemetry logger provider.
func InitLoggerProvider(stdLogger *zap.Logger, serviceName, serviceVersion string, protocols []string) (shutdown func(context.Context) error, core zapcore.Core, err error) {
	initOtelLogger(stdLogger)

	ctx := context.Background()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, fmt.Errorf("get hostname: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.HostName(hostname),
	)

	logOpts := []log.LoggerProviderOption{
		log.WithResource(res),
	}

	for _, protocol := range protocols {
		logExporter, err := newLogExporter(ctx, protocol)
		if err != nil {
			return nil, nil, err
		}

		logOpts = append(logOpts, log.WithProcessor(log.NewBatchProcessor(logExporter)))
	}

	loggerProvider := log.NewLoggerProvider(logOpts...)
	otelCore := otelzap.NewCore(
		serviceName,
		otelzap.WithLoggerProvider(loggerProvider),
	)
	global.SetLoggerProvider(loggerProvider)

	return loggerProvider.Shutdown, otelCore, nil
}

func initOtelLogger(stdLogger *zap.Logger) {
	if otlpLoggerInitialized.Load() {
		return
	}

	otel.SetLogger(zapr.NewLogger(stdLogger))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		stdLogger.Error(err.Error())
	}))

	otlpLoggerInitialized.Store(true)
}

// PrometheusRegistry returns the Prometheus registry.
func PrometheusRegistry() *prometheus.Registry {
	promRegistryMu.Lock()
	defer promRegistryMu.Unlock()

	if promRegistry == nil {
		promRegistry = prometheus.NewRegistry()
	}

	return promRegistry
}

var (
	otlpLoggerInitialized atomic.Bool
	promRegistry          *prometheus.Registry
	promRegistryMu        sync.Mutex
)
