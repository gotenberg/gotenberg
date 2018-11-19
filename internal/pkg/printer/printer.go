package printer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	pdfcpuAPI "github.com/hhrutter/pdfcpu/pkg/api"
	pdfcpuConfig "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

// Printer is a type that can create a PDF file from a source.
// The source is defined in the underlying implementation.
type Printer interface {
	Print(destination string) error
}

// Merge merges all PDF files from a directory.
func Merge(dirPath, destination string) error {
	if !fileExists(dirPath) {
		return fmt.Errorf("%s: directory does not exist", dirPath)
	}
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".pdf" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: walking in directory", dirPath)
	}
	cmd := pdfcpuAPI.MergeCommand(files, destination, pdfcpuConfig.NewDefaultConfiguration())
	_, err = pdfcpuAPI.Merge(cmd)
	return err
}

func writeBytesToFile(dst string, b []byte) error {
	if err := ioutil.WriteFile(dst, b, 0644); err != nil {
		return fmt.Errorf("%s: writting file: %v", dst, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
