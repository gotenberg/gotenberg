package uno

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(UNO{})
}

var (
	// ErrInvalidPDFformat happens if the PDF format option cannot be handled
	// by LibreOffice.
	ErrInvalidPDFformat = errors.New("invalid PDF format")

	// ErrMalformedPageRanges happens if the page ranges option cannot be
	// interpreted by LibreOffice.
	ErrMalformedPageRanges = errors.New("page ranges are malformed")
)

// UNO is a module which provides an API to interact with LibreOffice.
type UNO struct {
	unoconvBinPath              string
	libreOfficeBinPath          string
	libreOfficeStartTimeout     time.Duration
	libreOfficeRestartThreshold int

	listener listener
	logger   *zap.Logger
}

// Options gathers available options when converting a document to another format.
type Options struct {
	// Landscape allows to change the orientation of the resulting PDF.
	// Optional.
	Landscape bool

	// PageRanges allows to select the pages to convert.
	// TODO: should prefer a method form PDFEngine.
	// Optional.
	PageRanges string

	// PDFformat allows to convert the resulting PDF to PDF/A-1a, PDF/A-2b, or
	// PDF/A-3b.
	// Optional.
	PDFformat string

	// Optionally generate HTML output, particularly useful for rendering
	// spreadsheets with many columns.
	HTMLformat bool
}

// API is an abstraction on top of uno.
type API interface {
	Convert(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error
	Extensions() []string
}

// Provider is a module interface which exposes a method for creating an API
// for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(uno.Provider))
//		unoAPI, _      := provider.(uno.Provider).UNO()
//	}
type Provider interface {
	UNO() (API, error)
}

// Descriptor returns a UNO's module descriptor.
func (UNO) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "uno",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("uno", flag.ExitOnError)
			fs.Duration("uno-listener-start-timeout", time.Duration(10)*time.Second, "Time limit for restarting the LibreOffice listener")
			fs.Int("uno-listener-restart-threshold", 10, "Conversions limit after which the LibreOffice listener is restarted - 0 means no long-running LibreOffice listener")
			fs.Bool("unoconv-disable-listener", false, "Do not start a long-running listener - save resources in detriment of unitary performance")

			err := fs.MarkDeprecated("unoconv-disable-listener", "use uno-listener-restart-threshold with 0 instead")
			if err != nil {
				panic(fmt.Errorf("create deprecated flags for the uno module: %v", err))
			}

			return fs
		}(),
		New: func() gotenberg.Module { return new(UNO) },
	}
}

// Provision sets the module properties. It returns an error if the environment
// variables UNOCONV_BIN_PATH and LIBREOFFICE_BIN_PATH are not set.
func (mod *UNO) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.libreOfficeStartTimeout = flags.MustDuration("uno-listener-start-timeout")
	mod.libreOfficeRestartThreshold = flags.MustInt("uno-listener-restart-threshold")

	disableListener := flags.MustBool("unoconv-disable-listener")
	if disableListener {
		mod.libreOfficeRestartThreshold = 0
	}

	unoconvBinPath, ok := os.LookupEnv("UNOCONV_BIN_PATH")
	if !ok {
		return errors.New("UNOCONV_BIN_PATH environment variable is not set")
	}

	mod.unoconvBinPath = unoconvBinPath

	libreOfficeBinPath, ok := os.LookupEnv("LIBREOFFICE_BIN_PATH")
	if !ok {
		return errors.New("LIBREOFFICE_BIN_PATH environment variable is not set")
	}

	mod.libreOfficeBinPath = libreOfficeBinPath

	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}

	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(mod)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}

	mod.logger = logger

	mod.listener = newLibreOfficeListener(
		mod.logger,
		mod.libreOfficeBinPath,
		mod.libreOfficeStartTimeout,
		mod.libreOfficeRestartThreshold,
	)

	return nil
}

// Validate validates the module properties.
func (mod UNO) Validate() error {
	var err error

	_, statErr := os.Stat(mod.unoconvBinPath)
	if os.IsNotExist(statErr) {
		err = multierr.Append(err, fmt.Errorf("unoconv binary path does not exist: %w", statErr))
	}

	_, statErr = os.Stat(mod.libreOfficeBinPath)
	if os.IsNotExist(statErr) {
		err = multierr.Append(err, fmt.Errorf("LibreOffice binary path does not exist: %w", statErr))
	}

	return err
}

// Start does nothing: it is here to validate the contract from the
// gotenberg.App interface. The long-running LibreOffice Listener will be
// started on the first call to PDF.
func (mod UNO) Start() error {
	return nil
}

// StartupMessage returns a custom startup message.
func (mod UNO) StartupMessage() string {
	if mod.libreOfficeRestartThreshold == 0 {
		return "long-running LibreOffice listener disabled"
	}

	return "long-running LibreOffice listener ready to start"
}

// Stop stops the long-running LibreOffice Listener if it exists.
func (mod UNO) Stop(ctx context.Context) error {
	if mod.libreOfficeRestartThreshold == 0 {
		return nil
	}

	// Block until the context is done so that other module may gracefully stop
	// before we do a shutdown cleanup.
	mod.logger.Debug("wait for the end of grace duration")

	<-ctx.Done()

	err := mod.listener.stop(mod.logger)
	if err == nil {
		return nil
	}

	return fmt.Errorf("stop long-running LibreOffice listener")
}

// Metrics returns the metrics.
func (mod UNO) Metrics() ([]gotenberg.Metric, error) {
	return []gotenberg.Metric{
		{
			Name:        "unoconv_active_instances_count",
			Description: "Current number of active unoconv instances.",
			Read: func() float64 {
				activeInstancesCountMu.RLock()
				defer activeInstancesCountMu.RUnlock()

				return activeInstancesCount
			},
		},
		{
			Name:        "libreoffice_listener_active_instances_count",
			Description: "Current number of active LibreOffice listener instances.",
			Read: func() float64 {
				if mod.libreOfficeRestartThreshold == 0 {
					listenerActiveInstancesCountMu.RLock()
					defer listenerActiveInstancesCountMu.RUnlock()

					return listenerActiveInstancesCount
				}

				if mod.listener.healthy() {
					return 1
				}

				return 0
			},
		},
		{
			Name:        "unoconv_listener_active_instances_count",
			Description: "Current number of active unoconv listener instances - deprecated, prefer libreoffice_listener_active_instances_count.",
			Read: func() float64 {
				if mod.libreOfficeRestartThreshold == 0 {
					listenerActiveInstancesCountMu.RLock()
					defer listenerActiveInstancesCountMu.RUnlock()

					return listenerActiveInstancesCount
				}

				if mod.listener.healthy() {
					return 1
				}

				return 0
			},
		},
		{
			Name:        "libreoffice_listener_queue_length",
			Description: "Current number of processes in the LibreOffice listener queue.",
			Read: func() float64 {
				return float64(mod.listener.queue())
			},
		},
		{
			Name:        "unoconv_listener_queue_length",
			Description: "Current number of processes in the queue - deprecated, prefer libreoffice_listener_queue_length.",
			Read: func() float64 {
				return float64(mod.listener.queue())
			},
		},
	}, nil
}

// Checks adds a health check that verifies the health of the long-running
// LibreOffice listener.
func (mod UNO) Checks() ([]health.CheckerOption, error) {
	if mod.libreOfficeRestartThreshold == 0 {
		return nil, nil
	}

	return []health.CheckerOption{
		health.WithCheck(health.Check{
			Name: "uno",
			Check: func(_ context.Context) error {
				if mod.listener.healthy() {
					return nil
				}

				return errors.New("long-running LibreOffice listener unhealthy")
			},
		}),
	}, nil
}

// Convert a document to another format.
//
// If there is no long-running LibreOffice listener, it creates a dedicated
// LibreOffice instance for the conversion. Substantial calls to this method
// may increase CPU and memory usage drastically
//
// If there is a long-running LibreOffice listener, the conversion performance
// improves substantially. However, it cannot perform parallel operations.
func (mod UNO) Convert(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
	args := []string{
		"--no-launch",
		"--format",
	}
	if options.HTMLformat {
		args = append(args, "html")
	} else {
		args = append(args, "pdf")
	}

	switch mod.libreOfficeRestartThreshold {
	case 0:
		listener := newLibreOfficeListener(logger, mod.libreOfficeBinPath, mod.libreOfficeStartTimeout, 0)

		err := listener.start(logger)
		if err != nil {
			return fmt.Errorf("start LibreOffice listener: %w", err)
		}

		defer func() {
			err := listener.stop(logger)
			if err != nil {
				logger.Error(fmt.Sprintf("stop LibreOffice listener: %v", err))
			}
		}()

		args = append(args, "--port", fmt.Sprintf("%d", listener.port()))
	default:
		err := mod.listener.lock(ctx, logger)
		if err != nil {
			return fmt.Errorf("lock long-running LibreOffice listener: %w", err)
		}

		defer func() {
			go func() {
				err := mod.listener.unlock(logger)
				if err != nil {
					mod.logger.Error(fmt.Sprintf("unlock long-running LibreOffice listener: %v", err))

					return
				}
			}()
		}()

		// If the LibreOffice listener is restarting while acquiring the lock,
		// the port will change. It's therefore important to add the port args
		// after we acquire the lock.
		args = append(args, "--port", fmt.Sprintf("%d", mod.listener.port()))
	}

	checkedEntry := logger.Check(zap.DebugLevel, "check for debug level before setting high verbosity")
	if checkedEntry != nil {
		args = append(args, "-vvv")
	}

	// PDF-only options.
	if !options.HTMLformat {
		if options.Landscape {
			args = append(args, "--printer", "PaperOrientation=landscape")
		}

		if options.PageRanges != "" {
			args = append(args, "--export", fmt.Sprintf("PageRange=%s", options.PageRanges))
		}

		switch options.PDFformat {
		case "":
		case gotenberg.FormatPDFA1a:
			args = append(args, "--export", "SelectPdfVersion=1")
		case gotenberg.FormatPDFA2b:
			args = append(args, "--export", "SelectPdfVersion=2")
		case gotenberg.FormatPDFA3b:
			args = append(args, "--export", "SelectPdfVersion=3")
		default:
			return ErrInvalidPDFformat
		}
	}

	args = append(args, "--output", outputPath, inputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, mod.unoconvBinPath, args...)
	if err != nil {
		return fmt.Errorf("create unoconv command: %w", err)
	}

	logger.Debug(fmt.Sprintf("convert document with: %+v", options))

	activeInstancesCountMu.Lock()
	activeInstancesCount += 1
	activeInstancesCountMu.Unlock()

	exitCode, err := cmd.Exec()

	activeInstancesCountMu.Lock()
	activeInstancesCount -= 1
	activeInstancesCountMu.Unlock()

	if err == nil {
		return nil
	}

	// Unoconv/LibreOffice errors are not explicit.
	// That's why we have to make an educated guess according to the exit code
	// and given inputs.

	if exitCode == 5 && options.PageRanges != "" {
		return ErrMalformedPageRanges
	}

	// Possible errors:
	// 1. Unoconv/LibreOffice failed for some reason.
	// 2. Context done.
	//
	// On the second scenario, LibreOffice might not have time to remove some
	// of its temporary files, as it has been killed without warning. The
	// garbage collector will delete them for us (if the module is loaded).
	return fmt.Errorf("unoconv Convert: %w", err)
}

// Extensions returns the file extensions available for conversions.
func (mod UNO) Extensions() []string {
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
	}
}

// UNO returns an API for interacting with LibreOffice.
func (mod UNO) UNO() (API, error) {
	return mod, nil
}

var (
	listenerActiveInstancesCount   float64
	listenerActiveInstancesCountMu sync.RWMutex
	activeInstancesCount           float64
	activeInstancesCountMu         sync.RWMutex
)

// Interface guards.
var (
	_ gotenberg.Module          = (*UNO)(nil)
	_ gotenberg.Provisioner     = (*UNO)(nil)
	_ gotenberg.Validator       = (*UNO)(nil)
	_ gotenberg.App             = (*UNO)(nil)
	_ gotenberg.MetricsProvider = (*UNO)(nil)
	_ api.HealthChecker         = (*UNO)(nil)
	_ API                       = (*UNO)(nil)
	_ Provider                  = (*UNO)(nil)
)
