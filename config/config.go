// Package config implements a solution for parsing a configuration file ("gotenberg.yml")
// and populating an instance of Config which will be used accross the application.
package config

import (
	"fmt"
	"io/ioutil"
	"text/template"

	"gopkg.in/yaml.v2"
)

// Config represents the data provided by a YAML file.
type Config struct {
	// Port the port the application will listen to.
	Port string `yaml:"port"`
	// LogLevel the log level used by the logger of the application.
	LogLevel string `yaml:"logLevel"`
	// Commands the commands' templates from the configuration file.
	Commands struct {
		// HTMLtoPDF the command's template to convert an HTML file to a PDF file.
		HTMLtoPDF string `yaml:"HTMLtoPDF"`
		// WordToPDF the command's template to convert a Word file to a PDF file.
		WordToPDF string `yaml:"WordToPDF"`
		// MergePDF the command's template to merge many PDF files into one final PDF file.
		MergePDF string `yaml:"MergePDF"`
	} `yaml:"commands"`
	// Templates gathers all instances of Template which will be created from previous
	// Commands block.
	Templates struct {
		// HTMLtoPDF the instance of template created from Commands.HTMLtoPDF.
		HTMLtoPDF *template.Template
		// WordToPDF the instance of template created from Commands.HTMLtoPDF.
		WordToPDF *template.Template
		// MergePDF the instance of template created from Commands.HTMLtoPDF.
		MergePDF *template.Template
	}
}

// AppConfig is the configuration instance used accross the application.
var AppConfig *Config

// configurationFilePath is our default configuration file to parse.
const configurationFilePath = "gotenberg.yml"

// MakeConfig instantiates our configuration by parsing a YAML file.
func MakeConfig() error {
	AppConfig = &Config{}

	data, err := ioutil.ReadFile(configurationFilePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return err
	}

	tmpl, err := template.New("HTMLToPDF").Parse(AppConfig.Commands.HTMLtoPDF)
	if err != nil {
		return fmt.Errorf("Unable to parse the HTML to PDF command: %s", err)
	}
	AppConfig.Templates.HTMLtoPDF = tmpl

	tmpl, err = template.New("WordToPDF").Parse(AppConfig.Commands.WordToPDF)
	if err != nil {
		return fmt.Errorf("Unable to parse the Word to PDF command: %s", err)
	}
	AppConfig.Templates.WordToPDF = tmpl

	tmpl, err = template.New("MergePDF").Parse(AppConfig.Commands.MergePDF)
	if err != nil {
		return fmt.Errorf("Unable to parse the merge PDF command: %s", err)
	}
	AppConfig.Templates.MergePDF = tmpl

	return nil
}
