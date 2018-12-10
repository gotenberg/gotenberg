package api

import (
	"fmt"
	"net/http"
	"os"

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
	if r.webhookURL() == "" {
		// if no webhook URL given, run merge
		// and directly return the resulting PDF file
		// or and error.
		return printer.Merge(fpaths, dest)
	}
	// as a webhook URL has been given, we
	// run the following lines in a goroutine so that
	// it doesn't block.
	go func() {
		if err := printer.Merge(fpaths, dest); err != nil {
			c.Logger().Errorf("%v", err)
			return
		}
		f, err := os.Open(dest)
		if err != nil {
			c.Logger().Errorf("%v", err)
			return
		}
		defer f.Close()
		resp, err := http.Post(r.webhookURL(), "application/pdf", f)
		if err != nil {
			c.Logger().Errorf("%v", err)
			return
		}
		defer resp.Body.Close()
	}()
	return nil
}
