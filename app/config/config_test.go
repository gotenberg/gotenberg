package config

import (
	"path/filepath"
	"testing"
)

func TestNewAppConfig(t *testing.T) {
	// case 1: uses an empty configuration file path.
	if _, err := NewAppConfig(""); err == nil {
		t.Error("AppConfig should not have been instantiated!")
	}

	// case 2: uses a broken configuration file.
	path, _ := filepath.Abs("../../_tests/configurations/broken-gotenberg.yml")
	if _, err := NewAppConfig(path); err == nil {
		t.Error("AppConfig should not have been instantiated!")
	}

	// case 3: uses a configuration file with a wrong logging level.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-logging-level-gotenberg.yml")
	if _, err := NewAppConfig(path); err == nil {
		t.Error("AppConfig should not have been instantiated!")
	}

	// case 4: uses a configuration file with a wrong logging format.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-logging-format-gotenberg.yml")
	if _, err := NewAppConfig(path); err == nil {
		t.Error("AppConfig should not have been instantiated!")
	}

	// case 5: uses a configuration file with a wrong HTML command template.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-html-command-template-gotenberg.yml")
	if _, err := NewAppConfig(path); err == nil {
		t.Error("AppConfig should not have been instantiated!")
	}

	// case 6: uses a configuration file with a wrong Office command template.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-office-command-template-gotenberg.yml")
	if _, err := NewAppConfig(path); err == nil {
		t.Error("AppConfig should not have been instantiated!")
	}

	// case 7: uses a configuration file with a wrong merge command template.
	path, _ = filepath.Abs("../../_tests/configurations/wrong-merge-command-template-gotenberg.yml")
	if _, err := NewAppConfig(path); err == nil {
		t.Error("AppConfig should not have been instantiated!")
	}

	// case 8: uses a correct configuration file.
	path, _ = filepath.Abs("../../_tests/configurations/gotenberg.yml")
	if _, err := NewAppConfig(path); err != nil {
		t.Error("AppConfig should have been instantiated!")
	}
}

func TestWrongLoggingLevelError(t *testing.T) {
	err := &wrongLoggingLevelError{}
	if err.Error() != wrongLoggingLevelErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), wrongLoggingLevelErrorMessage)
	}
}

func TestWrongLoggingFormatError(t *testing.T) {
	err := &wrongLoggingFormatError{}
	if err.Error() != wrongLoggingFormatErrorMessage {
		t.Errorf("Error returned a wrong message: got %s want %s", err.Error(), wrongLoggingFormatErrorMessage)
	}
}
