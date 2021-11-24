package chromium

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func init() {
	gotenberg.MustRegisterModule(Chromium{})
}

var (
	// ErrURLNotAuthorized happens if a URL is not acceptable according to the
	// allowed/denied lists.
	ErrURLNotAuthorized = errors.New("URL not authorized")

	// ErrInvalidEmulatedMediaType happens if the emulated media type is not
	// "screen" nor "print". Empty value are allowed though.
	ErrInvalidEmulatedMediaType = errors.New("invalid emulated media type")

	// ErrInvalidEvaluationExpression happens if an evaluation expression
	// returns an exception or undefined.
	ErrInvalidEvaluationExpression = errors.New("invalid evaluation expression")

	// ErrInvalidPrinterSettings happens if the Options have one or more
	// aberrant values.
	ErrInvalidPrinterSettings = errors.New("invalid printer settings")

	// ErrPageRangesSyntaxError happens if the Options have an invalid page
	// ranges.
	ErrPageRangesSyntaxError = errors.New("page ranges syntax error")

	// ErrRpccMessageTooLarge happens when the messages received by
	// ChromeDevTools are larger than 100 MB.
	ErrRpccMessageTooLarge = errors.New("rpcc message too large")

	// ErrConsoleExceptions happens when there are exceptions in the Chromium
	// console. It also happens only if the Options.FailOnConsoleExceptions is
	// set to true.
	ErrConsoleExceptions = errors.New("console exceptions")
)

// Chromium is a module which provides both an API and routes for converting
// HTML document to PDF.
type Chromium struct {
	binPath                  string
	engine                   gotenberg.PDFEngine
	userAgent                string
	incognito                bool
	ignoreCertificateErrors  bool
	disableWebSecurity       bool
	allowFileAccessFromFiles bool
	proxyServer              string
	allowList                *regexp.Regexp
	denyList                 *regexp.Regexp
	disableJavaScript        bool
	disableRoutes            bool
}

// LinkTag represents an HTML <link> element.
type LinkTag struct {
	// Href is the "href" attribute of the HTML <link> element.
	// Required.
	Href string `json:"href"`
}

// ScriptTag represents an HTML <script> element.
type ScriptTag struct {
	// Src is the "src" attribute of the HTML <link> element.
	// Required.
	Src string `json:"src"`
}

// Options are the available options for converting HTML document to PDF.
type Options struct {
	// FailOnConsoleExceptions sets if the conversion should fail if there are
	// exceptions in the Chromium console.
	// Optional.
	FailOnConsoleExceptions bool

	// WaitDelay is the duration to wait when loading an HTML document before
	// converting it to PDF.
	// Optional.
	WaitDelay time.Duration

	// WaitWindowStatus is the window.status value to wait for before
	// converting an HTML document to PDF.
	// Optional.
	WaitWindowStatus string

	// WaitForExpression is the custom JavaScript expression to wait before
	// converting an HTML document to PDF until it returns true
	// Optional.
	WaitForExpression string

	// UserAgent overrides the default User-Agent header.
	// Optional.
	UserAgent string

	// ExtraHTTPHeaders are the HTTP headers to send by Chromium while loading
	// the HTML document.
	// Optional.
	ExtraHTTPHeaders map[string]string

	// ExtraLinkTags are HTML <link> attributes that are added on the fly.
	// Optional.
	ExtraLinkTags []LinkTag

	// EmulatedMediaType is the media type to emulate, either "screen" or
	// "print".
	// Optional.
	EmulatedMediaType string

	// ExtraScriptTags are HTML <script> attributes that are added on the fly.
	// Optional.
	ExtraScriptTags []ScriptTag

	// Landscape sets the paper orientation.
	// Optional.
	Landscape bool

	// PrintBackground prints the background graphics.
	// Optional.
	PrintBackground bool

	// Scale is the scale of the page rendering.
	// Optional.
	Scale float64

	// PaperWidth is the paper width, in inches.
	// Optional.
	PaperWidth float64

	// PaperHeight is the paper height, in inches.
	// Optional.
	PaperHeight float64

	// MarginTop is the top margin, in inches.
	// Optional.
	MarginTop float64

	// MarginBottom is the bottom margin, in inches.
	// Optional.
	MarginBottom float64

	// MarginLeft is the left margin, in inches.
	// Optional.
	MarginLeft float64

	// MarginRight is the right margin, in inches.
	// Optional.
	MarginRight float64

	// Page ranges to print, e.g., '1-5, 8, 11-13'. Empty means all pages.
	// Optional.
	PageRanges string

	// HeaderTemplate is the HTML template of the header. It should be valid
	// HTML  markup with following classes used to inject printing values into
	// them:
	// - date: formatted print date
	// - title: document title
	// - url: document location
	// - pageNumber: current page number
	// - totalPages: total pages in the document
	// For example, <span class=title></span> would generate span containing
	// the title.
	// Optional.
	HeaderTemplate string

	// FooterTemplate is the HTML template of the footer. It should use the
	// same format as the HeaderTemplate.
	// Optional.
	FooterTemplate string

	// PreferCSSPageSize defines whether to prefer page size as defined by CSS.
	// If false, the content will be scaled to fit the paper size.
	// Optional.
	PreferCSSPageSize bool
}

// DefaultOptions returns the default values for Options.
func DefaultOptions() Options {
	return Options{
		FailOnConsoleExceptions: false,
		WaitDelay:               0,
		WaitWindowStatus:        "",
		WaitForExpression:       "",
		UserAgent:               "",
		ExtraHTTPHeaders:        nil,
		ExtraLinkTags:           nil,
		EmulatedMediaType:       "",
		ExtraScriptTags:         nil,
		Landscape:               false,
		PrintBackground:         false,
		Scale:                   1.0,
		PaperWidth:              8.5,
		PaperHeight:             11,
		MarginTop:               0.39,
		MarginBottom:            0.39,
		MarginLeft:              0.39,
		MarginRight:             0.39,
		PageRanges:              "",
		HeaderTemplate:          "<html><head></head><body></body></html>",
		FooterTemplate:          "<html><head></head><body></body></html>",
		PreferCSSPageSize:       false,
	}
}

// API helps to interact with Chromium for converting HTML documents to PDF.
type API interface {
	PDF(ctx context.Context, logger *zap.Logger, URL, outputPath string, options Options) error
}

// Provider is a module interface which exposes a method for creating an API
// for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(chromium.Provider))
//		chromium, _ := provider.(chromium.Provider).Chromium()
//	}
type Provider interface {
	Chromium() (API, error)
}

// Descriptor returns a Chromium's module descriptor.
func (mod Chromium) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "chromium",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("chromium", flag.ExitOnError)
			fs.String("chromium-user-agent", "", "Override the default User-Agent header")
			fs.Bool("chromium-incognito", false, "Start Chromium with incognito mode")
			fs.Bool("chromium-ignore-certificate-errors", false, "Ignore the certificate errors")
			fs.Bool("chromium-disable-web-security", false, "Don't enforce the same-origin policy")
			fs.Bool("chromium-allow-file-access-from-files", false, "Allow file:// URIs to read other file:// URIs")
			fs.String("chromium-proxy-server", "", "Set the outbound proxy server; this switch only affects HTTP and HTTPS requests")
			fs.String("chromium-allow-list", "", "Set the allowed URLs for Chromium using a regular expression")
			fs.String("chromium-deny-list", "^file:///[^tmp].*", "Set the denied URLs for Chromium using a regular expression")
			fs.Bool("chromium-disable-javascript", false, "Disable JavaScript")
			fs.Bool("chromium-disable-routes", false, "Disable the routes")

			err := fs.MarkDeprecated("chromium-user-agent", "use the userAgent form field instead")
			if err != nil {
				panic(fmt.Errorf("create deprecated flags for chromium module: %v", err))
			}

			return fs
		}(),
		New: func() gotenberg.Module { return new(Chromium) },
	}
}

// Provision sets the module properties.
func (mod *Chromium) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.userAgent = flags.MustString("chromium-user-agent")
	mod.ignoreCertificateErrors = flags.MustBool("chromium-ignore-certificate-errors")
	mod.disableWebSecurity = flags.MustBool("chromium-disable-web-security")
	mod.allowFileAccessFromFiles = flags.MustBool("chromium-allow-file-access-from-files")
	mod.proxyServer = flags.MustString("chromium-proxy-server")
	mod.allowList = flags.MustRegexp("chromium-allow-list")
	mod.denyList = flags.MustRegexp("chromium-deny-list")
	mod.disableJavaScript = flags.MustBool("chromium-disable-javascript")
	mod.disableRoutes = flags.MustBool("chromium-disable-routes")

	binPath, ok := os.LookupEnv("CHROMIUM_BIN_PATH")
	if !ok {
		return errors.New("CHROMIUM_BIN_PATH environment variable is not set")
	}

	mod.binPath = binPath

	provider, err := ctx.Module(new(gotenberg.PDFEngineProvider))
	if err != nil {
		return fmt.Errorf("get PDF engine provider: %w", err)
	}

	engine, err := provider.(gotenberg.PDFEngineProvider).PDFEngine()
	if err != nil {
		return fmt.Errorf("get PDF engine: %w", err)
	}

	mod.engine = engine

	return nil
}

// Validate validates the module properties.
func (mod Chromium) Validate() error {
	_, err := os.Stat(mod.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("chromium binary path does not exist: %w", err)
	}

	return nil
}

// Metrics returns the metrics.
func (mod Chromium) Metrics() ([]gotenberg.Metric, error) {
	return []gotenberg.Metric{
		{
			Name:        "chromium_active_instances_count",
			Description: "Current number of active Chromium instances.",
			Read: func() float64 {
				activeInstancesCountMu.RLock()
				defer activeInstancesCountMu.RUnlock()

				return activeInstancesCount
			},
		},
	}, nil
}

// Chromium returns an API for interacting with Chromium for converting HTML
// documents to PDF.
func (mod Chromium) Chromium() (API, error) {
	return mod, nil
}

// Routes returns the HTTP routes.
func (mod Chromium) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	return []api.Route{
		convertURLRoute(mod, mod.engine),
		convertHTMLRoute(mod, mod.engine),
		convertMarkdownRoute(mod, mod.engine),
	}, nil
}

// PDF converts a URL to PDF. It creates a dedicated Chromium instance.
// Substantial calls to this method may increase CPU and memory usage
// drastically. In such a scenario, the given context may also be done before
// the end of the conversion.
func (mod Chromium) PDF(ctx context.Context, logger *zap.Logger, URL, outputPath string, options Options) error {
	debug := debugLogger{logger: logger.Named("chromium.debug")}
	userProfileDirPath := gotenberg.NewDirPath()

	args := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.CombinedOutput(debug),
		chromedp.ExecPath(mod.binPath),
		chromedp.NoSandbox,
		// See:
		// https://github.com/gotenberg/gotenberg/issues/327
		// https://github.com/chromedp/chromedp/issues/904
		chromedp.DisableGPU,
		// See:
		// https://github.com/puppeteer/puppeteer/issues/661
		// https://github.com/puppeteer/puppeteer/issues/2410
		chromedp.Flag("font-render-hinting", "none"),
		chromedp.UserDataDir(userProfileDirPath),
	)

	if mod.userAgent != "" && options.UserAgent == "" {
		// Deprecated.
		args = append(args, chromedp.UserAgent(mod.userAgent))
	}

	if mod.incognito {
		args = append(args, chromedp.Flag("incognito", mod.incognito))
	}

	if mod.ignoreCertificateErrors {
		args = append(args, chromedp.IgnoreCertErrors)
	}

	if mod.disableWebSecurity {
		args = append(args, chromedp.Flag("disable-web-security", true))
	}

	if mod.allowFileAccessFromFiles {
		// See https://github.com/gotenberg/gotenberg/issues/356.
		args = append(args, chromedp.Flag("allow-file-access-from-files", true))
	}

	if mod.proxyServer != "" {
		// See https://github.com/gotenberg/gotenberg/issues/376.
		args = append(args, chromedp.ProxyServer(mod.proxyServer))
	}

	if options.UserAgent != "" {
		args = append(args, chromedp.UserAgent(options.UserAgent))
	}

	allocatorCtx, cancel := chromedp.NewExecAllocator(ctx, args...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocatorCtx,
		chromedp.WithDebugf(debug.Printf),
	)
	defer cancel()

	// We validate the "main" URL against our allow / deny lists.
	if !mod.allowList.MatchString(URL) {
		return fmt.Errorf("'%s' does not match the expression from the allowed list: %w", URL, ErrURLNotAuthorized)
	}

	if mod.denyList.String() != "" && mod.denyList.MatchString(URL) {
		return fmt.Errorf("'%s' matches the expression from the denied list: %w", URL, ErrURLNotAuthorized)
	}

	var (
		consoleExceptions   error
		consoleExceptionsMu sync.RWMutex
	)

	printToPDF := func(URL string, options Options, result *[]byte) chromedp.Tasks {
		// We validate the underlying requests against our allow / deny lists.
		// If a request does not pass the validation, we make it fail.
		listenForEventRequestPaused(taskCtx, logger, mod.allowList, mod.denyList)

		// See https://github.com/gotenberg/gotenberg/issues/262.
		if options.FailOnConsoleExceptions {
			listenForEventExceptionThrown(taskCtx, logger, &consoleExceptions, &consoleExceptionsMu)
		}

		return chromedp.Tasks{
			network.Enable(),
			fetch.Enable(),
			runtime.Enable(),
			chromedp.ActionFunc(func(ctx context.Context) error {
				// See https://github.com/gotenberg/gotenberg/issues/175.
				if !mod.disableJavaScript {
					logger.Debug("JavaScript not disabled")

					return nil
				}

				logger.Debug("disable JavaScript")

				err := emulation.SetScriptExecutionDisabled(true).Do(ctx)
				if err == nil {
					return nil
				}

				return fmt.Errorf("disable JavaScript: %w", err)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if len(options.ExtraHTTPHeaders) == 0 {
					logger.Debug("no extra HTTP headers")

					return nil
				}

				logger.Debug(fmt.Sprintf("extra HTTP headers: %+v", options.ExtraHTTPHeaders))

				headers := make(network.Headers, len(options.ExtraHTTPHeaders))
				for key, value := range options.ExtraHTTPHeaders {
					headers[key] = value
				}

				err := network.SetExtraHTTPHeaders(headers).Do(ctx)
				if err == nil {
					return nil
				}

				return fmt.Errorf("set extra HTTP headers: %w", err)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				logger.Debug(fmt.Sprintf("navigate to '%s'", URL))

				_, _, _, err := page.Navigate(URL).Do(ctx)
				if err != nil {
					return fmt.Errorf("navigate to '%s': %w", URL, err)
				}

				err = runBatch(
					ctx,
					waitForEventDomContentEventFired(ctx, logger),
					waitForEventLoadEventFired(ctx, logger),
					waitForEventNetworkIdle(ctx, logger),
					waitForEventLoadingFinished(ctx, logger),
				)

				if err == nil {
					return nil
				}

				return fmt.Errorf("wait for events: %w", err)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				// See:
				// https://github.com/gotenberg/gotenberg/issues/354
				// https://github.com/puppeteer/puppeteer/issues/2685
				// https://github.com/chromedp/chromedp/issues/520
				script := `
(() => {
	const css = 'html { -webkit-print-color-adjust: exact !important; }';

	const style = document.createElement('style');
	style.type = 'text/css';
	style.appendChild(document.createTextNode(css));
	document.head.appendChild(style);
})();
`

				evaluate := chromedp.Evaluate(script, nil)
				err := evaluate.Do(ctx)

				if err == nil {
					return nil
				}

				return fmt.Errorf("add CSS for exact colors: %w", err)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if len(options.ExtraLinkTags) == 0 {
					logger.Debug("no extra link tags")

					return nil
				}

				logger.Debug(fmt.Sprintf("extra link tags: %+v", options.ExtraLinkTags))

				addLinkTag := func(i int, linkTag LinkTag) func() error {
					return func() error {
						script := `
(() => {
	const link = document.createElement('link');
	link.href = '%s';
	link.rel = 'stylesheet'
	document.head.appendChild(link);
})();
`

						evaluate := chromedp.Evaluate(fmt.Sprintf(script, linkTag.Href), nil)
						err := evaluate.Do(ctx)

						if err == nil {
							return nil
						}

						return fmt.Errorf("add extra link tag %d: %w", i, err)
					}
				}

				eg, _ := errgroup.WithContext(ctx)

				for i, linkTag := range options.ExtraLinkTags {
					eg.Go(addLinkTag(i, linkTag))
				}

				err := eg.Wait()
				if err == nil {
					return nil
				}

				return fmt.Errorf("add extra link tags: %w", err)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if options.EmulatedMediaType == "" {
					logger.Debug("no emulated media type")

					return nil
				}

				if options.EmulatedMediaType != "screen" && options.EmulatedMediaType != "print" {
					return fmt.Errorf("validate emulated media type '%s': %w", options.EmulatedMediaType, ErrInvalidEmulatedMediaType)
				}

				logger.Debug(fmt.Sprintf("emulate media type '%s'", options.EmulatedMediaType))

				emulatedMedia := emulation.SetEmulatedMedia()

				err := emulatedMedia.WithMedia(options.EmulatedMediaType).Do(ctx)
				if err == nil {
					return nil
				}

				return fmt.Errorf("emulate media type '%s': %w", options.EmulatedMediaType, err)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if mod.disableJavaScript {
					logger.Debug("JavaScript disabled, skipping extra script tags")

					return nil
				}

				if len(options.ExtraScriptTags) == 0 {
					logger.Debug("no extra script tags")

					return nil
				}

				logger.Debug(fmt.Sprintf("extra script tags: %+v", options.ExtraScriptTags))

				addScriptTag := func(i int, scriptTag ScriptTag) func() error {
					return func() error {
						script := `
(() => {
	const script = document.createElement('script');
	script.src = '%s';
	document.head.appendChild(script);
})();
`

						evaluate := chromedp.Evaluate(fmt.Sprintf(script, scriptTag.Src), nil)
						err := evaluate.Do(ctx)

						if err == nil {
							return nil
						}

						return fmt.Errorf("add extra script tag %d: %w", i, err)
					}
				}

				eg, _ := errgroup.WithContext(ctx)

				for i, scriptTag := range options.ExtraScriptTags {
					eg.Go(addScriptTag(i, scriptTag))
				}

				err := eg.Wait()
				if err == nil {
					return nil
				}

				return fmt.Errorf("add extra script tags: %w", err)
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if mod.disableJavaScript {
					logger.Debug("JavaScript disabled, skipping wait delay")

					return nil
				}

				if options.WaitDelay <= 0 {
					logger.Debug("no wait delay")

					return nil
				}

				// We wait for a given amount of time so that JavaScript
				// scripts have a chance to finish before printing the page
				// to PDF.
				logger.Debug(fmt.Sprintf("wait '%s' before print", options.WaitDelay))

				select {
				case <-ctx.Done():
					return fmt.Errorf("wait delay: %w", ctx.Err())
				case <-time.After(options.WaitDelay):
					return nil
				}
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if mod.disableJavaScript {
					logger.Debug("JavaScript disabled, skipping wait for window status / wait for expression")

					return nil
				}

				if options.WaitWindowStatus == "" && options.WaitForExpression == "" {
					logger.Debug("no wait for window status nor wait for expression")

					return nil
				}

				evaluate := func(expression string) error {
					// We wait until the evaluation of the expression is true or
					// until the context is done.
					logger.Debug(fmt.Sprintf("wait until '%s' is true before print", expression))

					ticker := time.NewTicker(time.Duration(100) * time.Millisecond)

					for {
						select {
						case <-ctx.Done():
							ticker.Stop()

							return fmt.Errorf("context done while evaluating '%s': %w", expression, ctx.Err())
						case <-ticker.C:
							var ok bool

							evaluate := chromedp.Evaluate(expression, &ok)
							err := evaluate.Do(ctx)

							if err != nil {
								return fmt.Errorf("evaluate: %v: %w", err, ErrInvalidEvaluationExpression)
							}

							if ok {
								ticker.Stop()

								return nil
							}

							continue
						}
					}
				}

				if options.WaitWindowStatus != "" {
					logger.Warn("option 'WaitWindowStatus' is deprecated; prefer 'WaitForExpression' instead")

					err := evaluate(fmt.Sprintf("window.status === '%s'", options.WaitWindowStatus))
					if err != nil {
						return fmt.Errorf("wait for window.status === '%s': %w", options.WaitWindowStatus, err)
					}
				}

				if options.WaitForExpression != "" {
					err := evaluate(options.WaitForExpression)
					if err != nil {
						return fmt.Errorf("wait for expression '%s': %w", options.WaitForExpression, err)
					}
				}

				return nil
			}),
			chromedp.ActionFunc(func(ctx context.Context) error {
				printToPDF := page.PrintToPDF().
					WithLandscape(options.Landscape).
					WithPrintBackground(options.PrintBackground).
					WithScale(options.Scale).
					WithPaperWidth(options.PaperWidth).
					WithPaperHeight(options.PaperHeight).
					WithMarginTop(options.MarginTop).
					WithMarginBottom(options.MarginBottom).
					WithMarginLeft(options.MarginLeft).
					WithMarginRight(options.MarginRight).
					WithIgnoreInvalidPageRanges(false).
					WithPageRanges(options.PageRanges).
					WithDisplayHeaderFooter(true).
					WithHeaderTemplate(options.HeaderTemplate).
					WithFooterTemplate(options.FooterTemplate).
					WithPreferCSSPageSize(options.PreferCSSPageSize)

				logger.Debug(fmt.Sprintf("print to PDF with: %+v", printToPDF))

				data, _, err := printToPDF.Do(ctx)
				if err != nil {
					return fmt.Errorf("print to PDF: %w", err)
				}

				*result = data

				return nil
			}),
		}
	}

	activeInstancesCountMu.Lock()
	activeInstancesCount += 1
	activeInstancesCountMu.Unlock()

	var buffer []byte
	err := chromedp.Run(taskCtx, printToPDF(URL, options, &buffer))

	activeInstancesCountMu.Lock()
	activeInstancesCount -= 1
	activeInstancesCountMu.Unlock()

	// Always remove the user profile directory created by Chromium.
	go func() {
		logger.Debug(fmt.Sprintf("remove user profile directory '%s'", userProfileDirPath))

		err := os.RemoveAll(userProfileDirPath)
		if err != nil {
			logger.Error(fmt.Sprintf("remove user profile directory: %s", err))
		}
	}()

	if err != nil {
		errMessage := err.Error()

		if strings.Contains(errMessage, "Show invalid printer settings error (-32000)") {
			return ErrInvalidPrinterSettings
		}

		if strings.Contains(errMessage, "Page range syntax error") {
			return ErrPageRangesSyntaxError
		}

		if strings.Contains(errMessage, "rpcc: message too large") {
			return ErrRpccMessageTooLarge
		}

		return fmt.Errorf("chromium PDF: %w", err)
	}

	// See https://github.com/gotenberg/gotenberg/issues/262.
	consoleExceptionsMu.RLock()
	defer consoleExceptionsMu.RUnlock()

	if consoleExceptions != nil {
		return fmt.Errorf("%v: %w", consoleExceptions, ErrConsoleExceptions)
	}

	err = ioutil.WriteFile(outputPath, buffer, 0600)
	if err != nil {
		return fmt.Errorf("write result to output path: %w", err)
	}

	return nil
}

var (
	activeInstancesCount   float64
	activeInstancesCountMu sync.RWMutex
)

// Interface guards.
var (
	_ gotenberg.Module          = (*Chromium)(nil)
	_ gotenberg.Provisioner     = (*Chromium)(nil)
	_ gotenberg.Validator       = (*Chromium)(nil)
	_ gotenberg.MetricsProvider = (*Chromium)(nil)
	_ api.Router                = (*Chromium)(nil)
	_ API                       = (*Chromium)(nil)
	_ Provider                  = (*Chromium)(nil)
)
