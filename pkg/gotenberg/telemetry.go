package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg/internal/log"
	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg/internal/otel"
)

const (
	AutoLoggingFormat = "auto"
	JsonLoggingFormat = "json"
	TextLoggingFormat = "text"
)

const (
	ErrorLoggingLevel = "error"
	WarnLoggingLevel  = "warn"
	InfoLoggingLevel  = "info"
	DebugLoggingLevel = "debug"
)

const (
	GrpcTelemetryExporterProtocol             = "grpc"
	PrometheusTelemetryMetricExporterProtocol = "prometheus"
)

// TelemetryConfig gathers the configuration data for Gotenberg's telemetry.
type TelemetryConfig struct {
	ServiceName             string
	ServiceVersion          string
	TraceExporterProtocols  []string
	MetricExporterProtocols []string
	LogExporterProtocols    []string

	LogLevel              string
	LogFieldsPrefix       string
	LogStdFormat          string
	LogStdEnableGcpFields bool
}

func (cfg TelemetryConfig) zapLevel() zapcore.Level {
	level := zapcore.InvalidLevel
	_ = level.UnmarshalText([]byte(cfg.LogLevel))
	return level
}

// Validate validates the telemetry configuration.
func (cfg TelemetryConfig) Validate() error {
	var err error

	if cfg.ServiceName == "" {
		err = multierr.Append(err,
			errors.New("service name must not be empty"),
		)
	}

	if cfg.ServiceVersion == "" {
		err = multierr.Append(err,
			errors.New("service version must not be empty"),
		)
	}

	for _, protocol := range cfg.TraceExporterProtocols {
		switch protocol {
		case GrpcTelemetryExporterProtocol:
			continue
		default:
			err = multierr.Append(err,
				fmt.Errorf("unknown trace export protocol %s: must be %s ", protocol, GrpcTelemetryExporterProtocol),
			)
		}
	}

	for _, protocol := range cfg.MetricExporterProtocols {
		switch protocol {
		case GrpcTelemetryExporterProtocol:
		case PrometheusTelemetryMetricExporterProtocol:
			continue
		default:
			err = multierr.Append(err,
				fmt.Errorf("unknown metric export protocol %s: must be at least one of %s and %s", protocol, GrpcTelemetryExporterProtocol, PrometheusTelemetryMetricExporterProtocol),
			)
		}
	}

	for _, protocol := range cfg.LogExporterProtocols {
		switch protocol {
		case GrpcTelemetryExporterProtocol:
			continue
		default:
			err = multierr.Append(err,
				fmt.Errorf("unknown trace export protocol %s: must be %s ", protocol, GrpcTelemetryExporterProtocol),
			)
		}
	}

	switch cfg.LogLevel {
	case ErrorLoggingLevel, WarnLoggingLevel, InfoLoggingLevel, DebugLoggingLevel:
		break
	default:
		err = multierr.Append(
			err,
			fmt.Errorf("log level must be either %s, %s, %s or %s", ErrorLoggingLevel, WarnLoggingLevel, InfoLoggingLevel, DebugLoggingLevel),
		)
	}

	switch cfg.LogStdFormat {
	case AutoLoggingFormat, JsonLoggingFormat, TextLoggingFormat:
		break
	default:
		err = multierr.Append(
			err,
			fmt.Errorf("standard log format must be either %s, %s or %s", AutoLoggingFormat, JsonLoggingFormat, TextLoggingFormat),
		)
	}

	return err
}

// StartTelemetry starts the telemetry utilities.
func StartTelemetry(cfg TelemetryConfig) (shutdown func(context.Context) error, err error) {
	var cores []zapcore.Core

	stdCore, err := log.NewStdCore(cfg.zapLevel(), cfg.LogStdFormat, cfg.LogFieldsPrefix, cfg.LogStdEnableGcpFields)
	if err != nil {
		return nil, fmt.Errorf("get standard logger core: %w", err)
	}
	stdLogger := zap.New(stdCore)
	cores = append(cores, stdCore)

	// OpenTelemetry.
	var shutdowns []func(context.Context) error

	shutdownFn, err := otel.InitTracerProvider(stdLogger, cfg.ServiceName, cfg.ServiceVersion, cfg.TraceExporterProtocols)
	if err != nil {
		return nil, fmt.Errorf("initialize OpenTelemetry tracer provider: %w", err)
	}
	shutdowns = append(shutdowns, shutdownFn)

	shutdownFn, err = otel.InitMeterProvider(stdLogger, cfg.ServiceName, cfg.ServiceVersion, cfg.MetricExporterProtocols)
	if err != nil {
		return nil, fmt.Errorf("initialize OpenTelemetry meter provider: %w", err)
	}
	shutdowns = append(shutdowns, shutdownFn)

	shutdownFn, otelCore, err := otel.InitLoggerProvider(stdLogger, cfg.ServiceName, cfg.ServiceVersion, cfg.LogExporterProtocols)
	if err != nil {
		return nil, fmt.Errorf("initialize OpenTelemetry logger provider: %w", err)
	}
	cores = append(cores, otelCore)
	shutdowns = append(shutdowns, shutdownFn)

	// Global logger.
	err = log.InitLogger(cfg.zapLevel(), cfg.LogFieldsPrefix, cores...)
	if err != nil {
		return nil, fmt.Errorf("initialize global logger: %w", err)
	}

	return func(ctx context.Context) error {
		filterErr := func(err error) error {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		var wg sync.WaitGroup
		var errs error
		var mu sync.Mutex

		for _, fn := range shutdowns {
			wg.Add(1)

			go func(shutdownFn func(context.Context) error) {
				defer wg.Done()

				shutdownErr := shutdownFn(ctx)
				if filterErr(shutdownErr) != nil {
					mu.Lock()
					errs = errors.Join(errs, shutdownErr)
					mu.Unlock()
				}
			}(fn)
		}

		wg.Wait()
		return errs
	}, nil
}

// Logger returns the global logger.
func Logger(mod Module) *zap.Logger {
	return log.Logger().Named(mod.Descriptor().ID)
}

// PrometheusRegistry returns the Prometheus registry.
func PrometheusRegistry() *prometheus.Registry {
	return otel.PrometheusRegistry()
}

// LeveledLogger is a wrapper around a [zap.Logger] so that it may be used by a
// [retryablehttp.Client].
type LeveledLogger struct {
	logger *zap.Logger
}

// NewLeveledLogger instantiates a [LeveledLogger].
func NewLeveledLogger(logger *zap.Logger) *LeveledLogger {
	return &LeveledLogger{
		logger: logger,
	}
}

// Error logs a message at the error level using the wrapped zap.Logger.
func (leveled LeveledLogger) Error(msg string, keysAndValues ...interface{}) {
	leveled.logger.Error(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Warn logs a message at the warning level using the wrapped zap.Logger.
func (leveled LeveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	leveled.logger.Warn(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Info logs a message at the info level using the wrapped zap.Logger.
func (leveled LeveledLogger) Info(msg string, keysAndValues ...interface{}) {
	leveled.logger.Info(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Debug logs a message at the debug level using the wrapped zap.Logger.
func (leveled LeveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	leveled.logger.Debug(fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Interface guards.
var (
	_ retryablehttp.LeveledLogger = (*LeveledLogger)(nil)
)
