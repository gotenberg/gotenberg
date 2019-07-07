package logger

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

// Logger enforces specific log message formats.
type Logger struct {
	entry *logrus.Entry
}

// New initializes the logger.
func New(level logrus.Level, trace string) *Logger {
	l := logrus.New()
	l.SetLevel(level)
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		l.SetFormatter(&logrus.JSONFormatter{})
	}
	return &Logger{
		entry: l.WithField("trace", trace),
	}
}

// WithFields creates a new logger with
// given fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		entry: l.entry.WithFields(fields),
	}
}

// DebugfOp logs a debug message for given
// logical operation.
func (l *Logger) DebugfOp(op string, format string, args ...interface{}) {
	l.entry.WithField("op", op).Debugf(format, args...)
}

// InfofOp logs an info message for given
// logical operation.
func (l *Logger) InfofOp(op string, format string, args ...interface{}) {
	l.entry.WithField("op", op).Infof(format, args...)
}

// ErrorOp logs an error for given
// logical operation.
func (l *Logger) ErrorOp(op string, err error) {
	l.entry.WithField("op", op).Error(err.Error())
}

// ErrorfOp logs an error message for given
// logical operation.
func (l *Logger) ErrorfOp(op string, message string) {
	l.entry.WithField("op", op).Error(message)
}

// FatalOp logs an error message for given
// logical operation.
func (l *Logger) FatalOp(op string, err error) {
	l.entry.WithField("op", op).Error(err.Error())
}
