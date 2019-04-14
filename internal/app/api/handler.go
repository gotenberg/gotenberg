package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

type errBadRequest struct {
	err error
}

func (e *errBadRequest) Error() string {
	return e.err.Error()
}

func merge(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.mergePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	fpaths, err := ctx.resource.fpaths(".pdf")
	if err != nil {
		return &errBadRequest{err}
	}
	p := printer.NewMerge(fpaths, opts)
	return convert(ctx, p)
}

func convertHTML(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.chromePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	fpath, err := ctx.resource.fpath("index.html")
	if err != nil {
		return &errBadRequest{err}
	}
	p := printer.NewHTML(fpath, opts)
	return convert(ctx, p)
}

func convertMarkdown(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.chromePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	fpath, err := ctx.resource.fpath("index.html")
	if err != nil {
		return &errBadRequest{err}
	}
	p, err := printer.NewMarkdown(fpath, opts)
	if err != nil {
		return err
	}
	return convert(ctx, p)
}

func convertURL(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.chromePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	remote, err := ctx.resource.get(remoteURL)
	if err != nil {
		return &errBadRequest{err}
	}
	p := printer.NewURL(remote, opts)
	return convert(ctx, p)
}

func convertOffice(c echo.Context) error {
	ctx := c.(*resourceContext)
	opts, err := ctx.resource.officePrinterOptions()
	if err != nil {
		return &errBadRequest{err}
	}
	fpaths, err := ctx.resource.fpaths(
		".txt",
		".rtf",
		".fodt",
		".doc",
		".docx",
		".odt",
		".xls",
		".xlsx",
		".ods",
		".ppt",
		".pptx",
		".odp",
	)
	if err != nil {
		return &errBadRequest{err}
	}
	p := printer.NewOffice(fpaths, opts)
	return convert(ctx, p)
}

func convert(ctx *resourceContext, p printer.Printer) error {
	baseFilename, err := rand.Get()
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s.pdf", baseFilename)
	fpath := fmt.Sprintf("%s/%s", ctx.resource.formFilesDirPath, filename)
	// if no webhook URL given, run conversion
	// and directly return the resulting PDF file
	// or an error.
	if !ctx.resource.has(webhookURL) {
		if err := p.Print(fpath); err != nil {
			return err
		}
		if ctx.resource.has(resultFilename) {
			filename, err = ctx.resource.get(resultFilename)
			if err != nil {
				return &errBadRequest{err}
			}
		}
		return ctx.Attachment(fpath, filename)
	}
	// as a webhook URL has been given, we
	// run the following lines in a goroutine so that
	// it doesn't block.
	go func() {
		defer ctx.resource.close() // nolint: errcheck
		if err := p.Print(fpath); err != nil {
			ctx.Logger().Error(err)
			return
		}
		f, err := os.Open(fpath)
		if err != nil {
			ctx.Logger().Error(err)
			return
		}
		defer f.Close() // nolint: errcheck
		webhook, err := ctx.resource.get(webhookURL)
		if err != nil {
			ctx.Logger().Error(err)
			return
		}
		resp, err := http.Post(webhook, "application/pdf", f) /* #nosec */
		if err != nil {
			ctx.Logger().Error(err)
			return
		}
		defer resp.Body.Close() // nolint: errcheck
	}()
	return nil
}
