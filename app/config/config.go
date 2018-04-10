/*
Package config contains all the logic allowing us to instantiate the application's configuration.

The application's configuration is loaded from a YAML file named gotenberg.yml.
It should be located where the user starts the application from the CLI.
*/
package config

import (
	"io/ioutil"
	"text/template"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// AppConfig gathers all data required to instantiate the application.
	AppConfig struct {
		// Port is the port which the application will listen to.
		Port string
		// Logs contains the logging configuration.
		Logs struct {
			// Level is the level of messages which will be logged.
			Level logrus.Level
			// Formatter defines the logging format when a TTY is not attached.
			Formatter logrus.Formatter
		}
		// CommandsConfig is... an instance of CommandsConfig.
		CommandsConfig *CommandsConfig
	}

	// CommandsConfig gathers all commands' configurations as defined
	// by the user in the gotenberg.yml file.
	CommandsConfig struct {
		// Markdown is the command's configuration for converting
		// an markdown file to PDF.
		Markdown *CommandConfig
		// HTML is the command's configuration for converting
		// an HTML file to PDF.
		HTML *CommandConfig
		// Office is the command's configuration for converting
		// an Office document to PDF.
		Office *CommandConfig
		// Merge is the command's configuration for merging
		// multiple PDF files into one PDF file.
		Merge *CommandConfig
	}

	// CommandConfig is a command's configuration.
	CommandConfig struct {
		// Timeout is the duration in seconds after which the command's process will be killed
		// if it does not finish before.
		Timeout int
		// Template is the data-driven template of the command.
		Template *template.Template
	}
)

// NewAppConfig instantiates the application's configuration.
// If something bad happens here, the application should not start.
func NewAppConfig(configurationFilePath string) (*AppConfig, error) {
	fileConfig, err := loadFileConfig(configurationFilePath)
	if err != nil {
		return nil, err
	}

	c := &AppConfig{}
	c.Port = fileConfig.Port

	lvl, err := getLoggingLevelFromFileConfig(fileConfig)
	if err != nil {
		return nil, err
	}

	formatter, err := getLoggingFormatterFromFileConfig(fileConfig)
	if err != nil {
		return nil, err
	}

	c.Logs.Level = lvl
	c.Logs.Formatter = formatter

	c.CommandsConfig = &CommandsConfig{}
	c.CommandsConfig.Markdown = &CommandConfig{}
	c.CommandsConfig.HTML = &CommandConfig{}
	c.CommandsConfig.Office = &CommandConfig{}
	c.CommandsConfig.Merge = &CommandConfig{}
	c.CommandsConfig.Markdown.Timeout = fileConfig.Commands.Markdown.Timeout
	c.CommandsConfig.HTML.Timeout = fileConfig.Commands.HTML.Timeout
	c.CommandsConfig.Office.Timeout = fileConfig.Commands.Office.Timeout
	c.CommandsConfig.Merge.Timeout = fileConfig.Commands.Merge.Timeout

	tmplMarkdown, err := getCommandTemplate(fileConfig.Commands.Markdown.Template, "Markdown")
	if err != nil {
		return nil, err
	}

	tmplHTML, err := getCommandTemplate(fileConfig.Commands.HTML.Template, "HTML")
	if err != nil {
		return nil, err
	}

	tmplOffice, err := getCommandTemplate(fileConfig.Commands.Office.Template, "Office")
	if err != nil {
		return nil, err
	}

	tmplMerge, err := getCommandTemplate(fileConfig.Commands.Merge.Template, "Merge")
	if err != nil {
		return nil, err
	}

	c.CommandsConfig.Markdown.Template = tmplMarkdown
	c.CommandsConfig.HTML.Template = tmplHTML
	c.CommandsConfig.Office.Template = tmplOffice
	c.CommandsConfig.Merge.Template = tmplMerge

	return c, nil
}

// fileConfig gathers all data coming from the configuration file gotenberg.yml.
type fileConfig struct {
	Port string `yaml:"port"`
	Logs struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logs"`
	Commands struct {
		Markdown struct {
			Timeout  int    `yaml:"timeout"`
			Template string `yaml:"template"`
		} `yaml:"markdown"`
		HTML struct {
			Timeout  int
			Template string
		} `yaml:"html"`
		Office struct {
			Timeout  int    `yaml:"timeout"`
			Template string `yaml:"template"`
		} `yaml:"office"`
		Merge struct {
			Timeout  int    `yaml:"timeout"`
			Template string `yaml:"template"`
		} `yaml:"merge"`
	} `yaml:"commands"`
}

// loadFileConfig instantiates a fileConfig instance by loading
// the configuration file gotenberg.yml.
func loadFileConfig(configurationFilePath string) (*fileConfig, error) {
	c := &fileConfig{}

	data, err := ioutil.ReadFile(configurationFilePath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	return c, nil
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

type wrongLoggingLevelError struct{}

const wrongLoggingLevelErrorMessage = "Accepted values for logging level: DEBUG, INFO, WARN, ERROR, FATAL, PANIC"

func (e *wrongLoggingLevelError) Error() string {
	return wrongLoggingLevelErrorMessage
}

// getLoggingLevelFromFileConfig returns a logrus level if a matching was found
// with the one defined by the user.
// If no match, throws an error.
func getLoggingLevelFromFileConfig(c *fileConfig) (logrus.Level, error) {
	l, ok := levels[c.Logs.Level]
	if !ok {
		return 999, &wrongLoggingLevelError{}
	}

	return l, nil
}

// levels associates logging formats as defined in the configuration file gotenberg.yml
// with its counterpart from the logrus library.
var formatters = map[string]logrus.Formatter{
	"text": &logrus.TextFormatter{},
	"json": &logrus.JSONFormatter{},
}

type wrongLoggingFormatError struct{}

const wrongLoggingFormatErrorMessage = "Accepted value for logging format: text, json"

func (e *wrongLoggingFormatError) Error() string {
	return wrongLoggingFormatErrorMessage
}

// getLoggingLevelFromFileConfig returns a logrus Formatter if a matching was found
// with the format defined by the user.
// If no match, throws an error.
func getLoggingFormatterFromFileConfig(c *fileConfig) (logrus.Formatter, error) {
	f, ok := formatters[c.Logs.Format]
	if !ok {
		return nil, &wrongLoggingFormatError{}
	}

	return f, nil
}

// getCommandTemplate is a simple helper for parsing a command template as defined by the user.
// If the user gives us a wrong template, throws an error.
func getCommandTemplate(command string, commandName string) (*template.Template, error) {
	t, err := template.New(commandName).Parse(command)
	if err != nil {
		return nil, err
	}

	return t, nil
}
