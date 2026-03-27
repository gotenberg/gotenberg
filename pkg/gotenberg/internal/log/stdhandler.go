package log

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"golang.org/x/term"
)

const (
	autoLoggingFormat = "auto"
	jsonLoggingFormat = "json"
	textLoggingFormat = "text"
)

// traceContextHandler is an internal wrapper that specifically injects
// trace_id and span_id into the log record before they are written to the standard output.
type traceContextHandler struct {
	slog.Handler
}

func (h traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		// Since slog.Record is immutable with regards to adding attributes in-place,
		// we must clone and add them.
		newR := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		r.Attrs(func(a slog.Attr) bool {
			newR.AddAttrs(a)
			return true
		})
		newR.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
		r = newR
	}

	return h.Handler.Handle(ctx, r)
}

func (h traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h traceContextHandler) WithGroup(name string) slog.Handler {
	return traceContextHandler{Handler: h.Handler.WithGroup(name)}
}

// NewStdHandler returns a [slog.Handler] instance for the standard output.
func NewStdHandler(level slog.Level, format string, fieldsPrefix string, enableGcpFields bool) (slog.Handler, error) {
	// #nosec: G115
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))

	// Normalize the log format based on the output device.
	if format == autoLoggingFormat {
		if isTerminal {
			format = textLoggingFormat
		} else {
			format = jsonLoggingFormat
		}
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		// Configure level encoding based on format and GCP settings.
		if a.Key == slog.LevelKey {
			l := a.Value.Any().(slog.Level)
			switch {
			case format == textLoggingFormat && isTerminal:
				if enableGcpFields {
					a.Value = slog.StringValue(gcpSeverityColorEncoder(l))
				} else {
					a.Value = slog.StringValue(levelToColor(l).Add(l.String()))
				}
			case enableGcpFields && format != textLoggingFormat:
				a.Key = "severity"
				a.Value = slog.StringValue(gcpSeverity(l))
			default:
				a.Value = slog.StringValue(strings.ToLower(l.String()))
			}
		}

		if a.Key == slog.TimeKey {
			if enableGcpFields && format != textLoggingFormat {
				a.Key = "time"
			} else {
				a.Key = "ts"
			}

			if isTerminal {
				a.Value = slog.StringValue(a.Value.Time().Local().Format("2006/01/02 15:04:05.000"))
			} else if !enableGcpFields {
				a.Value = slog.Float64Value(float64(a.Value.Time().UnixNano()) / 1e9)
			}
		}

		if a.Key == slog.MessageKey {
			if enableGcpFields && format != textLoggingFormat {
				a.Key = "message"
			} else {
				a.Key = "msg"
			}
		}

		return a
	}

	var handler slog.Handler
	if format == textLoggingFormat {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	return traceContextHandler{Handler: handler}, nil
}

func gcpSeverityColorEncoder(l slog.Level) string {
	severity := gcpSeverity(l)
	c := levelToColor(l)
	return c.Add(severity)
}
