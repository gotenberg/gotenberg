package xhttp

import (
	timeoutContext "context"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xcontext"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
)

const (
	pingEndpoint         string = "/ping"
	mergeEndpoint        string = "/merge"
	convertGroupEndpoint string = "/convert"
	htmlEndpoint         string = "/html"
	urlEndpoint          string = "/url"
	markdownEndpoint     string = "/markdown"
	officeEndpoint       string = "/office"
)

// pingHandler is the handler for healthcheck.
func pingHandler(c echo.Context) error {
	const op string = "xhttp.pingHandler"
	ctx := context.MustCastFromEchoContext(c)
	logger := ctx.XLogger()
	logger.DebugOp(op, "handling ping request...")
	resolver := func() error {
		if err := ctx.ProcessesHealthcheck(); err != nil {
			return err
		}
		if logger.Level() != xlog.DebugLevel {
			return nil
		}
		// TODO
		list, err := pm2.List()
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, list)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// mergeHandler is the handler for merging
// PDF files.
func mergeHandler(c echo.Context) error {
	const op string = "xhttp.mergeHandler"
	resolver := func() error {
		ctx := context.MustCastFromEchoContext(c)
		logger := ctx.XLogger()
		logger.DebugOp(op, "handling merge request...")
		config := ctx.Config()
		r := ctx.MustResource()
		timeout, err := resource.WaitTimeoutAndWaitDelayArg(r, config)
		if err != nil {
			return err
		}
		fpaths, err := r.Fpaths(".pdf")
		if err != nil {
			return err
		}
		prinry := ctx.Prinery()
		printFunc := func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error {
			return prinry.Merge(ctx, logger, dest, fpaths)
		}
		return convert(ctx, timeout, printFunc)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// htmlHandler is the handler for converting
// HTML to PDF.
func htmlHandler(c echo.Context) error {
	const op string = "xhttp.htmlHandler"
	resolver := func() error {
		ctx := context.MustCastFromEchoContext(c)
		logger := ctx.XLogger()
		logger.DebugOp(op, "handling HTML request...")
		config := ctx.Config()
		r := ctx.MustResource()
		timeout, err := resource.WaitTimeoutAndWaitDelayArg(r, config)
		if err != nil {
			return err
		}
		opts, err := chromePrintOptions(r, config)
		if err != nil {
			return err
		}
		fpath, err := r.Fpath("index.html")
		if err != nil {
			return err
		}
		prinry := ctx.Prinery()
		printFunc := func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error {
			return prinry.HTML(ctx, logger, dest, fpath, opts)
		}
		return convert(ctx, timeout, printFunc)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// urlHandler is the handler for converting
// a URL to PDF.
func urlHandler(c echo.Context) error {
	const op string = "xhttp.urlHandler"
	resolver := func() error {
		ctx := context.MustCastFromEchoContext(c)
		logger := ctx.XLogger()
		logger.DebugOp(op, "handling URL request...")
		config := ctx.Config()
		r := ctx.MustResource()
		timeout, err := resource.WaitTimeoutAndWaitDelayArg(r, config)
		if err != nil {
			return err
		}
		opts, err := chromePrintOptions(r, config)
		if err != nil {
			return err
		}
		if !r.HasArg(resource.RemoteURLArgKey) {
			return xerror.Invalid(
				op,
				fmt.Sprintf("'%s' not found or empty", resource.RemoteURLArgKey),
				nil,
			)
		}
		remoteURL, err := r.StringArg(resource.RemoteURLArgKey, "")
		if err != nil {
			return err
		}
		prinry := ctx.Prinery()
		printFunc := func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error {
			return prinry.URL(ctx, logger, dest, remoteURL, opts)
		}
		return convert(ctx, timeout, printFunc)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// markdownHandler is the handler for converting
// Markdown to PDF.
func markdownHandler(c echo.Context) error {
	const op string = "xhttp.markdownHandler"
	resolver := func() error {
		ctx := context.MustCastFromEchoContext(c)
		logger := ctx.XLogger()
		logger.DebugOp(op, "handling Markdown request...")
		config := ctx.Config()
		r := ctx.MustResource()
		timeout, err := resource.WaitTimeoutAndWaitDelayArg(r, config)
		if err != nil {
			return err
		}
		opts, err := chromePrintOptions(r, config)
		if err != nil {
			return err
		}
		fpath, err := r.Fpath("index.html")
		if err != nil {
			return err
		}
		prinry := ctx.Prinery()
		printFunc := func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error {
			return prinry.Markdown(ctx, logger, dest, fpath, opts)
		}
		return convert(ctx, timeout, printFunc)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// officeHandler is the handler for converting
// Office documents to PDF.
func officeHandler(c echo.Context) error {
	const op string = "xhttp.officeHandler"
	resolver := func() error {
		ctx := context.MustCastFromEchoContext(c)
		logger := ctx.XLogger()
		logger.DebugOp(op, "handling Office request...")
		config := ctx.Config()
		r := ctx.MustResource()
		timeout, err := resource.WaitTimeoutAndWaitDelayArg(r, config)
		if err != nil {
			return err
		}
		opts, err := unoconvPrintOptions(r, config)
		if err != nil {
			return err
		}
		fpaths, err := r.Fpaths(
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
			return err
		}
		prinry := ctx.Prinery()
		printFunc := func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error {
			return prinry.Office(ctx, logger, dest, fpaths, opts)
		}
		return convert(ctx, timeout, printFunc)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func convert(
	ctx context.Context,
	timeout float64,
	printFunc func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error,
) error {
	const op string = "xhttp.convert"
	resolver := func() error {
		logger := ctx.XLogger()
		r := ctx.MustResource()
		baseFilename := xrand.Get()
		filename := fmt.Sprintf("%s.pdf", baseFilename)
		fpath := fmt.Sprintf("%s/%s", r.DirPath(), filename)
		// if no webhook URL given, run conversion
		// and directly return the resulting PDF file
		// or an error.
		if !r.HasArg(resource.WebhookURLArgKey) {
			logger.DebugfOp(op, "no '%s' found, converting synchronously", resource.WebhookURLArgKey)
			return convertSync(ctx, timeout, filename, fpath, printFunc)
		}
		// as a webhook URL has been given, we
		// run the following lines in a goroutine so that
		// it doesn't block.
		logger.DebugfOp(op, "'%s' found, converting asynchronously", resource.WebhookURLArgKey)
		return convertAsync(ctx, timeout, filename, fpath, printFunc)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func convertSync(
	ctx context.Context,
	timeout float64,
	filename, dest string,
	printFunc func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error,
) error {
	const op = "xhttp.convertSync"
	logger := ctx.XLogger()
	r := ctx.MustResource()
	timeoutCtx, cancel := xcontext.WithTimeout(logger, timeout)
	defer cancel()
	resolver := func() error {
		if err := printFunc(timeoutCtx, logger, dest); err != nil {
			return err
		}
		if !r.HasArg(resource.ResultFilenameArgKey) {
			logger.DebugfOp(
				op,
				"no '%s' found, using generated filename '%s'",
				resource.RemoteURLArgKey,
				filename,
			)
			if err := ctx.Attachment(dest, filename); err != nil {
				return err
			}
			return nil
		}
		logger.DebugfOp(
			op,
			"'%s' found, so not using generated filename",
			resource.ResultFilenameArgKey,
		)
		filename, err := r.StringArg(resource.ResultFilenameArgKey, filename)
		if err != nil {
			return err
		}
		if err := ctx.Attachment(dest, filename); err != nil {
			return err
		}
		return nil
	}
	if err := resolver(); err != nil {
		return xcontext.MustHandleError(
			timeoutCtx,
			xerror.New(op, err),
		)
	}
	return nil
}

func convertAsync(
	ctx context.Context,
	timeout float64,
	filename, dest string,
	printFunc func(ctx timeoutContext.Context, logger xlog.Logger, dest string) error,
) error {
	const op = "xhttp.convertAsync"
	logger := ctx.XLogger()
	r := ctx.MustResource()
	webhookURL, err := r.StringArg(resource.WebhookURLArgKey, "")
	if err != nil {
		return xerror.New(op, err)
	}
	webhookURLTimeout, err := resource.WebhookURLTimeoutArg(r, ctx.Config())
	if err != nil {
		return xerror.New(op, err)
	}
	go func() {
		defer r.Close() // nolint: errcheck
		timeoutCtx, cancel := xcontext.WithTimeout(logger, timeout)
		defer cancel()
		if err := printFunc(timeoutCtx, logger, dest); err != nil {
			xerr := xerror.New(op, err)
			logger.ErrorOp(xerror.Op(xerr), xerr)
			return
		}
		f, err := os.Open(dest)
		if err != nil {
			xerr := xerror.New(op, err)
			logger.ErrorOp(xerror.Op(xerr), xerr)
			return
		}
		defer f.Close() // nolint: errcheck
		logger.DebugfOp(
			op,
			"sending result file '%s' to '%s'",
			filename,
			webhookURL,
		)
		httpClient := &http.Client{
			Timeout: xtime.Duration(webhookURLTimeout),
		}
		resp, err := httpClient.Post(webhookURL, "application/pdf", f) /* #nosec */
		if err != nil {
			xerr := xerror.New(op, err)
			logger.ErrorOp(xerror.Op(xerr), xerr)
			return
		}
		defer resp.Body.Close() // nolint: errcheck
	}()
	return nil
}
