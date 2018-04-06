// Package process handles all commands executions.
package process

import (
	"bytes"
	"fmt"
	"os/exec"
	"text/template"
	"time"

	"github.com/thecodingmachine/gotenberg/app/config"
	gfile "github.com/thecodingmachine/gotenberg/app/converter/file"
)

var commandsConfig *config.CommandsConfig

// Load loads the commands configuration coming from the application configuration.
func Load(config *config.CommandsConfig) {
	commandsConfig = config
}

// conversionData will be applied to the data-driven templates of conversions commands.
type conversionData struct {
	FilePath       string
	ResultFilePath string
}

type impossibleConversionError struct{}

const impossibleConversionErrorMessage = "Impossible conversion"

func (e *impossibleConversionError) Error() string {
	return impossibleConversionErrorMessage
}

// Unconv converts a file to PDF and returns the new file path.
func Unconv(workingDir string, file *gfile.File) (string, error) {
	cmdData := &conversionData{
		FilePath:       file.Path,
		ResultFilePath: fmt.Sprintf("%s%s", gfile.MakeFilePath(workingDir), gfile.PDFExt),
	}

	var (
		cmdTemplate *template.Template
		cmdTimeout  int
	)

	switch file.Type {
	case gfile.HTMLType:
		cmdTemplate = commandsConfig.HTML.Template
		cmdTimeout = commandsConfig.HTML.Timeout
		break
	case gfile.OfficeType:
		cmdTemplate = commandsConfig.Office.Template
		cmdTimeout = commandsConfig.Office.Timeout
		break
	default:
		return "", &impossibleConversionError{}
	}

	var data bytes.Buffer
	if err := cmdTemplate.Execute(&data, cmdData); err != nil {
		return "", err
	}

	err := run(data.String(), cmdTimeout)
	if err != nil {
		return "", err
	}

	return cmdData.ResultFilePath, nil
}

// mergeData will be applied to the data-driven template of the merge command.
type mergeData struct {
	FilesPaths     []string
	ResultFilePath string
}

// Merge merges many PDF files to one unique PDF file and returns the new file path.
func Merge(workingDir string, filesPaths []string) (string, error) {
	cmdData := &mergeData{
		FilesPaths:     filesPaths,
		ResultFilePath: fmt.Sprintf("%s%s", gfile.MakeFilePath(workingDir), gfile.PDFExt),
	}

	cmdTemplate := commandsConfig.Merge.Template
	cmdTimeout := commandsConfig.Merge.Timeout

	var data bytes.Buffer
	if err := cmdTemplate.Execute(&data, cmdData); err != nil {
		return "", err
	}

	err := run(data.String(), cmdTimeout)
	if err != nil {
		return "", err
	}

	return cmdData.ResultFilePath, nil
}

type commandTimeoutError struct{}

const commandTimeoutErrorMessage = "The command has reached timeout"

func (e *commandTimeoutError) Error() string {
	return commandTimeoutErrorMessage
}

// run runs the given command. If timeout is reached or
// something bad happened, returns an error.
func run(command string, timeout int) error {
	cmd := exec.Command("/bin/sh", "-c", command)
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// wait for the process to finish or kill it after a timeout.
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return err
		}

		return &commandTimeoutError{}
	case err := <-done:
		if err != nil {
			return err
		}

		return nil
	}
}
