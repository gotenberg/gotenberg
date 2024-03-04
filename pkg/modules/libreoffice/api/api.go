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

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func init() {
	gotenberg.MustRegisterModule(new(Api))
}

var (
	// ErrInvalidPdfFormats happens if the PDF formats option cannot be handled
	// by LibreOffice.
	ErrInvalidPdfFormats = errors.New("invalid PDF formats")

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
	// Optional.
	PageRanges string

	// PdfFormats allows to convert the resulting PDF to PDF/A-1b, PDF/A-2b,
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
			fs.Int64("libreoffice-restart-after", 10, "Number of conversions after which LibreOffice will automatically restart. Set to 0 to disable this feature")
			fs.Int64("libreoffice-max-queue-size", 0, "Maximum request queue size for LibreOffice. Set to 0 to disable this feature")
			fs.Bool("libreoffice-auto-start", false, "Automatically launch LibreOffice upon initialization if set to true; otherwise, LibreOffice will start at the time of the first conversion")
			fs.Duration("libreoffice-start-timeout", time.Duration(20)*time.Second, "Maximum duration to wait for LibreOffice to start or restart")

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
		startTimeout: flags.MustDuration("libreoffice-start-timeout"),
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
	a.supervisor = gotenberg.NewProcessSupervisor(a.logger, a.libreOffice, flags.MustInt64("libreoffice-restart-after"), flags.MustInt64("libreoffice-max-queue-size"))

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

// Ready returns no error if the module is ready.
func (a *Api) Ready() error {
	if !a.autoStart {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.args.startTimeout)
	defer cancel()

	ticker := time.NewTicker(time.Duration(100) * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return fmt.Errorf("context done while waiting for LibreOffice to be ready: %w", ctx.Err())
		case <-ticker.C:
			ok := a.libreOffice.Healthy(a.logger)
			if ok {
				ticker.Stop()
				return nil
			}

			continue
		}
	}
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
		".bmp",
		".csv",
		".dbf",
		".dif",
		".doc",
		".docx",
		".dotx",
		".emf",
		".eps",
		".epub",
		".fodg",
		".fodp",
		".fods",
		".fodt",
		".gif",
		".html",
		".jpeg",
		".jpg",
		".key",
		".ltx",
		".met",
		".odd",
		".odg",
		".odp",
		".ods",
		".odt",
		".otg",
		".ots",
		".ott",
		".pages",
		".pbm",
		".pct",
		".pdb",
		".pdf",
		".pgm",
		".png",
		".pot",
		".potm",
		".ppm",
		".pps",
		".ppt",
		".pptx",
		".psw",
		".pwp",
		".pxl",
		".ras",
		".rtf",
		".sda",
		".sdc",
		".sdd",
		".sdw",
		".slk",
		".stc",
		".std",
		".sti",
		".stw",
		".svg",
		".svm",
		".swf",
		".sxc",
		".sxd",
		".sxi",
		".sxw",
		".sxw",
		".tif",
		".tiff",
		".txt",
		".uop",
		".uos",
		".uot",
		".vor",
		".wmf",
		".wps",
		".xhtml",
		".xls",
		".xlsx",
		".xlt",
		".xltx",
		".xml",
		".xpm",
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
