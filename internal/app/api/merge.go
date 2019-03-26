package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

func merge(c echo.Context) error {
	r, err := newResource(c)
	if err != nil {
		return hijackErr(err, r)
	}
	fpaths, err := r.filePaths([]string{".pdf"})
	if err != nil {
		return hijackErr(err, r)
	}
	if len(fpaths) == 0 {
		return hijackErr(errors.New("no suitable PDF files to merge"), r)
	}
	baseFilename, err := rand.Get()
	if err != nil {
		return hijackErr(fmt.Errorf("getting result file name: %v", err), r)
	}
	filename := fmt.Sprintf("%s.pdf", baseFilename)
	fpath := fmt.Sprintf("%s/%s", r.dirPath, filename)
	// if no webhook URL given, run merge
	// and directly return the resulting PDF file
	// or an error.
	if r.webhookURL() == "" {
		defer r.removeAll()
		if err := printer.Merge(fpaths, fpath); err != nil {
			return err
		}
		if r.filename() != "" {
			filename = r.filename()
		}
		return c.Attachment(fpath, filename)
	}
	// as a webhook URL has been given, we
	// run the following lines in a goroutine so that
	// it doesn't block.
	go func() {
		defer r.removeAll()
		if err := printer.Merge(fpaths, fpath); err != nil {
			c.Logger().Errorf("%v", err)
			return
		}
		f, err := os.Open(fpath)
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
