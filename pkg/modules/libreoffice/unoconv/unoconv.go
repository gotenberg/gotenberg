package unoconv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(Unoconv{})
}

// ErrMalformedPageRanges happens if the page ranges option cannot be
// interpreted by LibreOffice.
var ErrMalformedPageRanges = errors.New("page ranges are malformed")

// Unoconv is a module which provides an API to interact with unoconv.
type Unoconv struct {
	binPath         string
	disableListener bool

	listenerCmd  gotenberg.Cmd
	listenerPort int
	logger       *zap.Logger
}

// Options gathers available options when converting a document to PDF.
type Options struct {
	// Landscape allows to change the orientation of the resulting PDF.
	// Optional.
	Landscape bool

	// PageRanges allows to select the pages to convert.
	// TODO: should prefer a method form PDFEngine.
	// Optional.
	PageRanges string

	// PDFArchive allows to convert the resulting PDF to PDF/A-1a.
	// In a module, prefer the Convert method from the gotenberg.PDFEngine
	// interface.
	// Optional.
	PDFArchive bool
}

// API is an abstraction on top of unoconv.
//
// See https://github.com/unoconv/unoconv.
type API interface {
	PDF(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error
	Extensions() []string
}

// Provider is a module interface which exposes a method for creating an API
// for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(unoconv.Provider))
//		uno, _      := provider.(unoconv.Provider).Unoconv()
//	}
type Provider interface {
	Unoconv() (API, error)
}

// Descriptor returns a Unoconv's module descriptor.
func (Unoconv) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "unoconv",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("unoconv", flag.ExitOnError)
			fs.Bool("unoconv-disable-listener", false, "Do not start a unoconv listener - save resources in detriment of performance")

			return fs
		}(),
		New: func() gotenberg.Module { return new(Unoconv) },
	}
}

// Provision sets the module properties. It returns an error if the environment
// variable UNOCONV_BIN_PATH is not set.
func (mod *Unoconv) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	mod.disableListener = flags.MustBool("unoconv-disable-listener")

	binPath, ok := os.LookupEnv("UNOCONV_BIN_PATH")
	if !ok {
		return errors.New("UNOCONV_BIN_PATH environment variable is not set")
	}

	mod.binPath = binPath

	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}

	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(mod)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}

	mod.logger = logger

	return nil
}

// Validate validates the module properties.
func (mod Unoconv) Validate() error {
	_, err := os.Stat(mod.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("unoconv binary path does not exist: %w", err)
	}

	return nil
}

func (mod *Unoconv) Start() error {
	if mod.disableListener {
		return nil
	}

	port, err := freePort(mod.logger)
	if err != nil {
		return fmt.Errorf("get free port: %w", err)
	}

	mod.listenerPort = port

	args := []string{
		"--listener",
		"--user-profile",
		// Just to make sure LibreOffice does not leak files in an unknown
		// directory. The directory will be removed anyway by the garbage
		// collector.
		fmt.Sprintf("//%s", gotenberg.NewDirPath()),
		"--port",
		fmt.Sprintf("%d", mod.listenerPort),
	}

	checkedEntry := mod.logger.Check(zap.DebugLevel, "check for debug level before setting high verbosity")
	if checkedEntry != nil {
		args = append(args, "-vvv")
	}

	mod.listenerCmd = gotenberg.Command(mod.logger, mod.binPath, args...)

	err = mod.listenerCmd.Start()
	if err != nil {
		return fmt.Errorf("start unoconv listener: %w", err)
	}

	listenerActiveInstancesCountMu.Lock()
	listenerActiveInstancesCount += 1
	listenerActiveInstancesCountMu.Unlock()

	return nil
}

// StartupMessage returns a custom startup message.
func (mod Unoconv) StartupMessage() string {
	if mod.disableListener {
		return "listener disabled"
	}

	return fmt.Sprintf("listener started on port %d", mod.listenerPort)
}

// Stop stops the HTTP server.
func (mod *Unoconv) Stop(ctx context.Context) error {
	if mod.disableListener {
		return nil
	}

	_, ok := ctx.Deadline()
	if !ok {
		return errors.New("no context dead line")
	}

	// Block until the context is done so that other module may gracefully stop
	// before we do a shutdown cleanup.
	mod.logger.Debug("wait for the end of grace duration")

	<-ctx.Done()

	err := mod.listenerCmd.Kill()
	if err != nil {
		return fmt.Errorf("kill unoconv listener: %w", err)
	}

	listenerActiveInstancesCountMu.Lock()
	listenerActiveInstancesCount -= 1
	listenerActiveInstancesCountMu.Unlock()

	return nil
}

// Metrics returns the metrics.
func (mod Unoconv) Metrics() ([]gotenberg.Metric, error) {
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
			Name:        "unoconv_listener_active_instances_count",
			Description: "Current number of active unoconv listener instances.",
			Read: func() float64 {
				listenerActiveInstancesCountMu.RLock()
				defer listenerActiveInstancesCountMu.RUnlock()

				return listenerActiveInstancesCount
			},
		},
		{
			Name:        "unoconv_listener_queue_length",
			Description: "Current number of processes in the queue.",
			Read: func() float64 {
				listenerQueueLengthMu.RLock()
				defer listenerQueueLengthMu.RUnlock()

				return listenerQueueLength
			},
		},
	}, nil
}

// Unoconv returns an API for interacting with unoconv.
func (mod *Unoconv) Unoconv() (API, error) {
	return mod, nil
}

// PDF converts a document to PDF.
//
// In stateless mode, it creates a dedicated LibreOffice instance thanks to a
// custom user profile directory and a free port. Substantial calls to this
// method may increase CPU and memory usage drastically. In such a scenario,
// the given context may also be done before the end of the conversion.
//
// In listener mode, it calls the unoconv listener to interact with
// LibreOffice, improving substantially the performance. However, it cannot
// perform parallel operations and have to wait for the lock to be available.
func (mod Unoconv) PDF(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error {
	args := []string{
		"--format",
		"pdf",
	}

	var userProfileDirPath string

	if mod.disableListener {
		port, err := freePort(logger)
		if err != nil {
			return fmt.Errorf("get free port: %w", err)
		}

		userProfileDirPath = gotenberg.NewDirPath()

		args = append(args,
			"--port",
			fmt.Sprintf("%d", port),
			"--user-profile",
			fmt.Sprintf("//%s", userProfileDirPath),
		)
	} else {
		args = append(args, "--port", fmt.Sprintf("%d", mod.listenerPort))
	}

	checkedEntry := logger.Check(zap.DebugLevel, "check for debug level before setting high verbosity")
	if checkedEntry != nil {
		args = append(args, "-vvv")
	}

	if options.Landscape {
		args = append(args, "--printer", "PaperOrientation=landscape")
	}

	if options.PageRanges != "" {
		args = append(args, "--export", fmt.Sprintf("PageRange=%s", options.PageRanges))
	}

	if options.PDFArchive {
		args = append(args, "--export", "SelectPdfVersion=1")
	}

	args = append(args, "--output", outputPath, inputPath)

	if !mod.disableListener {
		listenerQueueLengthMu.Lock()
		listenerQueueLength += 1
		listenerQueueLengthMu.Unlock()

		select {
		case listenerLock <- struct{}{}:
			logger.Debug("unoconv lock acquired")

			listenerQueueLengthMu.Lock()
			listenerQueueLength -= 1
			listenerQueueLengthMu.Unlock()

			break
		case <-ctx.Done():
			logger.Debug("failed to acquire the unoconv lock before deadline")

			listenerQueueLengthMu.Lock()
			listenerQueueLength -= 1
			listenerQueueLengthMu.Unlock()

			return fmt.Errorf("acquire unoconv lock: %w", ctx.Err())
		}

		defer func() {
			<-listenerLock
			logger.Debug("unoconv lock released")
		}()
	}

	cmd, err := gotenberg.CommandContext(ctx, logger, mod.binPath, args...)
	if err != nil {
		return fmt.Errorf("create unoconv command: %w", err)
	}

	logger.Debug(fmt.Sprintf("print to PDF with: %+v", options))

	activeInstancesCountMu.Lock()
	activeInstancesCount += 1
	activeInstancesCountMu.Unlock()

	err = cmd.Exec()

	activeInstancesCountMu.Lock()
	activeInstancesCount -= 1
	activeInstancesCountMu.Unlock()

	if mod.disableListener {
		// Always remove the user profile directory created by LibreOffice.
		// See https://github.com/gotenberg/gotenberg/issues/192.
		go func() {
			logger.Debug(fmt.Sprintf("remove user profile directory '%s'", userProfileDirPath))

			err := os.RemoveAll(userProfileDirPath)
			if err != nil {
				logger.Error(fmt.Sprintf("remove user profile directory: %s", err))
			}
		}()
	}

	if err == nil {
		return nil
	}

	// Unoconv/LibreOffice errors are not explicit.
	// That's why we have to make an educated guess according to the exit code
	// and given inputs.

	if strings.Contains(err.Error(), "exit status 5") && options.PageRanges != "" {
		return ErrMalformedPageRanges
	}

	// Possible errors:
	// 1. Unoconv/LibreOffice failed for some reason.
	// 2. Context done.
	//
	// On the second scenario, LibreOffice might not had time to remove some of
	// its temporary files, as it has been killed without warning. The garbage
	// collector will delete them for us (if the module is loaded).
	return fmt.Errorf("unoconv PDF: %w", err)
}

// Extensions returns the file extensions available with unoconv.
func (mod Unoconv) Extensions() []string {
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

var (
	listenerLock                   = make(chan struct{}, 1)
	listenerQueueLength            float64
	listenerQueueLengthMu          sync.RWMutex
	listenerActiveInstancesCount   float64
	listenerActiveInstancesCountMu sync.RWMutex
	activeInstancesCount           float64
	activeInstancesCountMu         sync.RWMutex
)

// Interface guards.
var (
	_ gotenberg.Module          = (*Unoconv)(nil)
	_ gotenberg.Provisioner     = (*Unoconv)(nil)
	_ gotenberg.Validator       = (*Unoconv)(nil)
	_ gotenberg.App             = (*Unoconv)(nil)
	_ gotenberg.MetricsProvider = (*Unoconv)(nil)
	_ API                       = (*Unoconv)(nil)
	_ Provider                  = (*Unoconv)(nil)
)
