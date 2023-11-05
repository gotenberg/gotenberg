package api

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
	gotenberg.MustRegisterModule(new(Api))
}

var (
	// ErrInvalidPdfFormat happens if the PDF format option cannot be handled
	// by LibreOffice.
	ErrInvalidPdfFormat = errors.New("invalid PDF format")

	// ErrMalformedPageRanges happens if the page ranges option cannot be
	// interpreted by LibreOffice.
	ErrMalformedPageRanges = errors.New("page ranges are malformed")
)

// Api is a module which provides a [Uno] to interact with LibreOffice.
type Api struct {
	autoStart bool
	args      libreOfficeArguments

	logger      *zap.Logger
	libreOffice libreOffice
	supervisor  gotenberg.ProcessSupervisor
}

// Options gathers available options when converting a document to PDF.
type Options struct {
	// Landscape allows to change the orientation of the resulting PDF.
	// Optional.
	Landscape bool

	// PageRanges allows to select the pages to convert.
	// TODO: should prefer a method form PdfEngine.
	// Optional.
	PageRanges string

	// PdfFormats allows to convert the resulting PDF to PDF/A-1a, PDF/A-2b,
	// PDF/A-3b and PDF/UA.
	// Optional.
	PdfFormats gotenberg.PdfFormats
}

// Uno is an abstraction on top of the Universal Network Objects API.
type Uno interface {
	Pdf(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error
	Extensions() []string
}

// Provider is a module interface which exposes a method for creating a
// [Uno] for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(libreofficeapi.Provider))
//		libreOffice, _      := provider.(api.Provider).LibreOffice()
//	}
type Provider interface {
	LibreOffice() (Uno, error)
}

// Descriptor returns a [Api]'s module descriptor.
func (a *Api) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "libreoffice-api",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("api", flag.ExitOnError)

			// Deprecated flags.
			fs.Duration("uno-listener-start-timeout", time.Duration(10)*time.Second, "Time limit for restarting the LibreOffice")
			fs.Int("uno-listener-restart-threshold", 10, "Conversions limit after which the LibreOffice listener is restarted - 0 means no restart")
			fs.Bool("unoconv-disable-listener", false, "Do not start a long-running listener - save resources in detriment of unitary performance")

			var err error
			err = multierr.Append(err, fs.MarkDeprecated("uno-listener-start-timeout", "use the libreOffice-start-timeout property instead"))
			err = multierr.Append(err, fs.MarkDeprecated("uno-listener-restart-threshold", "use the libreOffice-restart-after property instead"))
			err = multierr.Append(err, fs.MarkDeprecated("unoconv-disable-listener", "use the libreOffice-auto-start property instead"))

			if err != nil {
				panic(fmt.Errorf("create deprecated flags for the LibreOffice module: %v", err))
			}

			fs.Int64("libreoffice-restart-after", 10, "Number of conversions after which LibreOffice will automatically restart. Set to 0 to disable this feature")
			fs.Bool("libreoffice-auto-start", false, "Automatically launch LibreOffice upon initialization if set to true; otherwise, LibreOffice will start at the time of the first conversion")
			fs.Duration("libreoffice-start-timeout", time.Duration(10)*time.Second, "Maximum duration to wait for LibreOffice to start or restart")

			return fs
		}(),
		New: func() gotenberg.Module { return new(Api) },
	}
}

// Provision sets the module properties.
func (a *Api) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	a.autoStart = flags.MustBool("libreoffice-auto-start")

	libreOfficeBinPath, ok := os.LookupEnv("LIBREOFFICE_BIN_PATH")
	if !ok {
		return errors.New("LIBREOFFICE_BIN_PATH environment variable is not set")
	}

	unoBinPath, ok := os.LookupEnv("UNOCONVERTER_BIN_PATH")
	if !ok {
		return errors.New("UNOCONVERTER_BIN_PATH environment variable is not set")
	}

	a.args = libreOfficeArguments{
		binPath:      libreOfficeBinPath,
		unoBinPath:   unoBinPath,
		startTimeout: flags.MustDeprecatedDuration("uno-listener-start-timeout", "libreoffice-start-timeout"),
	}

	// Logger.
	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}
	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(a)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}
	a.logger = logger.Named("libreoffice")

	// Process.
	a.libreOffice = newLibreOfficeProcess(a.args)
	a.supervisor = gotenberg.NewProcessSupervisor(a.logger, a.libreOffice, flags.MustDeprecatedInt64("uno-listener-restart-threshold", "libreoffice-restart-after"))

	return nil
}

// Validate validates the module properties.
func (a *Api) Validate() error {
	var err error

	_, statErr := os.Stat(a.args.binPath)
	if os.IsNotExist(statErr) {
		err = multierr.Append(err, fmt.Errorf("LibreOffice binary path does not exist: %w", statErr))
	}

	_, statErr = os.Stat(a.args.unoBinPath)
	if os.IsNotExist(statErr) {
		err = multierr.Append(err, fmt.Errorf("unoconverter binary path does not exist: %w", statErr))
	}

	return err
}

// Start does nothing if auto-start is not enabled. Otherwise, it starts a
// LibreOffice instance.
func (a *Api) Start() error {
	if !a.autoStart {
		return nil
	}

	err := a.supervisor.Launch()
	if err != nil {
		return fmt.Errorf("launch supervisor: %w", err)
	}

	return nil
}

// StartupMessage returns a custom startup message.
func (a *Api) StartupMessage() string {
	if !a.autoStart {
		return "LibreOffice ready to start"
	}

	return "LibreOffice automatically started"
}

// Stop stops the current browser instance.
func (a *Api) Stop(ctx context.Context) error {
	// Block until the context is done so that other module may gracefully stop
	// before we do a shutdown.
	a.logger.Debug("wait for the end of grace duration")

	<-ctx.Done()

	err := a.supervisor.Shutdown()
	if err == nil {
		return nil
	}

	return fmt.Errorf("stop LibreOffice: %w", err)
}

// Metrics returns the metrics.
func (a *Api) Metrics() ([]gotenberg.Metric, error) {
	return []gotenberg.Metric{
		// TODO: remove deprecated.
		{
			Name:        "unoconv_active_instances_count",
			Description: "Current number of active unoconv instances - deprecated.",
			Read: func() float64 {
				return 1
			},
		},
		// TODO: remove deprecated.
		{
			Name:        "libreoffice_listener_active_instances_count",
			Description: "Current number of active LibreOffice listener instances - deprecated.",
			Read: func() float64 {
				return 1
			},
		},
		// TODO: remove deprecated.
		{
			Name:        "unoconv_listener_active_instances_count",
			Description: "Current number of active unoconv listener instances- deprecated.",
			Read: func() float64 {
				return 1
			},
		},
		// TODO: remove deprecated.
		{
			Name:        "libreoffice_listener_queue_length",
			Description: "Current number of processes in the LibreOffice listener queue - deprecated, prefer libreoffice_requests_queue_size.",
			Read: func() float64 {
				return float64(a.supervisor.ReqQueueSize())
			},
		},
		// TODO: remove deprecated.
		{
			Name:        "unoconv_listener_queue_length",
			Description: "Current number of processes in the queue - deprecated, prefer libreoffice_requests_queue_size.",
			Read: func() float64 {
				return float64(a.supervisor.ReqQueueSize())
			},
		},
		{
			Name:        "libreoffice_requests_queue_size",
			Description: "Current number of LibreOffice conversion requests waiting to be treated.",
			Read: func() float64 {
				return float64(a.supervisor.ReqQueueSize())
			},
		},
		{
			Name:        "libreoffice_restarts_count",
			Description: "Current number of LibreOffice restarts.",
			Read: func() float64 {
				return float64(a.supervisor.RestartsCount())
			},
		},
	}, nil
}

// Checks adds a health check that verifies if LibreOffice is healthy.
func (a *Api) Checks() ([]health.CheckerOption, error) {
	return []health.CheckerOption{
		health.WithCheck(health.Check{
			Name: "libreoffice",
			Check: func(_ context.Context) error {
				if a.supervisor.Healthy() {
					return nil
				}

				return errors.New("LibreOffice is unhealthy")
			},
		}),
	}, nil
}

// LibreOffice returns a [Uno] for interacting with LibreOffice.
func (a *Api) LibreOffice() (Uno, error) {
	return a, nil
}

// Pdf converts a document to PDF.
func (a *Api) Pdf(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
	return a.supervisor.Run(ctx, logger, func() error {
		return a.libreOffice.pdf(ctx, logger, inputPath, outputPath, options)
	})
}

// Extensions returns the file extensions available for conversions.
// FIXME: don't care, take all on the route level?
func (a *Api) Extensions() []string {
	return []string{
		".bib",
		".doc",
		".xml",
		".docx",
		".fodt",
		".html",
		".ltx",
		".txt",
		".odt",
		".ott",
		".pdb",
		".pdf",
		".psw",
		".rtf",
		".sdw",
		".stw",
		".sxw",
		".uot",
		".vor",
		".wps",
		".epub",
		".png",
		".bmp",
		".emf",
		".eps",
		".fodg",
		".gif",
		".jpg",
		".jpeg",
		".met",
		".odd",
		".otg",
		".pbm",
		".pct",
		".pgm",
		".ppm",
		".ras",
		".std",
		".svg",
		".svm",
		".swf",
		".sxd",
		".sxw",
		".tif",
		".tiff",
		".xhtml",
		".xpm",
		".odp",
		".fodp",
		".potm",
		".pot",
		".pptx",
		".pps",
		".ppt",
		".pwp",
		".sda",
		".sdd",
		".sti",
		".sxi",
		".uop",
		".wmf",
		".csv",
		".dbf",
		".dif",
		".fods",
		".ods",
		".ots",
		".pxl",
		".sdc",
		".slk",
		".stc",
		".sxc",
		".uos",
		".xls",
		".xlt",
		".xlsx",
		".odg",
		".dotx",
		".xltx",
	}
}

// Interface guards.
var (
	_ gotenberg.Module          = (*Api)(nil)
	_ gotenberg.Provisioner     = (*Api)(nil)
	_ gotenberg.Validator       = (*Api)(nil)
	_ gotenberg.App             = (*Api)(nil)
	_ gotenberg.MetricsProvider = (*Api)(nil)
	_ api.HealthChecker         = (*Api)(nil)
	_ Uno                       = (*Api)(nil)
	_ Provider                  = (*Api)(nil)
)
