// Package logger implements a simple helper for displaying outputs to the user.
package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// newLogger instantiates a logrus logger.
func newLogger() *logrus.Logger {
	l := logrus.New()
	l.Out = os.Stdout
	l.Level = logrus.InfoLevel

	return l
}

// Log is the logger instance used accross the application.
var Log = newLogger()

// logLevels associates log levels from the configuration file with its equivalent
// in the logrus library.
var logLevels = map[string]logrus.Level{
	"DEBUG": logrus.DebugLevel,
	"INFO":  logrus.InfoLevel,
	"WARN":  logrus.WarnLevel,
	"ERROR": logrus.ErrorLevel,
	"FATAL": logrus.FatalLevel,
	"PANIC": logrus.PanicLevel,
}

// SetLevel changes our logger's log level according to
// the log level defined in the configuration file.
func SetLevel(logLevel string) {
	lvl, ok := logLevels[logLevel]
	if ok {
		Log.Level = lvl
	}
}

func InfoR(transactionID string, msg string) {
	Log.WithFields(logrus.Fields{
		"transaction": transactionID,
	}).Info(msg)
}

func WarnR(transactionID string, msg string) {
	Log.WithFields(logrus.Fields{
		"transaction": transactionID,
	}).Warn(msg)
}

func ErrorR(transactionID string, err error, code int, msg string) {
	Log.WithFields(logrus.Fields{
		"transaction": transactionID,
		"code":        code,
		"err":         err.Error(),
	}).Error(msg)
}
