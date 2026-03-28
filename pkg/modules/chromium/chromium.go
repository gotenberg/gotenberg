package chromium

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/chromedp/cdproto/network"
	"github.com/dlclark/regexp2"
	flag "github.com/spf13/pflag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Chromium))
}

var (
	// ErrInvalidEmulatedMediaType happens if the emulated media type is not
	// "screen" nor "print". Empty value is allowed, though.
	ErrInvalidEmulatedMediaType = errors.New("invalid emulated media type")

	// ErrInvalidEvaluationExpression happens if an evaluation expression
	// returns an exception or undefined.
	ErrInvalidEvaluationExpression = errors.New("invalid evaluation expression")

	// ErrInvalidSelectorQuery happens if a selector query returns an exception
	// or undefined.
	ErrInvalidSelectorQuery = errors.New("invalid selector query")

	// ErrRpccMessageTooLarge happens when the messages received by
	// ChromeDevTools are larger than 100 MB.
	ErrRpccMessageTooLarge = errors.New("rpcc message too large")

	// ErrInvalidHttpStatusCode happens when the status code from the main page
	// matches with one of the entries in [Options.FailOnHttpStatusCodes].
	ErrInvalidHttpStatusCode = errors.New("invalid HTTP status code")

	// ErrInvalidResourceHttpStatusCode happens when the status code from one
	// or more resources matches with one of the entries in
	// [Options.FailOnResourceHttpStatusCodes].
	ErrInvalidResourceHttpStatusCode = errors.New("invalid resource HTTP status code")

	// ErrConsoleExceptions happens when there are exceptions in the Chromium
	// console. It also happens only if the [Options.FailOnConsoleExceptions]
	// is set to true.
	ErrConsoleExceptions = errors.New("console exceptions")

	// ErrLoadingFailed happens when the main page failed to load.
	ErrLoadingFailed = errors.New("loading failed")

	// ErrResourceLoadingFailed happens when one or more resources failed to load.
	ErrResourceLoadingFailed = errors.New("resource loading failed")

	// PDF specific.

	// ErrOmitBackgroundWithoutPrintBackground happens if
	// PdfOptions.OmitBackground is set to true but not PdfOptions.PrintBackground.
	ErrOmitBackgroundWithoutPrintBackground = errors.New("omit background without print background")

	// ErrPrintingFailed happens if the printing failed for an unknown reason.
	ErrPrintingFailed = errors.New("printing failed")

	// ErrInvalidPrinterSettings happens if the PdfOptions have one or more
	// aberrant values.
	ErrInvalidPrinterSettings = errors.New("invalid printer settings")

	// ErrPageRangesSyntaxError happens if the PdfOptions page
	// range syntax is invalid.
	ErrPageRangesSyntaxError = errors.New("page ranges syntax error")

	// ErrPageRangesExceedsPageCount happens if the PdfOptions have an invalid
	// page range.
	ErrPageRangesExceedsPageCount = errors.New("page ranges exceeds page count")
)

// Chromium is a module that provides both an [Api] and routes for converting
// an HTML document to PDF.
type Chromium struct {
	autoStart      bool
	disableRoutes  bool
	maxConcurrency int64
	args           browserArguments

	logger     *slog.Logger
	browser    browser
	supervisor gotenberg.ProcessSupervisor
	engine     gotenberg.PdfEngine

	reqsCounter               metric.Int64Counter
	errsCounter               metric.Int64Counter
	conversionDurationCounter metric.Float64Histogram
	queueWaitDurationCounter  metric.Float64Histogram
	pdfOutputSizeCounter      metric.Int64Histogram
	imageOutputSizeCounter    metric.Int64Histogram
}

// Options are the common options for all conversions.
type Options struct {
	// SkipNetworkIdleEvent set if the conversion should wait for the
	// "networkIdle" event, drastically improving the conversion speed. It may
	// not be suitable for all HTML documents, as some may not be fully
	// rendered until this event is fired.
	SkipNetworkIdleEvent bool

	// SkipNetworkAlmostIdleEvent set if the conversion should wait for the
	// "networkAlmostIdle" event.
	SkipNetworkAlmostIdleEvent bool

	// FailOnHttpStatusCodes sets if the conversion should fail if the status
	// code from the main page matches with one of its entries.
	FailOnHttpStatusCodes []int64

	// FailOnResourceHttpStatusCodes sets if the conversion should fail if the
	// status code from at least one resource matches with one if its entries.
	FailOnResourceHttpStatusCodes []int64

	// IgnoreResourceHttpStatusDomains excludes resources whose hostname matches
	// one of these domains from the application of
	// [Options.FailOnResourceHttpStatusCodes].
	//
	// A match happens if the hostname equals the domain or is a subdomain of it
	// (e.g., "browser.sentry-cdn.com" matches "sentry-cdn.com").
	//
	// Values are normalized (trimmed, lowercased) and may be provided as:
	// - "example.com"
	// - "*.example.com" or ".example.com"
	// - "example.com:443" (port is ignored)
	// - "https://example.com/path" (scheme/path are ignored)
	IgnoreResourceHttpStatusDomains []string

	// FailOnResourceLoadingFailed sets if the conversion should fail like the
	// main page if Chromium fails to load at least one resource.
	FailOnResourceLoadingFailed bool

	// FailOnConsoleExceptions sets if the conversion should fail if there are
	// exceptions in the Chromium console.
	FailOnConsoleExceptions bool

	// WaitDelay is the duration to wait when loading an HTML document before
	// converting it.
	WaitDelay time.Duration

	// WaitWindowStatus is the window.status value to wait for before
	// converting an HTML document.
	WaitWindowStatus string

	// WaitForExpression is the custom JavaScript expression to wait before
	// converting an HTML document until it returns true
	WaitForExpression string

	// WaitForSelector is the element query to wait until visible before
	// converting an HTML document.
	WaitForSelector string

	// Cookies are the cookies to put in the Chromium cookies' jar.
	Cookies []Cookie

	// UserAgent overrides the default 'User-Agent' HTTP header.
	UserAgent string

	// ExtraHttpHeaders are extra HTTP headers to send by Chromium while
	// loading the HTML document.
	ExtraHttpHeaders []ExtraHttpHeader

	// EmulatedMediaType is the media type to emulate, either "screen" or
	// "print".
	EmulatedMediaType string

	// EmulatedMediaFeatures are the media features to emulate, e.g.,
	// [{"name": "prefers-color-scheme", "value": "dark"}].
	EmulatedMediaFeatures []EmulatedMediaFeature

	// OmitBackground hides the default white background and allows generating
	// PDFs with transparency.
	OmitBackground bool

	// AllowedFilePrefixes restricts file:// sub-resource access to only these
	// directory prefixes. Applied in listenForEventRequestPaused in addition
	// to the global allow/deny lists. Set internally by route handlers, not
	// via form data.
	AllowedFilePrefixes []string
}

// EmulatedMediaFeature gathers the available entries for emulating a media
// feature.
type EmulatedMediaFeature struct {
	// Name is the media feature name (e.g., "prefers-color-scheme",
	// "prefers-reduced-motion").
	// Required.
	Name string `json:"name"`

	// Value is the media feature value (e.g., "dark", "reduce").
	// Required.
	Value string `json:"value"`
}

// DefaultOptions returns the default values for Options.
func DefaultOptions() Options {
	return Options{
		SkipNetworkIdleEvent:            true,
		SkipNetworkAlmostIdleEvent:      true,
		FailOnHttpStatusCodes:           []int64{499, 599},
		FailOnResourceHttpStatusCodes:   nil,
		IgnoreResourceHttpStatusDomains: nil,
		FailOnResourceLoadingFailed:     false,
		FailOnConsoleExceptions:         false,
		WaitDelay:                       0,
		WaitWindowStatus:                "",
		WaitForExpression:               "",
		WaitForSelector:                 "",
		Cookies:                         nil,
		UserAgent:                       "",
		ExtraHttpHeaders:                nil,
		EmulatedMediaType:               "",
		EmulatedMediaFeatures:           nil,
		OmitBackground:                  false,
	}
}

// PdfOptions are the available options for converting an HTML document to PDF.
type PdfOptions struct {
	Options

	// Landscape sets the paper orientation.
	Landscape bool

	// PrintBackground prints the background graphics.
	PrintBackground bool

	// Scale is the scale of the page rendering.
	Scale float64

	// SinglePage defines whether to print the entire content in one single
	// page.
	SinglePage bool

	// PaperWidth is the paper width, in inches.
	PaperWidth float64

	// PaperHeight is the paper height, in inches.
	PaperHeight float64

	// MarginTop is the top margin, in inches.
	MarginTop float64

	// MarginBottom is the bottom margin, in inches.
	MarginBottom float64

	// MarginLeft is the left margin, in inches.
	MarginLeft float64

	// MarginRight is the right margin, in inches.
	MarginRight float64

	// Page ranges to print, e.g., '1-5, 8, 11-13'. Empty means all pages.
	PageRanges string

	// HeaderTemplate is the HTML template of the header. It should be a valid
	// HTML markup with the following classes used to inject printing values
	// into them:
	// - date: formatted print date
	// - title: document title
	// - url: document location
	// - pageNumber: current page number
	// - totalPages: total pages in the document
	// For example, <span class=title></span> would generate span containing
	// the title.
	HeaderTemplate string

	// FooterTemplate is the HTML template of the footer. It should use the
	// same format as the HeaderTemplate.
	FooterTemplate string

	// PreferCssPageSize defines whether to prefer page size as defined by CSS.
	// If false, the content will be scaled to fit the paper size.
	PreferCssPageSize bool

	// GenerateDocumentOutline defines whether the document outline should be
	// embedded into the PDF.
	GenerateDocumentOutline bool

	// GenerateTaggedPdf defines whether to generate tagged (accessible)
	// PDF.
	GenerateTaggedPdf bool
}

// DefaultPdfOptions returns the default values for PdfOptions.
func DefaultPdfOptions() PdfOptions {
	return PdfOptions{
		Options:                 DefaultOptions(),
		Landscape:               false,
		PrintBackground:         false,
		Scale:                   1.0,
		SinglePage:              false,
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
		GenerateDocumentOutline: false,
		GenerateTaggedPdf:       false,
	}
}

// ScreenshotOptions are the available options for capturing a screenshot from
// an HTML document.
type ScreenshotOptions struct {
	Options

	// Width is the device screen width in pixels.
	Width int

	// Height is the device screen height in pixels.
	Height int

	// Clip defines whether to clip the screenshot according to the device
	// dimensions.
	Clip bool

	// Format is the image compression format, either "png" or "jpeg" or
	// "webp".
	Format string

	// Quality is the compression quality from range [0..100] (jpeg only).
	Quality int

	// OptimizeForSpeed defines whether to optimize image encoding for speed,
	// not for resulting size.
	OptimizeForSpeed bool
}

// DefaultScreenshotOptions returns the default values for ScreenshotOptions.
func DefaultScreenshotOptions() ScreenshotOptions {
	return ScreenshotOptions{
		Options:          DefaultOptions(),
		Width:            800,
		Height:           600,
		Clip:             false,
		Format:           "png",
		Quality:          100,
		OptimizeForSpeed: false,
	}
}

// Cookie gathers the available entries for setting a cookie in the Chromium
// cookies' jar.
type Cookie struct {
	// Name is the cookie name.
	// Required.
	Name string `json:"name"`

	// Value is the cookie value.
	// Required.
	Value string `json:"value"`

	// Domain is the cookie domain.
	// Required.
	Domain string `json:"domain"`

	// Path is the cookie path.
	// Optional.
	Path string `json:"path,omitempty"`

	// Secure sets the cookie secure if true.
	// Optional.
	Secure bool `json:"secure,omitempty"`

	// HttpOnly sets the cookie as HTTP-only if true.
	// Optional.
	HttpOnly bool `json:"httpOnly,omitempty"`

	// SameSite is cookie 'Same-Site' status.
	// Optional.
	SameSite network.CookieSameSite `json:"sameSite,omitempty"`
}

// ExtraHttpHeader are extra HTTP headers to send by Chromium.
type ExtraHttpHeader struct {
	// Name is the header name.
	// Required.
	Name string

	// Value is the header value.
	// Required.
	Value string

	// Scope is the header scope. If nil, the header will be applied to ALL
	// requests from the page.
	// Optional.
	Scope *regexp2.Regexp
}

// Api helps to interact with Chromium for converting HTML documents to PDF.
type Api interface {
	Pdf(ctx context.Context, logger *slog.Logger, url, outputPath string, options PdfOptions) error
	Screenshot(ctx context.Context, logger *slog.Logger, url, outputPath string, options ScreenshotOptions) error
}

// Provider is a module interface that exposes a method for creating an [Api]
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
			fs.Int64("chromium-restart-after", 100, "Number of conversions after which Chromium will automatically restart. Set to 0 to disable this feature")
			fs.Int64("chromium-max-queue-size", 0, "Maximum request queue size for Chromium. Set to 0 to disable this feature")
			fs.Duration("chromium-idle-shutdown-timeout", 0, "Shutdown Chromium after being idle for the given duration. Set to 0 to disable this feature")
			fs.Int64("chromium-max-concurrency", 6, "Maximum number of concurrent conversions. Chromium supports up to 6")
			fs.Bool("chromium-auto-start", false, "Automatically launch Chromium upon initialization if set to true; otherwise, Chromium will start at the time of the first conversion")
			fs.Duration("chromium-start-timeout", time.Duration(20)*time.Second, "Maximum duration to wait for Chromium to start or restart")
			fs.Bool("chromium-allow-insecure-localhost", false, "Ignore TLS/SSL errors on localhost")
			fs.Bool("chromium-ignore-certificate-errors", false, "Ignore the certificate errors")
			fs.Bool("chromium-disable-web-security", false, "Don't enforce the same-origin policy")
			fs.Bool("chromium-allow-file-access-from-files", false, "Allow file:// URIs to read other file:// URIs")
			fs.String("chromium-host-resolver-rules", "", "Set custom mappings to the host resolver")
			fs.String("chromium-proxy-server", "", "Set the outbound proxy server; this switch only affects HTTP and HTTPS requests")
			fs.StringSlice("chromium-allow-list", []string{}, "Set the allowed URLs for Chromium using regular expressions - supports multiple values")
			fs.StringSlice("chromium-deny-list", []string{`^file:(?!//\/tmp/).*`}, "Set the denied URLs for Chromium using regular expressions - supports multiple values")
			fs.Bool("chromium-clear-cache", false, "Clear Chromium cache between each conversion")
			fs.Bool("chromium-clear-cookies", false, "Clear Chromium cookies between each conversion")
			fs.Bool("chromium-disable-javascript", false, "Disable JavaScript")
			fs.Bool("chromium-disable-routes", false, "Disable the routes")

			// Deprecated flags.
			fs.Bool("chromium-incognito", false, "Start Chromium with incognito mode")
			err := fs.MarkDeprecated("chromium-incognito", "this flag is ignored as it provides no benefits")
			if err != nil {
				panic(err)
			}

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
	mod.maxConcurrency = flags.MustInt64("chromium-max-concurrency")

	binPath, ok := os.LookupEnv("CHROMIUM_BIN_PATH")
	if !ok {
		return errors.New("CHROMIUM_BIN_PATH environment variable is not set")
	}

	hyphenDataDirPath, ok := os.LookupEnv("CHROMIUM_HYPHEN_DATA_DIR_PATH")
	if !ok {
		return errors.New("CHROMIUM_HYPHEN_DATA_DIR_PATH environment variable is not set")
	}

	mod.args = browserArguments{
		binPath:                  binPath,
		allowInsecureLocalhost:   flags.MustBool("chromium-allow-insecure-localhost"),
		ignoreCertificateErrors:  flags.MustBool("chromium-ignore-certificate-errors"),
		disableWebSecurity:       flags.MustBool("chromium-disable-web-security"),
		allowFileAccessFromFiles: flags.MustBool("chromium-allow-file-access-from-files"),
		hostResolverRules:        flags.MustString("chromium-host-resolver-rules"),
		proxyServer:              flags.MustString("chromium-proxy-server"),
		wsUrlReadTimeout:         flags.MustDuration("chromium-start-timeout"),
		hyphenDataDirPath:        hyphenDataDirPath,

		allowList:         flags.MustRegexpSlice("chromium-allow-list"),
		denyList:          flags.MustRegexpSlice("chromium-deny-list"),
		clearCache:        flags.MustBool("chromium-clear-cache"),
		clearCookies:      flags.MustBool("chromium-clear-cookies"),
		disableJavaScript: flags.MustBool("chromium-disable-javascript"),
	}

	// Logger.
	mod.logger = gotenberg.Logger(mod).With(slog.String("logger", "browser"))

	// Process.
	mod.browser = newChromiumBrowser(mod.args)
	mod.supervisor = gotenberg.NewProcessSupervisor(mod.logger, mod.browser, flags.MustInt64("chromium-restart-after"), flags.MustInt64("chromium-max-queue-size"), mod.maxConcurrency, flags.MustDuration("chromium-idle-shutdown-timeout"))

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

	// Metrics.
	meter := gotenberg.Meter()

	// Observable gauges.
	_, err = meter.Int64ObservableGauge(
		"chromium.requests.active",
		metric.WithDescription("Current number of active Chromium requests"),
		metric.WithUnit("{request}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(mod.supervisor.ActiveTasksCount())
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("create chromium.requests.active gauge: %w", err)
	}

	_, err = meter.Int64ObservableGauge(
		"chromium.requests.queue_size",
		metric.WithDescription("Current number of Chromium conversion requests waiting to be treated"),
		metric.WithUnit("{request}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(mod.supervisor.ReqQueueSize())
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("create chromium.requests.queue_size gauge: %w", err)
	}

	_, err = meter.Int64ObservableCounter(
		"chromium.process.restarts.total",
		metric.WithDescription("Current number of Chromium restarts"),
		metric.WithUnit("{restart}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(mod.supervisor.RestartsCount())
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("create chromium.process.restarts.total counter: %w", err)
	}

	// Counters.
	mod.reqsCounter, err = meter.Int64Counter(
		"chromium.requests.total",
		metric.WithDescription("Total number of Chromium conversion requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return fmt.Errorf("create chromium.requests.total counter: %w", err)
	}

	mod.errsCounter, err = meter.Int64Counter(
		"chromium.errors.total",
		metric.WithDescription("Total number of Chromium conversion errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return fmt.Errorf("create chromium.errors.total counter: %w", err)
	}

	// Histograms.
	durationBuckets := metric.WithExplicitBucketBoundaries(0.5, 1, 2, 5, 10, 30, 60)

	mod.conversionDurationCounter, err = meter.Float64Histogram(
		"chromium.conversion.duration",
		metric.WithDescription("Duration of Chromium conversions"),
		metric.WithUnit("s"),
		durationBuckets,
	)
	if err != nil {
		return fmt.Errorf("create chromium.conversion.duration histogram: %w", err)
	}

	mod.queueWaitDurationCounter, err = meter.Float64Histogram(
		"chromium.queue.wait.duration",
		metric.WithDescription("Duration of waiting in queue for Chromium conversions"),
		metric.WithUnit("s"),
		durationBuckets,
	)
	if err != nil {
		return fmt.Errorf("create chromium.queue.wait.duration histogram: %w", err)
	}

	mod.pdfOutputSizeCounter, err = meter.Int64Histogram(
		"chromium.pdf.output.size",
		metric.WithDescription("Size of PDF output from Chromium conversions"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return fmt.Errorf("create chromium.pdf.output.size histogram: %w", err)
	}

	mod.imageOutputSizeCounter, err = meter.Int64Histogram(
		"chromium.image.output.size",
		metric.WithDescription("Size of image output from Chromium screenshots"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return fmt.Errorf("create chromium.image.output.size histogram: %w", err)
	}

	return nil
}

// Validate validates the module properties.
func (mod *Chromium) Validate() error {
	if mod.maxConcurrency < 1 || mod.maxConcurrency > 6 {
		return fmt.Errorf("chromium-max-concurrency must be between 1 and 6, got %d", mod.maxConcurrency)
	}

	_, err := os.Stat(mod.args.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("chromium binary path does not exist: %w", err)
	}

	_, err = os.Stat(mod.args.hyphenDataDirPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("chromium hyphen-data directory path does not exist: %w", err)
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
	// Block until the context is done so that another module may gracefully
	// stop before we do a shutdown.
	mod.logger.DebugContext(ctx, "wait for the end of grace duration")

	<-ctx.Done()

	err := mod.supervisor.Shutdown()
	if err == nil {
		return nil
	}

	return fmt.Errorf("stop Chromium: %w", err)
}

// Debug returns additional debug data.
func (mod *Chromium) Debug() map[string]any {
	debug := make(map[string]any)

	cmd := exec.Command(mod.args.binPath, "--version") //nolint:gosec
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	output, err := cmd.Output()
	if err != nil {
		debug["version"] = err.Error()
		return debug
	}

	debug["version"] = strings.TrimSpace(string(output))
	return debug
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
func (mod *Chromium) Pdf(ctx context.Context, logger *slog.Logger, url, outputPath string, options PdfOptions) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "chromium.Pdf",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(mod.args.binPath)),
	)
	defer span.End()

	start := time.Now()
	var conversionStart time.Time

	err := mod.supervisor.Run(ctx, logger, func() error {
		conversionStart = time.Now()
		return mod.browser.pdf(ctx, logger, url, outputPath, options)
	})

	end := time.Now()

	status := "success"
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			status = "timeout"
		} else {
			status = "error"
		}

		reason := "unknown"
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			reason = "timeout"
		case errors.Is(err, context.Canceled):
			reason = "context_cancelled"
		case errors.Is(err, ErrInvalidHttpStatusCode) || errors.Is(err, ErrInvalidResourceHttpStatusCode) || errors.Is(err, ErrLoadingFailed) || errors.Is(err, ErrResourceLoadingFailed) || errors.Is(err, ErrInvalidEvaluationExpression) || errors.Is(err, ErrInvalidSelectorQuery):
			reason = "invalid_input"
		case errors.Is(err, gotenberg.ErrMaximumQueueSizeExceeded) || errors.Is(err, gotenberg.ErrProcessAlreadyRestarting):
			reason = "chromium_unavailable"
		}

		mod.errsCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("reason", reason),
		))
	}

	if !conversionStart.IsZero() {
		waitDuration := conversionStart.Sub(start).Seconds()
		conversionDuration := end.Sub(conversionStart).Seconds()

		mod.queueWaitDurationCounter.Record(ctx, waitDuration, metric.WithAttributes(
			attribute.String("status", status),
		))
		mod.conversionDurationCounter.Record(ctx, conversionDuration, metric.WithAttributes(
			attribute.String("status", status),
		))
	} else {
		waitDuration := end.Sub(start).Seconds()
		mod.queueWaitDurationCounter.Record(ctx, waitDuration, metric.WithAttributes(
			attribute.String("status", status),
		))
	}

	mod.reqsCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))

	if err == nil {
		if fileInfo, statErr := os.Stat(outputPath); statErr == nil {
			mod.pdfOutputSizeCounter.Record(ctx, fileInfo.Size())
		}

		span.SetStatus(codes.Ok, "")
		return nil
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func (mod *Chromium) Screenshot(ctx context.Context, logger *slog.Logger, url, outputPath string, options ScreenshotOptions) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "chromium.Screenshot",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.ServerAddress(mod.args.binPath)),
	)
	defer span.End()

	start := time.Now()
	var conversionStart time.Time

	err := mod.supervisor.Run(ctx, logger, func() error {
		conversionStart = time.Now()
		return mod.browser.screenshot(ctx, logger, url, outputPath, options)
	})

	end := time.Now()

	status := "success"
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			status = "timeout"
		} else {
			status = "error"
		}

		reason := "unknown"
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			reason = "timeout"
		case errors.Is(err, context.Canceled):
			reason = "context_cancelled"
		case errors.Is(err, ErrInvalidHttpStatusCode) || errors.Is(err, ErrInvalidResourceHttpStatusCode) || errors.Is(err, ErrLoadingFailed) || errors.Is(err, ErrResourceLoadingFailed) || errors.Is(err, ErrInvalidEvaluationExpression) || errors.Is(err, ErrInvalidSelectorQuery):
			reason = "invalid_input"
		case errors.Is(err, gotenberg.ErrMaximumQueueSizeExceeded):
			reason = "chromium_maximum_queue_size_exceeded"
		case errors.Is(err, gotenberg.ErrProcessAlreadyRestarting):
			reason = "chromium_unavailable"
		}

		mod.errsCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("reason", reason),
		))
	}

	if !conversionStart.IsZero() {
		waitDuration := conversionStart.Sub(start).Seconds()
		conversionDuration := end.Sub(conversionStart).Seconds()

		mod.queueWaitDurationCounter.Record(ctx, waitDuration, metric.WithAttributes(
			attribute.String("status", status),
		))
		mod.conversionDurationCounter.Record(ctx, conversionDuration, metric.WithAttributes(
			attribute.String("status", status),
		))
	} else {
		waitDuration := end.Sub(start).Seconds()
		mod.queueWaitDurationCounter.Record(ctx, waitDuration, metric.WithAttributes(
			attribute.String("status", status),
		))
	}

	mod.reqsCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))

	if err == nil {
		if fileInfo, statErr := os.Stat(outputPath); statErr == nil {
			mod.imageOutputSizeCounter.Record(ctx, fileInfo.Size())
		}

		span.SetStatus(codes.Ok, "")
		return nil
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// Interface guards.
var (
	_ gotenberg.Module          = (*Chromium)(nil)
	_ gotenberg.Provisioner     = (*Chromium)(nil)
	_ gotenberg.Validator       = (*Chromium)(nil)
	_ gotenberg.App             = (*Chromium)(nil)
	_ gotenberg.Debuggable      = (*Chromium)(nil)
	_ gotenberg.MetricsProvider = (*Chromium)(nil)
	_ api.HealthChecker         = (*Chromium)(nil)
	_ api.Router                = (*Chromium)(nil)
	_ Api                       = (*Chromium)(nil)
	_ Provider                  = (*Chromium)(nil)
)
