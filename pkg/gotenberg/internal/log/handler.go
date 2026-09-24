package log

import (
	"context"
	"errors"
	"log/slog"
)

type gotenbergHandler struct {
	slog.Handler
	fieldsPrefix string
	loggerName   string
}

func NewGotenbergHandler(next slog.Handler, prefix string) slog.Handler {
	return &gotenbergHandler{Handler: next, fieldsPrefix: prefix}
}

func (h *gotenbergHandler) Handle(ctx context.Context, r slog.Record) error {
	var newAttrs []slog.Attr

	if h.loggerName != "" {
		newAttrs = append(newAttrs, slog.String("logger", h.loggerName))
	}

	var needsNewRecord bool
	if len(newAttrs) > 0 {
		needsNewRecord = true
	}

	if h.fieldsPrefix != "" {
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "logger" || a.Key == "correlation_id" || a.Key == "trace_id" || a.Key == "span_id" {
				newAttrs = append(newAttrs, a)
				needsNewRecord = true
				return true
			}
			a.Key = h.fieldsPrefix + "_" + a.Key
			newAttrs = append(newAttrs, a)
			needsNewRecord = true
			return true
		})
	} else if needsNewRecord {
		r.Attrs(func(a slog.Attr) bool {
			newAttrs = append(newAttrs, a)
			return true
		})
	}

	if needsNewRecord {
		newR := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		newR.AddAttrs(newAttrs...)
		r = newR
	}

	return h.Handler.Handle(ctx, r)
}

func (h *gotenbergHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var prefixed []slog.Attr
	newLoggerName := h.loggerName

	for _, a := range attrs {
		if a.Key == "logger" {
			if newLoggerName == "" {
				newLoggerName = a.Value.String()
			} else {
				newLoggerName = newLoggerName + "." + a.Value.String()
			}
			continue
		}

		if h.fieldsPrefix != "" {
			if a.Key == "correlation_id" || a.Key == "trace_id" || a.Key == "span_id" {
				// Don't prefix these keys
			} else {
				a.Key = h.fieldsPrefix + "_" + a.Key
			}
		}
		prefixed = append(prefixed, a)
	}

	newHandler := h.Handler
	if len(prefixed) > 0 {
		newHandler = h.Handler.WithAttrs(prefixed)
	}

	return &gotenbergHandler{
		Handler:      newHandler,
		fieldsPrefix: h.fieldsPrefix,
		loggerName:   newLoggerName,
	}
}

func (h *gotenbergHandler) WithGroup(name string) slog.Handler {
	return &gotenbergHandler{
		Handler:      h.Handler.WithGroup(name),
		fieldsPrefix: h.fieldsPrefix,
		loggerName:   h.loggerName,
	}
}

type multiHandler struct {
	handlers []slog.Handler
}

func FanOut(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cloned := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		cloned[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: cloned}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	cloned := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		cloned[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: cloned}
}

type levelHandler struct {
	slog.Handler
	level slog.Level
}

func LevelFilter(next slog.Handler, level slog.Level) slog.Handler {
	return &levelHandler{Handler: next, level: level}
}

func (h *levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.Handler.Enabled(ctx, level)
}

func (h *levelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.Handler.Handle(ctx, r)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{Handler: h.Handler.WithAttrs(attrs), level: h.level}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{Handler: h.Handler.WithGroup(name), level: h.level}
}
