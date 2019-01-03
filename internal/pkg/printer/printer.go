package printer

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	pdfcpuAPI "github.com/hhrutter/pdfcpu/pkg/api"
	pdfcpuLog "github.com/hhrutter/pdfcpu/pkg/log"
	pdfcpuConfig "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

func init() {
	// disable loggers when merging
	// PDFs.
	pdfcpuLog.DisableLoggers()
}

// Printer is a type that can create a PDF file from a source.
// The source is defined in the underlying implementation.
type Printer interface {
	Print(destination string) error
}

// Merge merges PDF files.
func Merge(fpaths []string, destination string) error {
	cmdcpu := pdfcpuAPI.MergeCommand(fpaths, destination, pdfcpuConfig.NewDefaultConfiguration())
	_, err := pdfcpuAPI.Merge(cmdcpu)
	if err == nil {
		return nil
	}
	// if pdfcpu failed to merge PDF files...
	// https://github.com/thecodingmachine/gotenberg/issues/29
	var cmdArgs []string
	cmdArgs = append(cmdArgs, fpaths...)
	cmdArgs = append(cmdArgs, "cat", "output", destination)
	cmd := exec.Command("pdftk", cmdArgs...)
	return cmd.Run()
}

func writeBytesToFile(dst string, b []byte) error {
	if err := ioutil.WriteFile(dst, b, 0644); err != nil {
		return fmt.Errorf("%s: writing file: %v", dst, err)
	}
	return nil
}
