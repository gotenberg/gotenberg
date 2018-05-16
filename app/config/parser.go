package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ParseFile instantiates the application's configuration using the given YAML file.
func ParseFile(configurationFilePath string) error {
	fileConfig, err := readFile(configurationFilePath)
	if err != nil {
		return err
	}

	WithPort(fileConfig.Port)

	if err := WithLogsLevel(fileConfig.Logs.Level); err != nil {
		return err
	}

	if err := WithLogsFormatter(fileConfig.Logs.Formatter); err != nil {
		return err
	}

	WithLock(fileConfig.Commands.Lock)

	// handles merge command first...
	cmd, err := NewCommand(fileConfig.Commands.Merge.Template, fileConfig.Commands.Merge.Interpreter, fileConfig.Commands.Merge.Timeout)
	if err != nil {
		return err
	}

	WithCommand(".pdf", cmd)

	// ...then conversion commands!
	for _, command := range fileConfig.Commands.Conversions {
		cmd, err := NewCommand(command.Template, command.Interpreter, command.Timeout)
		if err != nil {
			return err
		}

		for _, ext := range command.Extensions {
			if err := WithCommand(ext, cmd); err != nil {
				return err
			}
		}
	}

	return nil
}

type (
	// fileConfig gathers all data coming from the configuration file gotenberg.yml.
	fileConfig struct {
		Port string `yaml:"port"`
		Logs struct {
			Level     string `yaml:"level"`
			Formatter string `yaml:"formatter"`
		} `yaml:"logs"`
		Commands struct {
			Lock        bool                 `yaml:"lock"`
			Merge       *mergeCommand        `yaml:"merge"`
			Conversions []*conversionCommand `yaml:"conversions,omitempty"`
		} `yaml:"commands"`
	}

	// mergeCommand gathers all data regarding the... merge command.
	mergeCommand struct {
		Template    string `yaml:"template"`
		Interpreter string `yaml:"interpreter"`
		Timeout     int    `yaml:"timeout"`
	}

	// conversionCommand gathers all data regarding a conversion command.
	conversionCommand struct {
		Template    string   `yaml:"template"`
		Interpreter string   `yaml:"interpreter"`
		Timeout     int      `yaml:"timeout"`
		Extensions  []string `yaml:"extensions"`
	}
)

// readFile instantiates a fileConfig instance by reading
// the given YAML file.
func readFile(configurationFilePath string) (*fileConfig, error) {
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
