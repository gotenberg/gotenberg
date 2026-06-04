package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hashicorp/go-retryablehttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg/internal/log"
	internalotel "github.com/gotenberg/gotenberg/v8/pkg/gotenberg/internal/otel"
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
	LowerLevelCase = "lower"
	UpperLevelCase = "upper"
)

// TelemetryConfig gathers the configuration data for Gotenberg's telemetry.
type TelemetryConfig struct {
	ServiceName    string
	ServiceVersion string

	LogLevel              string
	LogFieldsPrefix       string
	LogStdFormat          string
	LogStdEnableGcpFields bool
	LogStdLevelCase       string
}

func (cfg TelemetryConfig) slogLevel() slog.Level {
	var level slog.Level
	err := level.UnmarshalText([]byte(cfg.LogLevel))
	if err != nil {
		return slog.LevelInfo
	}
	return level
}

// Validate validates the telemetry configuration.
func (cfg TelemetryConfig) Validate() error {
	var err error

	if cfg.ServiceName == "" {
		err = errors.Join(err,
			errors.New("service name must not be empty"),
		)
	}

	if cfg.ServiceVersion == "" {
		err = errors.Join(err,
			errors.New("service version must not be empty"),
		)
	}

	switch cfg.LogLevel {
	case ErrorLoggingLevel, WarnLoggingLevel, InfoLoggingLevel, DebugLoggingLevel:
		break
	default:
		err = errors.Join(
			err,
			fmt.Errorf("log level must be either %s, %s, %s or %s", ErrorLoggingLevel, WarnLoggingLevel, InfoLoggingLevel, DebugLoggingLevel),
		)
	}

	switch cfg.LogStdFormat {
	case AutoLoggingFormat, JsonLoggingFormat, TextLoggingFormat:
		break
	default:
		err = errors.Join(
			err,
			fmt.Errorf("standard log format must be either %s, %s or %s", AutoLoggingFormat, JsonLoggingFormat, TextLoggingFormat),
		)
	}

	switch cfg.LogStdLevelCase {
	case LowerLevelCase, UpperLevelCase:
		break
	default:
		err = errors.Join(
			err,
			fmt.Errorf("standard log level case must be either %s or %s", LowerLevelCase, UpperLevelCase),
		)
	}

	return err
}

// StartTelemetry starts the telemetry utilities.
func StartTelemetry(cfg TelemetryConfig) (shutdown func(context.Context) error, err error) {
	var handlers []slog.Handler

	stdHandler, err := log.NewStdHandler(cfg.slogLevel(), cfg.LogStdFormat, cfg.LogFieldsPrefix, cfg.LogStdEnableGcpFields, cfg.LogStdLevelCase)
	if err != nil {
		return nil, fmt.Errorf("get standard logger handler: %w", err)
	}
	handlers = append(handlers, stdHandler)

	// We need a logger for the other providers.
	// We'll use the stdHandler for now.
	bootstrapLogger := slog.New(stdHandler)

	// OpenTelemetry.
	var shutdowns []func(context.Context) error

	shutdownFn, err := internalotel.InitTracerProvider(bootstrapLogger, cfg.ServiceName, cfg.ServiceVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize OpenTelemetry tracer provider: %w", err)
	}
	shutdowns = append(shutdowns, shutdownFn)

	shutdownFn, err = internalotel.InitMeterProvider(bootstrapLogger, cfg.ServiceName, cfg.ServiceVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize OpenTelemetry meter provider: %w", err)
	}
	shutdowns = append(shutdowns, shutdownFn)

	shutdownFn, otelHandler, err := internalotel.InitLoggerProvider(bootstrapLogger, cfg.ServiceName, cfg.ServiceVersion)
	if err != nil {
		return nil, fmt.Errorf("initialize OpenTelemetry logger provider: %w", err)
	}
	handlers = append(handlers, log.LevelFilter(otelHandler, cfg.slogLevel()))
	shutdowns = append(shutdowns, shutdownFn)

	// Global logger.
	log.InitLogger(log.NewGotenbergHandler(log.FanOut(handlers...), cfg.LogFieldsPrefix))

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
func Logger(mod Module) *slog.Logger {
	return log.Logger().With(slog.String("logger", mod.Descriptor().ID))
}

const (
	// instrumentationName is the name of the OpenTelemetry instrumentation
	// library.
	instrumentationName = "github.com/gotenberg/gotenberg"
)

// Tracer returns a [trace.Tracer] with the instrumentation name and version
// already set.
func Tracer() trace.Tracer {
	return otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(Version),
	)
}

// Meter returns a [metric.Meter] with the instrumentation name and version
// already set.
func Meter() metric.Meter {
	return otel.GetMeterProvider().Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(Version),
	)
}

// EmitStartupSpan records a single gotenberg.startup span carrying static,
// process-wide attributes that are only known once modules are provisioned,
// such as the chromium and libreoffice binary versions gathered by
// [BuildDebug]. It surfaces version data per trace without re-detecting it on
// every conversion. The engine versions live here, on a span, rather than on
// the resource because the resource is built before modules report them.
func EmitStartupSpan(ctx context.Context) {
	info := Debug()

	var attrs []attribute.KeyValue
	if v := debugModuleVersion(info, "chromium"); v != "" {
		attrs = append(attrs, attribute.String("gotenberg.chromium.version", v))
	}
	if v := debugModuleVersion(info, "libreoffice-api"); v != "" {
		attrs = append(attrs, attribute.String("gotenberg.libreoffice.version", v))
	}

	_, span := Tracer().Start(ctx, "gotenberg.startup",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)
	span.End()
}

// debugModuleVersion returns the "version" entry reported by the module with
// the given ID, or an empty string when it is missing.
func debugModuleVersion(info DebugInfo, moduleID string) string {
	data, ok := info.ModulesAdditionalData[moduleID]
	if !ok {
		return ""
	}

	version, ok := data["version"].(string)
	if !ok {
		return ""
	}

	return version
}

// LeveledLogger is a wrapper around a [slog.Logger] so that it may be used by a
// [retryablehttp.Client].
type LeveledLogger struct {
	logger *slog.Logger
	ctx    context.Context
}

// NewLeveledLogger instantiates a [LeveledLogger].
func NewLeveledLogger(logger *slog.Logger) *LeveledLogger {
	return &LeveledLogger{
		logger: logger,
		ctx:    context.Background(),
	}
}

// WithContext returns a new [LeveledLogger] with the given context.
func (leveled LeveledLogger) WithContext(ctx context.Context) *LeveledLogger {
	return &LeveledLogger{
		logger: leveled.logger,
		ctx:    ctx,
	}
}

// Error logs a message at the error level using the wrapped slog.Logger.
func (leveled LeveledLogger) Error(msg string, keysAndValues ...any) {
	leveled.logger.ErrorContext(leveled.ctx, fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Warn logs a message at the warning level using the wrapped slog.Logger.
func (leveled LeveledLogger) Warn(msg string, keysAndValues ...any) {
	leveled.logger.WarnContext(leveled.ctx, fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Info logs a message at the info level using the wrapped slog.Logger.
func (leveled LeveledLogger) Info(msg string, keysAndValues ...any) {
	leveled.logger.InfoContext(leveled.ctx, fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Debug logs a message at the debug level using the wrapped slog.Logger.
func (leveled LeveledLogger) Debug(msg string, keysAndValues ...any) {
	leveled.logger.DebugContext(leveled.ctx, fmt.Sprintf("%s: %+v", msg, keysAndValues))
}

// Interface guards.
var (
	_ retryablehttp.LeveledLogger = (*LeveledLogger)(nil)
)
