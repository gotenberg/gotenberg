/*
Package config contains all the logic allowing us to instantiate the application's configuration.

The application's configuration is loaded from a YAML file named gotenberg.yml.
It should be located where the user starts the application from the CLI.
*/
package config

import (
	"fmt"
	"text/template"

	"github.com/sirupsen/logrus"
)

type (
	// appConfig gathers all configuration data.
	appConfig struct {
		port          string
		logsLevel     logrus.Level
		logsFormatter logrus.Formatter
		// commands associates a file extension with a Command instance.
		// Particular case: ".pdf" extension is used for the merge command.
		commands map[string]*Command
	}

	// Command gathers information on how to launch an external binary used for converting
	// a file to PDF.
	Command struct {
		// Template is the data-driven template of the command.
		Template *template.Template
		// Timeout is the duration in seconds after which the command's process will be killed
		// if it does not finish before.
		Timeout int
	}
)

// our default instance of appConfig.
var config = &appConfig{}

// Reset reinitializes our configuration.
func Reset() {
	config = &appConfig{}
}

// WithPort sets the port which will be used by the application.
func WithPort(port string) {
	config.port = port
}

// GetPort returns the current port.
func GetPort() string {
	return config.port
}

// levels associates logging levels as defined in the configuration file gotenberg.yml
// with its counterpart from the logrus library.
var levels = map[string]logrus.Level{
	"DEBUG": logrus.DebugLevel,
	"INFO":  logrus.InfoLevel,
	"WARN":  logrus.WarnLevel,
	"ERROR": logrus.ErrorLevel,
	"FATAL": logrus.FatalLevel,
	"PANIC": logrus.PanicLevel,
}

type wrongLogsLevelError struct{}

const wrongLogsLevelErrorMessage = "accepted values for logs level: DEBUG, INFO, WARN, ERROR, FATAL, PANIC"

func (e *wrongLogsLevelError) Error() string {
	return wrongLogsLevelErrorMessage
}

// WithLogsLevel sets the logs level.
// If the given string does not match with a logrus level,
// throws an error.
func WithLogsLevel(level string) error {
	l, ok := levels[level]
	if !ok {
		return &wrongLogsLevelError{}
	}

	config.logsLevel = l
	return nil
}

// GetLogsLevel returns the current logs level.
func GetLogsLevel() logrus.Level {
	return config.logsLevel
}

// formatters associates logging formatter as defined in the configuration file gotenberg.yml
// with its counterpart from the logrus library.
var formatters = map[string]logrus.Formatter{
	"text": &logrus.TextFormatter{},
	"json": &logrus.JSONFormatter{},
}

type wrongLogsFormatterError struct{}

const wrongLogsFormatterErrorMessage = "accepted value for logs formatter: text, json"

func (e *wrongLogsFormatterError) Error() string {
	return wrongLogsFormatterErrorMessage
}

// WithLogsFormatter sets the logs formatter.
// If the given string does not match with a logrus formatter,
// throws an error.
func WithLogsFormatter(formatter string) error {
	f, ok := formatters[formatter]
	if !ok {
		return &wrongLogsFormatterError{}
	}

	config.logsFormatter = f
	return nil
}

// GetLogsFormatter returns the current logs formatter.
func GetLogsFormatter() logrus.Formatter {
	return config.logsFormatter
}

// NewCommand instantiates a Command. If the given command string
// is not a valid template, throws an error.
func NewCommand(command string, timeout int) (*Command, error) {
	t, err := template.New(command).Parse(command)
	if err != nil {
		return nil, err
	}

	return &Command{t, timeout}, nil
}

type fileExtensionAlreadyUsedError struct {
	extension       string
	command         *Command
	existingCommand *Command
}

const fileExtensionAlreadyUsedErrorMessage = "file extension '%s' from command '%s' is already used by command '%s'"

func (e *fileExtensionAlreadyUsedError) Error() string {
	return fmt.Sprintf(fileExtensionAlreadyUsedErrorMessage, e.extension, e.command.Template.Name(), e.existingCommand.Template.Name())
}

// WithCommand adds a Command instance and associates it with the given
// file extension. If the file extension is already used by another Command
// instance, throws an error.
func WithCommand(extension string, command *Command) error {
	if config.commands == nil {
		config.commands = make(map[string]*Command)
	}

	existingCommand, ok := config.commands[extension]
	if ok {
		return &fileExtensionAlreadyUsedError{extension, command, existingCommand}
	}

	config.commands[extension] = command
	return nil
}

type noCommandFoundForFileExtensionError struct {
	extension string
}

const noCommandFoundForFileExtensionErrorMessage = "no command found for file extension '%s'"

func (e *noCommandFoundForFileExtensionError) Error() string {
	return fmt.Sprintf(noCommandFoundForFileExtensionErrorMessage, e.extension)
}

// GetCommand returns the Command instance associated with the given
// file extension. If no Command instance found, throws an error.
func GetCommand(extension string) (*Command, error) {
	c, ok := config.commands[extension]
	if !ok {
		return nil, &noCommandFoundForFileExtensionError{extension}
	}

	return c, nil
}
