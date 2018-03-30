package config

import (
	"io/ioutil"
	"text/template"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	AppConfig struct {
		Port string
		Logs struct {
			Level     logrus.Level
			Formatter logrus.Formatter
		}
		CommandsConfig *CommandsConfig
	}

	CommandsConfig struct {
		HTML   *CommandConfig
		Office *CommandConfig
		Merge  *CommandConfig
	}

	CommandConfig struct {
		Timeout  int
		Template *template.Template
	}
)

func NewAppConfig() (*AppConfig, error) {
	fileConfig, err := loadFileConfig()
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
	c.CommandsConfig.HTML = &CommandConfig{}
	c.CommandsConfig.Office = &CommandConfig{}
	c.CommandsConfig.Merge = &CommandConfig{}
	c.CommandsConfig.HTML.Timeout = fileConfig.Commands.HTML.Timeout
	c.CommandsConfig.Office.Timeout = fileConfig.Commands.Office.Timeout
	c.CommandsConfig.Merge.Timeout = fileConfig.Commands.Merge.Timeout

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

	c.CommandsConfig.HTML.Template = tmplHTML
	c.CommandsConfig.Office.Template = tmplOffice
	c.CommandsConfig.Merge.Template = tmplMerge

	return c, nil
}

type fileConfig struct {
	Port string `yaml:"port"`
	Logs struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logs"`
	Commands struct {
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

// configurationFilePath is our default configuration file to parse.
const configurationFilePath = "gotenberg.yml"

func loadFileConfig() (*fileConfig, error) {
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

var levels = map[string]logrus.Level{
	"DEBUG": logrus.DebugLevel,
	"INFO":  logrus.InfoLevel,
	"WARN":  logrus.WarnLevel,
	"ERROR": logrus.ErrorLevel,
	"FATAL": logrus.FatalLevel,
	"PANIC": logrus.PanicLevel,
}

type wrongLoggingLevelError struct{}

func (e *wrongLoggingLevelError) Error() string {
	return "Accepted values for logging level: DEBUG, INFO, WARN, ERROR, FATAL, PANIC"
}

func getLoggingLevelFromFileConfig(c *fileConfig) (logrus.Level, error) {
	l, ok := levels[c.Logs.Level]
	if !ok {
		return 999, &wrongLoggingLevelError{}
	}

	return l, nil
}

var formatters = map[string]logrus.Formatter{
	"text": &logrus.TextFormatter{},
	"json": &logrus.JSONFormatter{},
}

type wrongLoggingFormatError struct{}

func (e *wrongLoggingFormatError) Error() string {
	return "Accepted value for logging format: text, json"
}

func getLoggingFormatterFromFileConfig(c *fileConfig) (logrus.Formatter, error) {
	f, ok := formatters[c.Logs.Format]
	if !ok {
		return nil, &wrongLoggingFormatError{}
	}

	return f, nil
}

func getCommandTemplate(command string, commandName string) (*template.Template, error) {
	t, err := template.New(commandName).Parse(command)
	if err != nil {
		return nil, err
	}

	return t, nil
}
