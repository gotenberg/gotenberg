package config

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestReset(t *testing.T) {
	c := &appConfig{}
	config.port = "3000"
	Reset()

	if c.port != config.port {
		t.Error("Configuration should have been reset")
	}
}

func TestWithPort(t *testing.T) {
	port := "3000"
	WithPort(port)

	if config.port != port {
		t.Errorf("Configuration populated with a wrong port: got '%s' want '%s'", config.port, port)
	}
}

func TestGetPort(t *testing.T) {
	port := "3000"
	config.port = port

	if GetPort() != port {
		t.Errorf("Configuration returned a wrong port: got '%s' want '%s'", GetPort(), port)
	}
}

func TestWrongLogsLevelError(t *testing.T) {
	err := &wrongLogsLevelError{}
	if err.Error() != wrongLogsLevelErrorMessage {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), wrongLogsLevelErrorMessage)
	}
}

func TestWithLogsLevel(t *testing.T) {
	var lvl string

	// case 1: uses a wrong logs level.
	lvl = "text"
	if err := WithLogsLevel(lvl); err == nil {
		t.Errorf("Configuration should not have been populated by using '%s' as logs level", lvl)
	}

	// case 2: uses a correct logs level.
	lvl = "DEBUG"
	if err := WithLogsLevel(lvl); err != nil {
		t.Errorf("Configuration should have been populated by using '%s' as logs level", lvl)
	}
}

func TestGetLogsLevel(t *testing.T) {
	lvl := logrus.DebugLevel
	config.logsLevel = lvl

	if GetLogsLevel() != lvl {
		t.Errorf("Configuration returned a wrong logs level: got '%s' want '%s'", GetLogsLevel(), lvl)
	}
}

func TestWrongLogsFormatterError(t *testing.T) {
	err := &wrongLogsFormatterError{}
	if err.Error() != wrongLogsFormatterErrorMessage {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), wrongLogsFormatterErrorMessage)
	}
}

func TestWithLogsFormatter(t *testing.T) {
	var formatter string

	// case 1: uses a wrong logs formatter.
	formatter = "DEBUG"
	if err := WithLogsFormatter(formatter); err == nil {
		t.Errorf("Configuration should not have been populated by using '%s' as logs formatter", formatter)
	}

	// case 2: uses a correct logs formatter.
	formatter = "text"
	if err := WithLogsFormatter(formatter); err != nil {
		t.Errorf("Configuration should have been populated by using '%s' as logs formatter", formatter)
	}
}

func TestGetLogsFormatter(t *testing.T) {
	formatter := &logrus.TextFormatter{}
	config.logsFormatter = formatter

	if GetLogsFormatter() != formatter {
		t.Errorf("Configuration returned a wrong logs formatter: got '%v' want '%v'", GetLogsFormatter(), formatter)
	}
}

func TestNewCommand(t *testing.T) {
	var cmd string

	// case 1: uses a wrong command template.
	cmd = "pdftk {{ range $filePath := FilesPaths }} {{ $filePath }} {{ end }} cat output {{ .ResultFilePath }}"
	if _, err := NewCommand(cmd, 0); err == nil {
		t.Errorf("Command should not have been instantiated by using '%s' as command template", cmd)
	}

	// case 2: uses a correct command template.
	cmd = "pdftk {{ range $filePath := .FilesPaths }} {{ $filePath }} {{ end }} cat output {{ .ResultFilePath }}"
	if _, err := NewCommand(cmd, 0); err != nil {
		t.Errorf("Command should have been instantiated by using '%s' as command template", cmd)
	}
}

func TestFileExtensionAlreadyUsedError(t *testing.T) {
	ext := ".pdf"
	cmd1, _ := NewCommand("echo", 0)
	cmd2, _ := NewCommand("echo", 0)
	err := &fileExtensionAlreadyUsedError{ext, cmd1, cmd2}
	expected := fmt.Sprintf(fileExtensionAlreadyUsedErrorMessage, err.extension, err.command.Template.Name(), err.existingCommand.Template.Name())

	if err.Error() != expected {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), expected)
	}
}

func TestWithCommand(t *testing.T) {
	ext := ".pdf"
	cmd, _ := NewCommand("echo", 0)

	// case 1: uses a command with a file extension not already referenced.
	if err := WithCommand(ext, cmd); err != nil {
		t.Errorf("Configuration should have been populated by using a command with the file extension '%s'", ext)
	}

	// case 2: uses a command with a file extension already referenced.
	if err := WithCommand(ext, cmd); err == nil {
		t.Errorf("Configuration should not have been populated by using a command with the file extension '%s'", ext)
	}
}

func TestNoCommandFoundForFileExtensionError(t *testing.T) {
	err := &noCommandFoundForFileExtensionError{".pdf"}
	expected := fmt.Sprintf(noCommandFoundForFileExtensionErrorMessage, err.extension)

	if err.Error() != expected {
		t.Errorf("Error returned a wrong message: got '%s' want '%s'", err.Error(), expected)
	}
}

func TestGetCommand(t *testing.T) {
	Reset()
	ext := ".pdf"
	cmd, _ := NewCommand("echo", 0)
	WithCommand(ext, cmd)

	// case 1: uses a file extension which has a command associated.
	if _, err := GetCommand(ext); err != nil {
		t.Errorf("Configuration should have been able to return a command by using the file extension '%s'", ext)
	}

	// case 2: uses a file extension which has no command associated.
	ext = ".docx"
	if _, err := GetCommand(ext); err == nil {
		t.Errorf("Configuration should not have been able to return a command by using the file extension '%s'", ext)
	}
}
