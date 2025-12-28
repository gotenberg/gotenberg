package gotenberg

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerProvider is an interface for a module that supplies a method for
// creating a [zap.Logger] instance for use by other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.LoggerProvider))
//		logger, _   := provider.(gotenberg.LoggerProvider).Logger(m)
//	}
type LoggerProvider interface {
	Logger(mod Module) (*zap.Logger, error)
}

// LogExporterHook is an interface for a module that accepts a [zapcore.Core]
// for exporting logs. This allows the telemetry modules to register itself
// with the a logger provider module after the Logger has already been created.
type LogExporterHook interface {
	RegisterCore(core zapcore.Core) error
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

type MetricInstrument int

const (
	CounterInstrument MetricInstrument = iota
	UpDownCounterInstrument
	HistogramInstrument
	GaugeInstrument // trailing number with no extension.
)

// Metric represents a unitary metric.
type Metric struct {
	// Name is the unique identifier.
	// Required.
	Name string

	// Description describes the metric.
	// Optional.
	Description string

	// Instrument is the type of the metric.
	// Required.
	Instrument MetricInstrument

	// Read returns the current value.
	// Required.
	Read func() float64
}

// MetricsProvider is a module interface which provides a list of [Metric].
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.MetricsProvider))
//		metrics, _  := provider.(gotenberg.MetricsProvider).Metrics()
//	}
type MetricsProvider interface {
	Metrics() ([]Metric, error)
}

// TracerSpan is a wrapper around an OpenTelemetry trace span.
type TracerSpan interface {
	oteltrace.Span
}

// TracerProvider provides distributed tracing.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.TracerProvider))
//	}
type TracerProvider interface {
	// TraceStart creates a span using the tracer. It starts a new span with
	// the given name and returns a context containing the span, as well as the
	// span itself. It is the caller's responsibility to end the span.
	TraceStart(ctx context.Context, name string) (context.Context, TracerSpan)

	// Inject propagates the trace context from the context into the given HTTP
	// headers. This allows downstream services (or the client) to continue the
	// trace.
	Inject(ctx context.Context, headers http.Header)
}

// Interface guards.
var (
	_ retryablehttp.LeveledLogger = (*LeveledLogger)(nil)
)
