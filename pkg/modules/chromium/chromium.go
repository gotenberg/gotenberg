package chromium

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alexliesenfeld/health"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Chromium))
}

var (
	// ErrInvalidEmulatedMediaType happens if the emulated media type is not
	// "screen" nor "print". Empty value are allowed though.
	ErrInvalidEmulatedMediaType = errors.New("invalid emulated media type")

	// ErrInvalidEvaluationExpression happens if an evaluation expression
	// returns an exception or undefined.
	ErrInvalidEvaluationExpression = errors.New("invalid evaluation expression")

	// ErrRpccMessageTooLarge happens when the messages received by
	// ChromeDevTools are larger than 100 MB.
	ErrRpccMessageTooLarge = errors.New("rpcc message too large")

	// ErrInvalidHttpStatusCode happens when the status code from the main page
	// matches with one of the entry in [Options.FailOnHttpStatusCodes].
	ErrInvalidHttpStatusCode = errors.New("invalid HTTP status code")

	// ErrConsoleExceptions happens when there are exceptions in the Chromium
	// console. It also happens only if the [Options.FailOnConsoleExceptions]
	// is set to true.
	ErrConsoleExceptions = errors.New("console exceptions")

	// PDF specific.

	// ErrOmitBackgroundWithoutPrintBackground happens if
	// PdfOptions.OmitBackground is set to true but not PdfOptions.PrintBackground.
	ErrOmitBackgroundWithoutPrintBackground = errors.New("omit background without print background")

	// ErrInvalidPrinterSettings happens if the PdfOptions have one or more
	// aberrant values.
	ErrInvalidPrinterSettings = errors.New("invalid printer settings")

	// ErrPageRangesSyntaxError happens if the PdfOptions have an invalid page
	// ranges.
	ErrPageRangesSyntaxError = errors.New("page ranges syntax error")
)

// Chromium is a module which provides both an [Api] and routes for converting
// HTML document to PDF.
type Chromium struct {
	autoStart     bool
	disableRoutes bool
	args          browserArguments

	logger     *zap.Logger
	browser    browser
	supervisor gotenberg.ProcessSupervisor
	engine     gotenberg.PdfEngine
}

// Options are the common options for all conversions.
type Options struct {
	// SkipNetworkIdleEvent set if the conversion should wait for the
	// "networkIdle" event, drastically improving the conversion speed. It may
	// not be suitable for all HTML documents, as some may not be fully
	// rendered until this event is fired.
	// Optional.
	SkipNetworkIdleEvent bool

	// FailOnHttpStatusCodes sets if the conversion should fail if the status
	// code from the main page matches with one of its entries.
	// Optional.
	FailOnHttpStatusCodes []int64

	// FailOnConsoleExceptions sets if the conversion should fail if there are
	// exceptions in the Chromium console.
	// Optional.
	FailOnConsoleExceptions bool

	// WaitDelay is the duration to wait when loading an HTML document before
	// converting it.
	// Optional.
	WaitDelay time.Duration

	// WaitWindowStatus is the window.status value to wait for before
	// converting an HTML document.
	// Optional.
	WaitWindowStatus string

	// WaitForExpression is the custom JavaScript expression to wait before
	// converting an HTML document until it returns true
	// Optional.
	WaitForExpression string

	// ExtraHttpHeaders are the HTTP headers to send by Chromium while loading
	// the HTML document.
	// Optional.
	ExtraHttpHeaders map[string]string

	// EmulatedMediaType is the media type to emulate, either "screen" or
	// "print".
	// Optional.
	EmulatedMediaType string

	// OmitBackground hides default white background and allows generating PDFs
	// with transparency.
	// Optional.
	OmitBackground bool
}

// DefaultOptions returns the default values for Options.
func DefaultOptions() Options {
	return Options{
		SkipNetworkIdleEvent:    false,
		FailOnHttpStatusCodes:   []int64{499, 599},
		FailOnConsoleExceptions: false,
		WaitDelay:               0,
		WaitWindowStatus:        "",
		WaitForExpression:       "",
		ExtraHttpHeaders:        nil,
		EmulatedMediaType:       "",
		OmitBackground:          false,
	}
}

// PdfOptions are the available options for converting an HTML document to PDF.
type PdfOptions struct {
	Options

	// Landscape sets the paper orientation.
	// Optional.
	Landscape bool

	// PrintBackground prints the background graphics.
	// Optional.
	PrintBackground bool

	// Scale is the scale of the page rendering.
	// Optional.
	Scale float64

	// SinglePage defines whether to print the entire content in one single
	// page.
	// Optional.
	SinglePage bool

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

	// PreferCssPageSize defines whether to prefer page size as defined by CSS.
	// If false, the content will be scaled to fit the paper size.
	// Optional.
	PreferCssPageSize bool
}

// DefaultPdfOptions returns the default values for PdfOptions.
func DefaultPdfOptions() PdfOptions {
	return PdfOptions{
		Options:           DefaultOptions(),
		Landscape:         false,
		PrintBackground:   false,
		Scale:             1.0,
		SinglePage:        false,
		PaperWidth:        8.5,
		PaperHeight:       11,
		MarginTop:         0.39,
		MarginBottom:      0.39,
		MarginLeft:        0.39,
		MarginRight:       0.39,
		PageRanges:        "",
		HeaderTemplate:    "<html><head></head><body></body></html>",
		FooterTemplate:    "<html><head></head><body></body></html>",
		PreferCssPageSize: false,
	}
}

// ScreenshotOptions are the available options for capturing a screenshot from
// an HTML document.
type ScreenshotOptions struct {
	Options

	// Format is the image compression format, either "png" or "jpeg" or
	// "webp".
	// Optional.
	Format string

	// Quality is the compression quality from range [0..100] (jpeg only).
	// Optional.
	Quality int

	// OptimizeForSpeed defines whether to optimize image encoding for speed,
	// not for resulting size.
	// Optional.
	OptimizeForSpeed bool
}

// DefaultScreenshotOptions returns the default values for ScreenshotOptions.
func DefaultScreenshotOptions() ScreenshotOptions {
	return ScreenshotOptions{
		Options:          DefaultOptions(),
		Format:           "png",
		Quality:          100,
		OptimizeForSpeed: false,
	}
}

// Api helps to interact with Chromium for converting HTML documents to PDF.
type Api interface {
	Pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error
	Screenshot(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error
}

// Provider is a module interface which exposes a method for creating an [Api]
// for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(chromium.Provider))
//		api, _ := provider.(chromium.Provider).Chromium()
//	}
type Provider interface {
	Chromium() (Api, error)
}

// Descriptor returns a [Chromium]'s module descriptor.
func (mod *Chromium) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "chromium",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("chromium", flag.ExitOnError)
			fs.Int64("chromium-restart-after", 0, "Number of conversions after which Chromium will automatically restart. Set to 0 to disable this feature")
			fs.Int64("chromium-max-queue-size", 0, "Maximum request queue size for Chromium. Set to 0 to disable this feature")
			fs.Bool("chromium-auto-start", false, "Automatically launch Chromium upon initialization if set to true; otherwise, Chromium will start at the time of the first conversion")
			fs.Duration("chromium-start-timeout", time.Duration(20)*time.Second, "Maximum duration to wait for Chromium to start or restart")
			fs.Bool("chromium-incognito", false, "Start Chromium with incognito mode")
			fs.Bool("chromium-allow-insecure-localhost", false, "Ignore TLS/SSL errors on localhost")
			fs.Bool("chromium-ignore-certificate-errors", false, "Ignore the certificate errors")
			fs.Bool("chromium-disable-web-security", false, "Don't enforce the same-origin policy")
			fs.Bool("chromium-allow-file-access-from-files", false, "Allow file:// URIs to read other file:// URIs")
			fs.String("chromium-host-resolver-rules", "", "Set custom mappings to the host resolver")
			fs.String("chromium-proxy-server", "", "Set the outbound proxy server; this switch only affects HTTP and HTTPS requests")
			fs.String("chromium-allow-list", "", "Set the allowed URLs for Chromium using a regular expression")
			fs.String("chromium-deny-list", `^file:(?!//\/tmp/).*`, "Set the denied URLs for Chromium using a regular expression")
			fs.Bool("chromium-clear-cache", false, "Clear Chromium cache between each conversion")
			fs.Bool("chromium-clear-cookies", false, "Clear Chromium cookies between each conversion")
			fs.Bool("chromium-disable-javascript", false, "Disable JavaScript")
			fs.Bool("chromium-disable-routes", false, "Disable the routes")

			return fs
		}(),
		New: func() gotenberg.Module { return new(Chromium) },
	}
}

// Provision sets the module properties.
func (mod *Chromium) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.autoStart = flags.MustBool("chromium-auto-start")
	mod.disableRoutes = flags.MustBool("chromium-disable-routes")

	binPath, ok := os.LookupEnv("CHROMIUM_BIN_PATH")
	if !ok {
		return errors.New("CHROMIUM_BIN_PATH environment variable is not set")
	}

	mod.args = browserArguments{
		binPath:                  binPath,
		incognito:                flags.MustBool("chromium-incognito"),
		allowInsecureLocalhost:   flags.MustBool("chromium-allow-insecure-localhost"),
		ignoreCertificateErrors:  flags.MustBool("chromium-ignore-certificate-errors"),
		disableWebSecurity:       flags.MustBool("chromium-disable-web-security"),
		allowFileAccessFromFiles: flags.MustBool("chromium-allow-file-access-from-files"),
		hostResolverRules:        flags.MustString("chromium-host-resolver-rules"),
		proxyServer:              flags.MustString("chromium-proxy-server"),
		wsUrlReadTimeout:         flags.MustDuration("chromium-start-timeout"),

		allowList:         flags.MustRegexp("chromium-allow-list"),
		denyList:          flags.MustRegexp("chromium-deny-list"),
		clearCache:        flags.MustBool("chromium-clear-cache"),
		clearCookies:      flags.MustBool("chromium-clear-cookies"),
		disableJavaScript: flags.MustBool("chromium-disable-javascript"),
	}

	// Logger.
	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}
	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(mod)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}
	mod.logger = logger.Named("browser")

	// Process.
	mod.browser = newChromiumBrowser(mod.args)
	mod.supervisor = gotenberg.NewProcessSupervisor(mod.logger, mod.browser, flags.MustInt64("chromium-restart-after"), flags.MustInt64("chromium-max-queue-size"))

	// PDF Engine.
	provider, err := ctx.Module(new(gotenberg.PdfEngineProvider))
	if err != nil {
		return fmt.Errorf("get PDF engine provider: %w", err)
	}
	engine, err := provider.(gotenberg.PdfEngineProvider).PdfEngine()
	if err != nil {
		return fmt.Errorf("get PDF engine: %w", err)
	}
	mod.engine = engine

	return nil
}

// Validate validates the module properties.
func (mod *Chromium) Validate() error {
	_, err := os.Stat(mod.args.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("chromium binary path does not exist: %w", err)
	}

	return nil
}

// Start does nothing if auto-start is not enabled. Otherwise, it starts a
// browser instance.
func (mod *Chromium) Start() error {
	if !mod.autoStart {
		return nil
	}

	err := mod.supervisor.Launch()
	if err != nil {
		return fmt.Errorf("launch supervisor: %w", err)
	}

	return nil
}

// StartupMessage returns a custom startup message.
func (mod *Chromium) StartupMessage() string {
	if !mod.autoStart {
		return "Chromium ready to start"
	}

	return "Chromium automatically started"
}

// Stop stops the current browser instance.
func (mod *Chromium) Stop(ctx context.Context) error {
	// Block until the context is done so that other module may gracefully stop
	// before we do a shutdown.
	mod.logger.Debug("wait for the end of grace duration")

	<-ctx.Done()

	err := mod.supervisor.Shutdown()
	if err == nil {
		return nil
	}

	return fmt.Errorf("stop Chromium: %w", err)
}

// Metrics returns the metrics.
func (mod *Chromium) Metrics() ([]gotenberg.Metric, error) {
	return []gotenberg.Metric{
		{
			Name:        "chromium_requests_queue_size",
			Description: "Current number of Chromium conversion requests waiting to be treated.",
			Read: func() float64 {
				return float64(mod.supervisor.ReqQueueSize())
			},
		},
		{
			Name:        "chromium_restarts_count",
			Description: "Current number of Chromium restarts.",
			Read: func() float64 {
				return float64(mod.supervisor.RestartsCount())
			},
		},
	}, nil
}

// Checks adds a health check that verifies if Chromium is healthy.
func (mod *Chromium) Checks() ([]health.CheckerOption, error) {
	return []health.CheckerOption{
		health.WithCheck(health.Check{
			Name: "chromium",
			Check: func(_ context.Context) error {
				if mod.supervisor.Healthy() {
					return nil
				}

				return errors.New("Chromium is unhealthy")
			},
		}),
	}, nil
}

// Ready returns no error if the module is ready.
func (mod *Chromium) Ready() error {
	if !mod.autoStart {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), mod.args.wsUrlReadTimeout)
	defer cancel()

	ticker := time.NewTicker(time.Duration(100) * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return fmt.Errorf("context done while waiting for Chromium to be ready: %w", ctx.Err())
		case <-ticker.C:
			ok := mod.browser.Healthy(mod.logger)
			if ok {
				ticker.Stop()
				return nil
			}

			continue
		}
	}
}

// Chromium returns an [Api] for interacting with Chromium for converting HTML
// documents to PDF.
func (mod *Chromium) Chromium() (Api, error) {
	return mod, nil
}

// Routes returns the HTTP routes.
func (mod *Chromium) Routes() ([]api.Route, error) {
	if mod.disableRoutes {
		return nil, nil
	}

	return []api.Route{
		convertUrlRoute(mod, mod.engine),
		screenshotUrlRoute(mod),
		convertHtmlRoute(mod, mod.engine),
		screenshotHtmlRoute(mod),
		convertMarkdownRoute(mod, mod.engine),
		screenshotMarkdownRoute(mod),
	}, nil
}

// Pdf converts a URL to PDF.
func (mod *Chromium) Pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
	// Note: no error wrapping because it leaks on errors we want to display to
	// the end user.
	return mod.supervisor.Run(ctx, logger, func() error {
		return mod.browser.pdf(ctx, logger, url, outputPath, options)
	})
}

func (mod *Chromium) Screenshot(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
	// Note: no error wrapping because it leaks on errors we want to display to
	// the end user.
	return mod.supervisor.Run(ctx, logger, func() error {
		return mod.browser.screenshot(ctx, logger, url, outputPath, options)
	})
}

// Interface guards.
var (
	_ gotenberg.Module          = (*Chromium)(nil)
	_ gotenberg.Provisioner     = (*Chromium)(nil)
	_ gotenberg.Validator       = (*Chromium)(nil)
	_ gotenberg.App             = (*Chromium)(nil)
	_ gotenberg.MetricsProvider = (*Chromium)(nil)
	_ api.HealthChecker         = (*Chromium)(nil)
	_ api.Router                = (*Chromium)(nil)
	_ Api                       = (*Chromium)(nil)
	_ Provider                  = (*Chromium)(nil)
)
