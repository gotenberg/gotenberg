package process

import (
	"bytes"
	"fmt"
	"os/exec"
	"text/template"
	"time"

	"github.com/gulien/gotenberg/app/config"
	gfile "github.com/gulien/gotenberg/app/handlers/converter/file"
)

var commandsConfig *config.CommandsConfig

func Load(config *config.CommandsConfig) {
	commandsConfig = config
}

func Reset() {
	commandsConfig = nil
}

type conversionData struct {
	FilePath       string
	ResultFilePath string
}

func ExecConversion(file *gfile.File) (string, error) {
	cmdData := &conversionData{
		FilePath:       file.Path,
		ResultFilePath: fmt.Sprintf("%s%s", gfile.MakeFilePath(), gfile.PDFExt),
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

	err := execCommand(data.String(), cmdTimeout)
	if err != nil {
		return "", err
	}

	return cmdData.ResultFilePath, nil
}

type mergeData struct {
	FilesPaths     []string
	ResultFilePath string
}

func ExecMerge(filesPaths []string) (string, error) {
	cmdData := &mergeData{
		FilesPaths:     filesPaths,
		ResultFilePath: fmt.Sprintf("%s%s", gfile.MakeFilePath(), gfile.PDFExt),
	}

	cmdTemplate := commandsConfig.Merge.Template
	cmdTimeout := commandsConfig.Merge.Timeout

	var data bytes.Buffer
	if err := cmdTemplate.Execute(&data, cmdData); err != nil {
		return "", err
	}

	err := execCommand(data.String(), cmdTimeout)
	if err != nil {
		return "", err
	}

	return cmdData.ResultFilePath, nil
}

func execCommand(command string, timeout int) error {
	// Wait for the process to finish or kill it after a timeout.
	cmd := exec.Command("/bin/sh", "-c", command)
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

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
