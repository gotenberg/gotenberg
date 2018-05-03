package config

import (
	"path/filepath"
	"testing"
)

func load(configurationFilePath string) error {
	Reset()
	return ParseFile(configurationFilePath)
}

func TestParseFile(t *testing.T) {
	var path string

	// case 1: uses an empty configuration file path.
	if err := load(""); err == nil {
		t.Error("Configuration should not have been populated by using an empty configuration file path")
	}

	// case 2: uses a broken configuration file.
	path, _ = filepath.Abs("../../_tests/configurations/broken-gotenberg.yml")
	if err := load(path); err == nil {
		t.Errorf("Configuration should not have been populated with '%s'", path)
	}

	// case 3: uses a configuration file with a wrong logging level.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-logging-level-gotenberg.yml")
	if err := load(path); err == nil {
		t.Errorf("Configuration should not have been populated with '%s'", path)
	}

	// case 4: uses a configuration file with a wrong logging formatter.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-logging-formatter-gotenberg.yml")
	if err := load(path); err == nil {
		t.Errorf("Configuration should not have been populated with '%s'", path)
	}

	// case 5: uses a configuration file with a wrong merge command template.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-merge-command-template-gotenberg.yml")
	if err := load(path); err == nil {
		t.Errorf("Configuration should not have been populated with '%s'", path)
	}

	// case 6: uses a configuration file with a wrong command template.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-command-template-gotenberg.yml")
	if err := load(path); err == nil {
		t.Errorf("Configuration should not have been populated with '%s'", path)
	}

	// case 7: uses a configuration file with a duplicate command.
	path, _ = filepath.Abs("../../_tests/configurations/duplicate-command-gotenberg.yml")
	if err := load(path); err == nil {
		t.Errorf("Configuration should not have been populated with '%s'", path)
	}

	// case 8: uses a correct configuration file.
	path, _ = filepath.Abs("../../_tests/configurations/gotenberg.yml")
	if err := load(path); err != nil {
		t.Errorf("Configuration should have been populated with '%s'", path)
	}
}
