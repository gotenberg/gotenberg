package xhttp

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/context"
	"github.com/thecodingmachine/gotenberg/internal/app/xhttp/pkg/resource"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
)

func pingEndpoint(config conf.Config) string {
	return fmt.Sprintf("%s%s", config.RootPath(), "ping")
}

func mergeEndpoint(config conf.Config) string {
	return fmt.Sprintf("%s%s", config.RootPath(), "merge")
}

func htmlEndpoint(config conf.Config) string {
	return fmt.Sprintf("%s%s", config.RootPath(), "convert/html")
}

func urlEndpoint(config conf.Config) string {
	return fmt.Sprintf("%s%s", config.RootPath(), "convert/url")
}

func markdownEndpoint(config conf.Config) string {
	return fmt.Sprintf("%s%s", config.RootPath(), "convert/markdown")
}

func officeEndpoint(config conf.Config) string {
	return fmt.Sprintf("%s%s", config.RootPath(), "convert/office")
}

func isMultipartFormDataEndpoint(config conf.Config, path string) bool {
	var multipartFormDataEndpoints []string
	multipartFormDataEndpoints = append(multipartFormDataEndpoints, mergeEndpoint(config))
	if !config.DisableGoogleChrome() {
		multipartFormDataEndpoints = append(
			multipartFormDataEndpoints,
			htmlEndpoint(config),
			urlEndpoint(config),
			markdownEndpoint(config),
		)
	}
	if !config.DisableUnoconv() {
		multipartFormDataEndpoints = append(
			multipartFormDataEndpoints,
			officeEndpoint(config),
		)
	}
	for _, endpoint := range multipartFormDataEndpoints {
		if endpoint == path {
			return true
		}
	}
	return false
}

// pingHandler is the handler for healthcheck.
func pingHandler(c echo.Context) error {
	const op string = "xhttp.pingHandler"
	ctx := context.MustCastFromEchoContext(c)
	logger := ctx.XLogger()
	logger.DebugOp(op, "handling ping request...")
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
		r := ctx.MustResource()
		opts, err := mergePrinterOptions(r, ctx.Config())
		if err != nil {
			return xerror.New(op, err)
		}
		fpaths, err := r.Fpaths(".pdf")
		if err != nil {
			return err
		}
		p := printer.NewMergePrinter(logger, fpaths, opts)
		return convert(ctx, p)
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
		r := ctx.MustResource()
		opts, err := chromePrinterOptions(r, ctx.Config())
		if err != nil {
			return err
		}
		fpath, err := r.Fpath("index.html")
		if err != nil {
			return err
		}
		p := printer.NewHTMLPrinter(logger, fpath, opts)
		return convert(ctx, p)
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
		r := ctx.MustResource()
		opts, err := chromePrinterOptions(r, ctx.Config())
		if err != nil {
			return err
		}
		opts.CustomHTTPHeaders = resource.RemoteURLCustomHTTPHeaders(r)
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
		p := printer.NewURLPrinter(logger, remoteURL, opts)
		return convert(ctx, p)
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
		r := ctx.MustResource()
		opts, err := chromePrinterOptions(r, ctx.Config())
		if err != nil {
			return err
		}
		fpath, err := r.Fpath("index.html")
		if err != nil {
			return err
		}
		p, err := printer.NewMarkdownPrinter(logger, fpath, opts)
		if err != nil {
			return err
		}
		return convert(ctx, p)
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
		r := ctx.MustResource()
		opts, err := officePrinterOptions(r, ctx.Config())
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
		p := printer.NewOfficePrinter(logger, fpaths, opts)
		return convert(ctx, p)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func convert(ctx context.Context, p printer.Printer) error {
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
			return convertSync(ctx, p, filename, fpath)
		}
		// as a webhook URL has been given, we
		// run the following lines in a goroutine so that
		// it doesn't block.
		logger.DebugfOp(op, "'%s' found, converting asynchronously", resource.WebhookURLArgKey)
		return convertAsync(ctx, p, filename, fpath)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func convertSync(ctx context.Context, p printer.Printer, filename, fpath string) error {
	const op = "xhttp.convertSync"
	resolver := func() error {
		logger := ctx.XLogger()
		r := ctx.MustResource()

		if err := p.Print(fpath); err != nil {
			return err
		}
		if !r.HasArg(resource.ResultFilenameArgKey) {
			logger.DebugfOp(
				op,
				"no '%s' found, using generated filename '%s'",
				resource.RemoteURLArgKey,
				filename,
			)
			if err := ctx.Attachment(fpath, filename); err != nil {
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
		if err := ctx.Attachment(fpath, filename); err != nil {
			return err
		}
		return nil
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func convertAsync(ctx context.Context, p printer.Printer, filename, fpath string) error {
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
		if err := p.Print(fpath); err != nil {
			xerr := xerror.New(op, err)
			logger.ErrorOp(xerror.Op(xerr), xerr)
			return
		}
		f, err := os.Open(fpath)
		if err != nil {
			xerr := xerror.New(op, err)
			logger.ErrorOp(xerror.Op(xerr), xerr)
			return
		}
		defer f.Close() // nolint: errcheck
		logger.DebugfOp(
			op,
			"preparing to send result file '%s' to '%s'...",
			filename,
			webhookURL,
		)
		httpClient := &http.Client{
			Timeout: xtime.Duration(webhookURLTimeout),
		}
		req, err := http.NewRequest(http.MethodPost, webhookURL, f)
		if err != nil {
			xerr := xerror.New(op, err)
			logger.ErrorOp(xerror.Op(xerr), xerr)
			return
		}
		req.Header.Set(echo.HeaderContentType, "application/pdf")
		// set custom headers (if any).
		customHeaders := resource.WebhookURLCustomHTTPHeaders(r)
		if len(customHeaders) > 0 {
			for key, value := range customHeaders {
				req.Header.Set(key, value)
				logger.DebugfOp(op, "set '%s' to custom HTTP header '%s'", value, key)
			}
		} else {
			logger.DebugOp(op, "skipping custom HTTP headers as none have been provided...")
		}
		// send the result file.
		logger.DebugfOp(
			op,
			"sending result file '%s' to '%s'...",
			filename,
			webhookURL,
		)
		resp, err := httpClient.Do(req) /* #nosec */
		if err != nil {
			xerr := xerror.New(op, err)
			logger.ErrorOp(xerror.Op(xerr), xerr)
			return
		}
		defer resp.Body.Close() // nolint: errcheck
		logger.DebugfOp(
			op,
			"result file '%s' sent to '%s'",
			filename,
			webhookURL,
		)
	}()
	return nil
}
