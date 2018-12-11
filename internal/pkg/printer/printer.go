package printer

import (
	"fmt"
	"io/ioutil"

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
	cmd := pdfcpuAPI.MergeCommand(fpaths, destination, pdfcpuConfig.NewDefaultConfiguration())
	_, err := pdfcpuAPI.Merge(cmd)
	return err
}

func writeBytesToFile(dst string, b []byte) error {
	if err := ioutil.WriteFile(dst, b, 0644); err != nil {
		return fmt.Errorf("%s: writing file: %v", dst, err)
	}
	return nil
}
