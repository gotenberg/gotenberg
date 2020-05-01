package xlog

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

// Level helps setting the severity
// of the messages displayed.
type Level string

const (
	// DebugLevel is the lowest level.
	DebugLevel Level = "DEBUG"
	// InfoLevel is the intermediate level.
	InfoLevel Level = "INFO"
	// ErrorLevel is the highest level.
	ErrorLevel Level = "ERROR"
)

// Logger enforces specific log message formats.
type Logger struct {
	entry *logrus.Entry
	level Level
}

// New returns a xlog.Logger.
func New(level Level, trace string) Logger {
	l := logrus.New()
	l.SetLevel(mustLogrusLevel(level))
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		l.SetFormatter(&logrus.JSONFormatter{})
	}
	return Logger{
		entry: l.WithField("trace", trace),
		level: level,
	}
}

func mustLogrusLevel(level Level) logrus.Level {
	const op string = "xlog.mustLogrusLevel"
	switch level {
	case DebugLevel:
		return logrus.DebugLevel
	case InfoLevel:
		return logrus.InfoLevel
	case ErrorLevel:
		return logrus.ErrorLevel
	default:
		panic(fmt.Sprintf("%s: '%s' is not associated with any logrus.Level", op, level))
	}
}

// Levels returns a slice of string
// with all severities.
func Levels() []string {
	return []string{
		string(DebugLevel),
		string(InfoLevel),
		string(ErrorLevel),
	}
}

/*
MustParseLevel returns the Level corresponding
to given string.

It panics if no correspondence.
*/
func MustParseLevel(level string) Level {
	const op string = "xlog.MustParseLevel"
	switch level {
	case string(DebugLevel):
		return DebugLevel
	case string(InfoLevel):
		return InfoLevel
	case string(ErrorLevel):
		return ErrorLevel
	default:
		panic(fmt.Sprintf("%s: '%s' is not one of '%v'", op, level, Levels()))
	}
}

// Level returns the current Level.
func (l Logger) Level() Level {
	return l.level
}

// WithFields returns a new xlog.Logger with
// given fields.
func (l Logger) WithFields(fields map[string]interface{}) Logger {
	return Logger{
		entry: l.entry.WithFields(fields),
		level: l.level,
	}
}

// DebugOp logs a debug message for given
// logical operation.
func (l Logger) DebugOp(op, message string) {
	l.entry.WithField("op", op).Debug(message)
}

// DebugOpf logs a debug message for given
// logical operation and format.
func (l Logger) DebugOpf(op, format string, args ...interface{}) {
	l.entry.WithField("op", op).Debugf(format, args...)
}

// InfoOp logs an info message for given
// logical operation.
func (l Logger) InfoOp(op, message string) {
	l.entry.WithField("op", op).Info(message)
}

// InfoOpf logs an info message for given
// logical operation and format.
func (l Logger) InfoOpf(op, format string, args ...interface{}) {
	l.entry.WithField("op", op).Infof(format, args...)
}

// ErrorOp logs an error for given
// logical operation.
func (l Logger) ErrorOp(op string, err error) {
	l.entry.WithField("op", op).Error(err.Error())
}

// ErrorOpf logs an error message for given
// logical operation and format.
func (l Logger) ErrorOpf(op, format string, args ...interface{}) {
	l.entry.WithField("op", op).Errorf(format, args...)
}

// FatalOp logs an error for given
// logical operation and exit 1.
func (l Logger) FatalOp(op string, err error) {
	l.entry.WithField("op", op).Fatal(err.Error())
}
