// Package logger
package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type logger struct {
	logger *logrus.Logger
}

var log *logger = newLogger()

func newLogger() *logger {
	l := &logger{
		logger: logrus.New(),
	}

	l.logger.Out = os.Stdout
	l.logger.Level = logrus.InfoLevel

	return l
}

func SetLevel(level logrus.Level) {
	log.logger.SetLevel(level)
}

func SetFormatter(formatter logrus.Formatter) {
	log.logger.Formatter = formatter
}

func Debug(message string) {
	log.logger.Debug(message)
}

func Debugf(format string, args ...interface{}) {
	log.logger.Debugf(format, args)
}

func Info(message string) {
	log.logger.Info(message)
}

func Infof(format string, args ...interface{}) {
	log.logger.Infof(format, args)
}

func Warn(message string) {
	log.logger.Warn(message)
}

func Error(err error) {
	log.logger.Error(err.Error())
}

func Fatal(err error) {
	log.logger.Fatal(err.Error())
}

func Panic(err error) {
	log.logger.Panic(err.Error())
}
