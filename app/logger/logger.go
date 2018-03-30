// Package logger implements a simple wrapper of the logrus library.
package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// logger wraps a logrus.Logger instance.
type logger struct {
	logger *logrus.Logger
}

// log is our logger instance used across the application.
var log = newLogger()

// newLogger instantiates a logger instance with default values.
func newLogger() *logger {
	l := &logger{
		logger: logrus.New(),
	}

	l.logger.Out = os.Stdout
	l.logger.Level = logrus.InfoLevel

	return l
}

// SetLevel updates the level of messages which will be logged.
func SetLevel(level logrus.Level) {
	log.logger.SetLevel(level)
}

// SetFormatter updates the output format.
// When a TTY is not attached, the output will be in the defined format.
func SetFormatter(formatter logrus.Formatter) {
	log.logger.Formatter = formatter
}

// Debug is a wrapper of the logrus Debug function.
func Debug(message string) {
	log.logger.Debug(message)
}

// Debugf is a wrapper of the logrus Debugf function.
func Debugf(format string, args ...interface{}) {
	log.logger.Debugf(format, args)
}

// Info is a wrapper of the logrus Info function.
func Info(message string) {
	log.logger.Info(message)
}

// Infof is a wrapper of the logrus Infof function.
func Infof(format string, args ...interface{}) {
	log.logger.Infof(format, args)
}

// Warn is a wrapper of the logrus Warn function.
func Warn(message string) {
	log.logger.Warn(message)
}

// Error is a wrapper of the logrus Error function.
func Error(err error) {
	log.logger.Error(err.Error())
}

// Fatal is a wrapper of the logrus Fatal function.
func Fatal(err error) {
	log.logger.Fatal(err.Error())
}

// Panic is a wrapper of the logrus Panic function.
func Panic(err error) {
	log.logger.Panic(err.Error())
}
