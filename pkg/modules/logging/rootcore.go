package logging

import (
	"go.uber.org/zap/zapcore"
)

type rootCore struct {
	zapcore.Core
	fieldsPrefix string
}

func (c rootCore) With(fields []zapcore.Field) zapcore.Core {
	if c.fieldsPrefix != "" {
		for i := range fields {
			fields[i].Key = c.fieldsPrefix + "_" + fields[i].Key
		}
	}

	return rootCore{
		Core:         c.Core.With(fields),
		fieldsPrefix: c.fieldsPrefix,
	}
}

func (c rootCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if entry.Level < zapcore.DebugLevel {
		entry.Level = zapcore.DebugLevel
	}

	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}

	return ce
}

func (c rootCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if entry.Level < zapcore.DebugLevel {
		entry.Level = zapcore.DebugLevel
	}

	if c.fieldsPrefix != "" {
		for i := range fields {
			fields[i].Key = c.fieldsPrefix + "_" + fields[i].Key
		}
	}

	return c.Core.Write(entry, fields)
}
