package chromium

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alexliesenfeld/health"
	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Chromium))
}

var (
	// ErrUrlNotAuthorized happens if a URL is not acceptable according to the
	// allowed/denied lists.
	ErrUrlNotAuthorized = errors.New("URL not authorized")

	// ErrOmitBackgroundWithoutPrintBackground happens if
	// Options.OmitBackground is set to true but not Options.PrintBackground.
	ErrOmitBackgroundWithoutPrintBackground = errors.New("omit background without print background")

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
	// console. It also happens only if the [Options.FailOnConsoleExceptions]
	// is set to true.
	ErrConsoleExceptions = errors.New("console exceptions")
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

// Options are the available expectedOptions for converting HTML document to PDF.
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

	// ExtraHttpHeaders are the HTTP headers to send by Chromium while loading
	// the HTML document.
	// Optional.
	ExtraHttpHeaders map[string]string

	// EmulatedMediaType is the media type to emulate, either "screen" or
	// "print".
	// Optional.
	EmulatedMediaType string

	// Landscape sets the paper orientation.
	// Optional.
	Landscape bool

	// PrintBackground prints the background graphics.
	// Optional.
	PrintBackground bool

	// OmitBackground hides default white background and allows generating PDFs
	// with transparency.
	// Optional.
	OmitBackground bool

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

	// PreferCssPageSize defines whether to prefer page size as defined by CSS.
	// If false, the content will be scaled to fit the paper size.
	// Optional.
	PreferCssPageSize bool
}

// DefaultOptions returns the default values for Options.
func DefaultOptions() Options {
	return Options{
		FailOnConsoleExceptions: false,
		WaitDelay:               0,
		WaitWindowStatus:        "",
		WaitForExpression:       "",
		ExtraHttpHeaders:        nil,
		EmulatedMediaType:       "",
		Landscape:               false,
		PrintBackground:         false,
		OmitBackground:          false,
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
		PreferCssPageSize:       false,
	}
}

// Api helps to interact with Chromium for converting HTML documents to PDF.
type Api interface {
	Pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error
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
			// Deprecated flags.
			fs.String("chromium-user-agent", "", "Override the default User-Agent header")
			fs.Int("chromium-failed-starts-threshold", 5, "Set the number of consecutive failed starts after which the module is considered unhealthy - 0 means ignore")

			var err error
			err = multierr.Append(err, fs.MarkDeprecated("chromium-user-agent", "use the extraHttpHeaders form field instead"))
			err = multierr.Append(err, fs.MarkDeprecated("chromium-failed-starts-threshold", "use the chromium-restart-after property instead"))

			if err != nil {
				panic(fmt.Errorf("create deprecated flags for the Chromium module: %v", err))
			}

			fs.Int64("chromium-restart-after", 0, "Number of conversions after which Chromium will automatically restart. Set to 0 to disable this feature")
			fs.Bool("chromium-auto-start", false, "Automatically launch Chromium upon initialization if set to true; otherwise, Chromium will start at the time of the first conversion")
			fs.Duration("chromium-start-timeout", time.Duration(10)*time.Second, "Maximum duration to wait for Chromium to start or restart")
			fs.Bool("chromium-incognito", false, "Start Chromium with incognito mode")
			fs.Bool("chromium-allow-insecure-localhost", false, "Ignore TLS/SSL errors on localhost")
			fs.Bool("chromium-ignore-certificate-errors", false, "Ignore the certificate errors")
			fs.Bool("chromium-disable-web-security", false, "Don't enforce the same-origin policy")
			fs.Bool("chromium-allow-file-access-from-files", false, "Allow file:// URIs to read other file:// URIs")
			fs.String("chromium-host-resolver-rules", "", "Set custom mappings to the host resolver")
			fs.String("chromium-proxy-server", "", "Set the outbound proxy server; this switch only affects HTTP and HTTPS requests")
			fs.String("chromium-allow-list", "", "Set the allowed URLs for Chromium using a regular expression")
			fs.String("chromium-deny-list", "^file:///[^tmp].*", "Set the denied URLs for Chromium using a regular expression")
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
	mod.supervisor = gotenberg.NewProcessSupervisor(mod.logger, mod.browser, flags.MustInt64("chromium-restart-after"))

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
			// TODO: remove deprecated.
			Name:        "chromium_active_instances_count",
			Description: "Current number of active Chromium instances - deprecated.",
			Read: func() float64 {
				return 1
			},
		},
		{
			// TODO: remove deprecated.
			Name:        "chromium_failed_starts_count",
			Description: "Current number of Chromium consecutive starting failures - deprecated.",
			Read: func() float64 {
				return 0
			},
		},
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
		convertHtmlRoute(mod, mod.engine),
		convertMarkdownRoute(mod, mod.engine),
	}, nil
}

// Pdf converts a URL to PDF.
func (mod *Chromium) Pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
	// FIXME: no error wrapping because it leaks on console exceptions output.
	return mod.supervisor.Run(ctx, logger, func() error {
		return mod.browser.pdf(ctx, logger, url, outputPath, options)
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
