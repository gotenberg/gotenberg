package api

import (
	"fmt"

	"github.com/labstack/echo"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

func merge(c echo.Context) error {
	r, err := newResource(c)
	if err != nil {
		return err
	}
	defer r.removeAll()
	fpaths, err := r.filePaths([]string{".pdf"})
	if err != nil {
		return err
	}
	baseFilename, err := rand.Get()
	if err != nil {
		return fmt.Errorf("getting result file name: %v", err)
	}
	filename := fmt.Sprintf("%s.pdf", baseFilename)
	dest := fmt.Sprintf("%s/%s", r.dirPath, filename)
	return printer.Merge(fpaths, dest)
}
