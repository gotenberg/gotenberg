package config

import (
	"io/ioutil"
	"text/template"

	"github.com/gulien/gotenberg/app/logger"

	"github.com/satori/go.uuid"
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
		logger.Error(err)
		return nil, &fileConfigError{}
	}

	c := &AppConfig{}
	c.Port = fileConfig.Port
	c.Logs.Level = getLoggingLevelFromFileConfig(fileConfig)
	c.Logs.Formatter = getLoggingFormatterFromFileConfig(fileConfig)

	if c.Logs.Level == 999 {
		return nil, &wrongLoggingLevelError{}
	}

	if c.Logs.Formatter == nil {
		return nil, &wrongLoggingFormatError{}
	}

	c.CommandsConfig = &CommandsConfig{}
	c.CommandsConfig.HTML = &CommandConfig{}
	c.CommandsConfig.Office = &CommandConfig{}
	c.CommandsConfig.Merge = &CommandConfig{}
	c.CommandsConfig.HTML.Timeout = fileConfig.Commands.HTML.Timeout
	c.CommandsConfig.Office.Timeout = fileConfig.Commands.Office.Timeout
	c.CommandsConfig.Merge.Timeout = fileConfig.Commands.Merge.Timeout

	tmplHTML, err := getCommandTemplate(fileConfig.Commands.HTML.Template)
	if err != nil {
		logger.Error(err)
		return nil, &wrongHTMLCommandTemplate{}
	}

	tmplOffice, err := getCommandTemplate(fileConfig.Commands.Office.Template)
	if err != nil {
		logger.Error(err)
		return nil, &wrongOfficeCommandTemplate{}
	}

	tmplMerge, err := getCommandTemplate(fileConfig.Commands.Merge.Template)
	if err != nil {
		logger.Error(err)
		return nil, &wrongMergeCommandTemplate{}
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
		logger.Error(err)
		return nil, &readFileError{}
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		logger.Error(err)
		return nil, &unmarshalError{}
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

func getLoggingLevelFromFileConfig(c *fileConfig) logrus.Level {
	l, ok := levels[c.Logs.Level]
	if !ok {
		return 999
	}

	return l
}

var formatters = map[string]logrus.Formatter{
	"text": &logrus.TextFormatter{},
	"json": &logrus.JSONFormatter{},
}

func getLoggingFormatterFromFileConfig(c *fileConfig) logrus.Formatter {
	f, ok := formatters[c.Logs.Format]
	if !ok {
		return nil
	}

	return f
}

func getCommandTemplate(command string) (*template.Template, error) {
	t, err := template.New(uuid.NewV4().String()).Parse(command)
	if err != nil {
		return nil, err
	}

	return t, nil
}
