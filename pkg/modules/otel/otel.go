package otel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	flag "github.com/spf13/pflag"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(Otel))
}

// Otel is a module that provides OpenTelemetry instrumentation.
type Otel struct {
	serviceName            string
	logExporterProtocol    string
	disableLogExporter     bool
	metricExporterProcotol string
	metricsCollectInterval time.Duration
	disableMetricExporter  bool
	spanExporterProtocol   string
	disableSpanExporter    bool

	logger             *zap.Logger
	logExporterHook    gotenberg.LogExporterHook
	metrics            []gotenberg.Metric
	otlpLoggerProvider *sdklog.LoggerProvider
	otlpMeterProvider  *metric.MeterProvider
	otlpTracerProvider *trace.TracerProvider
	otlpTracer         oteltrace.Tracer
}

// Descriptor returns a [Otel]'s module descriptor.
func (mod *Otel) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "otel",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("otel", flag.ExitOnError)
			fs.String("otel-service-name", "gotenberg", "Set the OTLP service name")
			fs.String("otel-log-exporter-protocol", "grpc", "Set the OTLP log exporter protocol")
			fs.Bool("otel-disable-log-exporter", true, "Disable the OTLP log exporter")
			fs.String("otel-metric-exporter-protocol", "grpc", "Set the OTLP metric exporter protocol")
			fs.Duration("otel-metrics-collect-interval", time.Duration(5)*time.Second, "Set the interval for collecting modules' metrics")
			fs.Bool("otel-disable-metric-exporter", true, "Disable the OTLP metric exporter")
			fs.String("otel-span-exporter-protocol", "grpc", "Set the OTLP span exporter protocol")
			fs.Bool("otel-disable-span-exporter", true, "Disable the OTLP span exporter")

			return fs
		}(),
		New: func() gotenberg.Module { return new(Otel) },
	}
}

// Provision sets the module properties.
func (mod *Otel) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.serviceName = flags.MustString("otel-service-name")
	mod.logExporterProtocol = flags.MustString("otel-log-exporter-protocol")
	mod.disableLogExporter = flags.MustBool("otel-disable-log-exporter")
	mod.metricExporterProcotol = flags.MustString("otel-metric-exporter-protocol")
	mod.metricsCollectInterval = flags.MustDuration("otel-metrics-collect-interval")
	mod.disableMetricExporter = flags.MustBool("otel-disable-metric-exporter")
	mod.spanExporterProtocol = flags.MustString("otel-span-exporter-protocol")
	mod.disableSpanExporter = flags.MustBool("otel-disable-span-exporter")

	// Logger.
	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}
	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(mod)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}
	mod.logger = logger.
		Named("internal").
		With(zap.Bool("internal", true))

	if !mod.disableLogExporter {
		// Get log exporter hook.
		hook, err := ctx.Module(new(gotenberg.LogExporterHook))
		if err != nil {
			return fmt.Errorf("get log exporter hook: %w", err)
		}
		mod.logExporterHook = hook.(gotenberg.LogExporterHook)
	}

	if !mod.disableMetricExporter {
		// Get metrics from modules.
		mods, err := ctx.Modules(new(gotenberg.MetricsProvider))
		if err != nil {
			return fmt.Errorf("get metrics providers: %w", err)
		}

		metricsProviders := make([]gotenberg.MetricsProvider, len(mods))
		for i, metricsProvider := range mods {
			metricsProviders[i] = metricsProvider.(gotenberg.MetricsProvider)
		}

		for _, metricsProvider := range metricsProviders {
			metrics, err := metricsProvider.Metrics()
			if err != nil {
				return fmt.Errorf("get metrics: %w", err)
			}

			mod.metrics = append(mod.metrics, metrics...)
		}
	}

	return nil
}

// Validate validates the module properties.
func (mod *Otel) Validate() error {
	if mod.disableMetricExporter && mod.disableSpanExporter && mod.disableLogExporter {
		return nil
	}

	var err error

	if mod.serviceName == "" {
		err = multierr.Append(err,
			errors.New("service name must not be empty"),
		)
	}

	if !mod.disableLogExporter {
		if mod.logExporterProtocol != "grpc" {
			err = multierr.Append(err,
				errors.New("currently, only the 'grpc' protocol is supported for the OTLP log exporter"),
			)
		}
	}

	if !mod.disableMetricExporter {
		if mod.metricExporterProcotol != "grpc" {
			err = multierr.Append(err,
				errors.New("currently, only the 'grpc' protocol is supported for the OTLP metric exporter"),
			)
		}
	}

	if !mod.disableSpanExporter {
		if mod.spanExporterProtocol != "grpc" {
			err = multierr.Append(err,
				errors.New("currently, only the 'grpc' protocol is supported for the OTLP span exporter"),
			)
		}
	}

	return err
}

// TraceStart creates a span using the tracer.
func (mod *Otel) TraceStart(ctx context.Context, name string) (context.Context, gotenberg.TracerSpan) {
	return mod.otlpTracer.Start(ctx, name)
}

// Start starts the OTLP exporter(s).
func (mod *Otel) Start() error {
	if mod.disableMetricExporter && mod.disableSpanExporter && mod.disableLogExporter {
		return nil
	}

	// Internal OTLP logger (never exported).
	otel.SetLogger(zapr.NewLogger(mod.logger))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		mod.logger.Error(err.Error())
	}))

	ctx := context.Background()

	hostName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("get hostname: %w", err)
	}
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(mod.serviceName),
		semconv.ServiceVersion(gotenberg.Version),
		semconv.HostName(hostName),
	)

	if !mod.disableLogExporter {
		logExporter, err := newLogExporter(ctx, mod.logExporterProtocol)
		if err != nil {
			return fmt.Errorf("create OTLP log exporter: %w", err)
		}

		processor := sdklog.NewBatchProcessor(logExporter)

		mod.otlpLoggerProvider = sdklog.NewLoggerProvider(
			sdklog.WithProcessor(processor),
			sdklog.WithResource(res),
		)

		global.SetLoggerProvider(mod.otlpLoggerProvider)

		otelCore := otelzap.NewCore(
			mod.serviceName,
			otelzap.WithLoggerProvider(mod.otlpLoggerProvider),
		)

		err = mod.logExporterHook.RegisterCore(otelCore)
		if err != nil {
			return fmt.Errorf("register core to hook: %w", err)
		}
	}

	if !mod.disableMetricExporter {
		metricExporter, err := newMetricExporter(ctx, mod.metricExporterProcotol)
		if err != nil {
			return fmt.Errorf("create OTLP metric exporter: %w", err)
		}

		mod.otlpMeterProvider = metric.NewMeterProvider(
			metric.WithReader(metric.NewPeriodicReader(metricExporter)),
			metric.WithResource(res),
		)

		otel.SetMeterProvider(mod.otlpMeterProvider)

		meter := mod.otlpMeterProvider.Meter(mod.serviceName)
		for _, m := range mod.metrics {
			switch m.Instrument {
			case gotenberg.CounterInstrument:
				counter, err := meter.Float64Counter(m.Name, otelmetric.WithDescription(m.Description),
					otelmetric.WithUnit("{count}"))
				if err != nil {
					return fmt.Errorf("create counter instrument: %w", err)
				}
				go func(ctx context.Context, counter otelmetric.Float64Counter, metric gotenberg.Metric) {
					for {
						counter.Add(ctx, metric.Read())
						time.Sleep(mod.metricsCollectInterval)
					}
				}(ctx, counter, m)
			case gotenberg.UpDownCounterInstrument:
				counter, err := meter.Float64UpDownCounter(m.Name,
					otelmetric.WithDescription(m.Description),
					otelmetric.WithUnit("{count}"),
				)
				if err != nil {
					return fmt.Errorf("create up down counter instrument: %w", err)
				}
				go func(ctx context.Context, counter otelmetric.Float64UpDownCounter, metric gotenberg.Metric) {
					for {
						counter.Add(ctx, metric.Read())
						time.Sleep(mod.metricsCollectInterval)
					}
				}(ctx, counter, m)
			case gotenberg.HistogramInstrument:
				histogram, err := meter.Float64Histogram(m.Name,
					otelmetric.WithDescription(m.Description),
					otelmetric.WithUnit("{count}"),
				)
				if err != nil {
					return fmt.Errorf("create histogram instrument: %w", err)
				}
				go func(ctx context.Context, histogram otelmetric.Float64Histogram, metric gotenberg.Metric) {
					for {
						histogram.Record(ctx, metric.Read())
						time.Sleep(mod.metricsCollectInterval)
					}
				}(ctx, histogram, m)
			case gotenberg.GaugeInstrument:
				gauge, err := meter.Float64Gauge(m.Name,
					otelmetric.WithDescription(m.Description),
					otelmetric.WithUnit("{count}"),
				)
				if err != nil {
					return fmt.Errorf("create gauge instrument: %w", err)
				}
				go func(ctx context.Context, gauge otelmetric.Float64Gauge, metric gotenberg.Metric) {
					for {
						gauge.Record(ctx, metric.Read())
						time.Sleep(mod.metricsCollectInterval)
					}
				}(ctx, gauge, m)
			default:
				return fmt.Errorf("unknown instrument: %d", m.Instrument)
			}
		}
	}

	if !mod.disableSpanExporter {
		spanExporter, err := newSpanExporter(ctx, mod.spanExporterProtocol)
		if err != nil {
			return fmt.Errorf("OTLP span exporter: %w", err)
		}

		mod.otlpTracerProvider = trace.NewTracerProvider(
			trace.WithBatcher(spanExporter),
			trace.WithResource(res),
		)

		otel.SetTracerProvider(mod.otlpTracerProvider)

		mod.otlpTracer = mod.otlpTracerProvider.Tracer(mod.serviceName)
	}

	return nil
}

// StartupMessage returns a custom startup message.
func (mod *Otel) StartupMessage() string {
	if mod.disableMetricExporter && mod.disableSpanExporter && mod.disableLogExporter {
		return "OTLP exporters are disabled"
	}

	var exporters []string
	if !mod.disableLogExporter {
		exporters = append(exporters, fmt.Sprintf("%s log exporter", mod.logExporterProtocol))
	}
	if !mod.disableMetricExporter {
		exporters = append(exporters, fmt.Sprintf("%s metric exporter", mod.metricExporterProcotol))
	}
	if !mod.disableSpanExporter {
		exporters = append(exporters, fmt.Sprintf("%s span exporter", mod.spanExporterProtocol))
	}

	return fmt.Sprintf("the following OTLP exporter(s) are enabled: %s", strings.Join(exporters, ", "))
}

// Stop shutdowns the OTLP exporter(s).
func (mod *Otel) Stop(ctx context.Context) error {
	if mod.disableMetricExporter && mod.disableSpanExporter && mod.disableLogExporter {
		return nil
	}

	filterErr := func(err error) error {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}

	errLogChan := make(chan error, 1)
	go func() {
		if mod.disableLogExporter {
			errLogChan <- nil
			return
		}
		errLogChan <- mod.otlpLoggerProvider.Shutdown(ctx)
	}()
	errMetricChan := make(chan error, 1)
	go func() {
		if mod.disableMetricExporter {
			errMetricChan <- nil
			return
		}

		errMetricChan <- mod.otlpMeterProvider.Shutdown(ctx)
	}()
	errTracerChan := make(chan error, 1)
	go func() {
		if mod.disableSpanExporter {
			errTracerChan <- nil
			return
		}

		errTracerChan <- mod.otlpTracerProvider.Shutdown(ctx)
	}()

	errLog := filterErr(<-errLogChan)
	errMetric := filterErr(<-errMetricChan)
	errTracer := filterErr(<-errTracerChan)

	return errors.Join(errLog, errMetric, errTracer)
}

// Interface guards.
var (
	_ gotenberg.Module         = (*Otel)(nil)
	_ gotenberg.Provisioner    = (*Otel)(nil)
	_ gotenberg.Validator      = (*Otel)(nil)
	_ gotenberg.TracerProvider = (*Otel)(nil)
	_ gotenberg.App            = (*Otel)(nil)
)
