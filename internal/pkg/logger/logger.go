package logger

import (
	"github.com/sirupsen/logrus"
)

// Logger enforces specific log message formats.
type Logger struct {
	*logrus.Entry
}

// New initializes the logger.
func New(level logrus.Level, trace string) *Logger {
	l := logrus.New()
	l.SetLevel(level)
	// TODO no formatter if TTY.
	l.SetFormatter(&logrus.JSONFormatter{})
	return &Logger{
		l.WithField("trace", trace),
	}
}

// DebugfOp logs a debug message for given
// logical operation.
func (l *Logger) DebugfOp(op string, format string, args ...interface{}) {
	l.WithField("op", op).Debugf(format, args...)
}

// ErrorOp logs an error message for given
// logical operation.
func (l *Logger) ErrorOp(op string, err error) {
	l.WithField("op", op).Error(err.Error())
}
