package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
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
	// ErrInvalidPdfFormats happens if LibreOffice cannot handle the PDF
	// formats option.
	ErrInvalidPdfFormats = errors.New("invalid PDF formats")

	// ErrUnoException happens when unoconverter returns exit code 5.
	ErrUnoException = errors.New("uno exception")

	// ErrRuntimeException happens when unoconverter returns exit code 6.
	ErrRuntimeException = errors.New("uno exception")

	// ErrCoreDumped happens randomly; sometimes a conversion will work as
	// expected, and some other time the same conversion will fail.
	// See https://github.com/gotenberg/gotenberg/issues/639.
	ErrCoreDumped = errors.New("core dumped")
)

// Api is a module that provides a [Uno] to interact with LibreOffice.
type Api struct {
	autoStart bool
	args      libreOfficeArguments

	logger      *zap.Logger
	libreOffice libreOffice
	supervisor  gotenberg.ProcessSupervisor
}

// Options gathers available options when converting a document to PDF.
// See: https://help.libreoffice.org/latest/en-US/text/shared/guide/pdf_params.html.
type Options struct {
	// Password specifies the password for opening the source file.
	Password string

	// Landscape allows changing the orientation of the resulting PDF.
	Landscape bool

	// PageRanges allows selecting the pages to convert.
	PageRanges string

	// UpdateIndexes specifies whether to update the indexes before conversion,
	// keeping in mind that doing so might result in missing links in the final
	// PDF.
	UpdateIndexes bool

	// ExportFormFields specifies whether form fields are exported as widgets
	// or only their fixed print representation is exported.
	ExportFormFields bool

	// AllowDuplicateFieldNames specifies whether multiple form fields exported
	// are allowed to have the same field name.
	AllowDuplicateFieldNames bool

	// ExportBookmarks specifies if bookmarks are exported to PDF.
	ExportBookmarks bool

	// ExportBookmarksToPdfDestination specifies that the bookmarks contained
	// in the source LibreOffice file should be exported to the PDF file as
	// Named Destination.
	ExportBookmarksToPdfDestination bool

	// ExportPlaceholders exports the placeholder fields visual markings only.
	// The exported placeholder is ineffective.
	ExportPlaceholders bool

	// ExportNotes specifies if notes are exported to PDF.
	ExportNotes bool

	// ExportNotesPages specifies if notes pages are exported to PDF.
	// Notes pages are available in Impress documents only.
	ExportNotesPages bool

	// ExportOnlyNotesPages specifies if the property ExportNotesPages is set
	// to true if only notes pages are exported to PDF.
	ExportOnlyNotesPages bool

	// ExportNotesInMargin specifies if notes in the margin are exported to
	// PDF.
	ExportNotesInMargin bool

	// ConvertOooTargetToPdfTarget specifies that the target documents with
	// .od[tpgs] extension will have that extension changed to .pdf when the
	// link is exported to PDF. The source document remains untouched.
	ConvertOooTargetToPdfTarget bool

	// ExportLinksRelativeFsys specifies that the file system related
	// hyperlinks (file:// protocol) present in the document will be exported
	// as relative to the source document location.
	ExportLinksRelativeFsys bool

	// ExportHiddenSlides exports, for LibreOffice Impress, slides that are not
	// included in slide shows.
	ExportHiddenSlides bool

	// SkipEmptyPages specifies that automatically inserted empty pages are
	// suppressed. This option is active only if storing Writer documents.
	SkipEmptyPages bool

	// AddOriginalDocumentAsStream specifies that a stream is inserted to the
	// PDF file which contains the original document for archiving purposes.
	AddOriginalDocumentAsStream bool

	// SinglePageSheets ignores each sheet’s paper size, print ranges and
	// shown/hidden status and puts every sheet (even hidden sheets) on exactly
	// one page.
	SinglePageSheets bool

	// LosslessImageCompression specifies if images are exported to PDF using
	// a lossless compression format like PNG or compressed using the JPEG
	// format.
	LosslessImageCompression bool

	// Quality specifies the quality of the JPG export. A higher value produces
	// a higher-quality image and a larger file. Between 1 and 100.
	Quality int

	// ReduceImageResolution specifies if the resolution of each image is
	// reduced to the resolution specified by the property MaxImageResolution.
	ReduceImageResolution bool

	// MaxImageResolution, if the property ReduceImageResolution is set to
	// true, tells if all images will be reduced to the given value in DPI.
	// Possible values are: 75, 150, 300, 600 and 1200.
	MaxImageResolution int

	// PdfFormats allows to convert the resulting PDF to PDF/A-1b, PDF/A-2b,
	// PDF/A-3b and PDF/UA.
	PdfFormats gotenberg.PdfFormats
}

// DefaultOptions returns the default values for Options.
func DefaultOptions() Options {
	return Options{
		Password:                        "",
		Landscape:                       false,
		PageRanges:                      "",
		UpdateIndexes:                   true,
		ExportFormFields:                true,
		AllowDuplicateFieldNames:        false,
		ExportBookmarks:                 true,
		ExportBookmarksToPdfDestination: false,
		ExportPlaceholders:              false,
		ExportNotes:                     false,
		ExportNotesPages:                false,
		ExportOnlyNotesPages:            false,
		ExportNotesInMargin:             false,
		ConvertOooTargetToPdfTarget:     false,
		ExportLinksRelativeFsys:         false,
		ExportHiddenSlides:              false,
		SkipEmptyPages:                  false,
		AddOriginalDocumentAsStream:     false,
		SinglePageSheets:                false,
		LosslessImageCompression:        false,
		Quality:                         90,
		ReduceImageResolution:           false,
		MaxImageResolution:              300,
		PdfFormats: gotenberg.PdfFormats{
			PdfA:  "",
			PdfUa: false,
		},
	}
}

// Uno is an abstraction on top of the Universal Network Objects API.
type Uno interface {
	Pdf(ctx context.Context, logger *zap.Logger, inputPath, outputPath string, options Options) error
	Extensions() []string
}

// Provider is a module interface that exposes a method for creating a
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
	// Block until the context is done so that another module may gracefully
	// stop before we do a shutdown.
	a.logger.Debug("wait for the end of grace duration")

	<-ctx.Done()

	err := a.supervisor.Shutdown()
	if err == nil {
		return nil
	}

	return fmt.Errorf("stop LibreOffice: %w", err)
}

// Debug returns additional debug data.
func (a *Api) Debug() map[string]interface{} {
	debug := make(map[string]interface{})

	cmd := exec.Command(a.args.binPath, "--version") //nolint:gosec
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
func (a *Api) Metrics() ([]gotenberg.Metric, error) {
	return []gotenberg.Metric{
		{
			Name:        "libreoffice_requests_queue_size",
			Description: "Current number of LibreOffice conversion requests waiting to be treated.",
			Instrument:  gotenberg.HistogramInstrument,
			Read: func() float64 {
				return float64(a.supervisor.ReqQueueSize())
			},
		},
		{
			Name:        "libreoffice_restarts_count",
			Description: "Current number of LibreOffice restarts.",
			Instrument:  gotenberg.CounterInstrument,
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
	err := a.supervisor.Run(ctx, logger, func() error {
		return a.libreOffice.pdf(ctx, logger, inputPath, outputPath, options)
	})

	if err == nil {
		return nil
	}

	// See https://github.com/gotenberg/gotenberg/issues/639.
	if errors.Is(err, ErrCoreDumped) {
		logger.Debug(fmt.Sprintf("got a '%s' error, retry conversion", err))
		return a.Pdf(ctx, logger, inputPath, outputPath, options)
	}

	return fmt.Errorf("supervisor run task: %w", err)
}

// Extensions returns the file extensions available for conversions.
// FIXME: don't care, take all on the route level?
func (a *Api) Extensions() []string {
	return []string{
		".123",
		".602",
		".abw",
		".bib",
		".bmp",
		".cdr",
		".cgm",
		".cmx",
		".csv",
		".cwk",
		".dbf",
		".dif",
		".doc",
		".docm",
		".docx",
		".dot",
		".dotm",
		".dotx",
		".dxf",
		".emf",
		".eps",
		".epub",
		".fodg",
		".fodp",
		".fods",
		".fodt",
		".fopd",
		".gif",
		".htm",
		".html",
		".hwp",
		".jpeg",
		".jpg",
		".key",
		".ltx",
		".lwp",
		".mcw",
		".met",
		".mml",
		".mw",
		".numbers",
		".odd",
		".odg",
		".odm",
		".odp",
		".ods",
		".odt",
		".otg",
		".oth",
		".otp",
		".ots",
		".ott",
		".pages",
		".pbm",
		".pcd",
		".pct",
		".pcx",
		".pdb",
		".pdf",
		".pgm",
		".png",
		".pot",
		".potm",
		".potx",
		".ppm",
		".pps",
		".ppt",
		".pptm",
		".pptx",
		".psd",
		".psw",
		".pub",
		".pwp",
		".pxl",
		".ras",
		".rtf",
		".sda",
		".sdc",
		".sdd",
		".sdp",
		".sdw",
		".sgl",
		".slk",
		".smf",
		".stc",
		".std",
		".sti",
		".stw",
		".svg",
		".svm",
		".swf",
		".sxc",
		".sxd",
		".sxg",
		".sxi",
		".sxm",
		".sxw",
		".tga",
		".tif",
		".tiff",
		".txt",
		".uof",
		".uop",
		".uos",
		".uot",
		".vdx",
		".vor",
		".vsd",
		".vsdm",
		".vsdx",
		".wb2",
		".wk1",
		".wks",
		".wmf",
		".wpd",
		".wpg",
		".wps",
		".xbm",
		".xhtml",
		".xls",
		".xlsb",
		".xlsm",
		".xlsx",
		".xlt",
		".xltm",
		".xltx",
		".xlw",
		".xml",
		".xpm",
		".zabw",
	}
}

// Interface guards.
var (
	_ gotenberg.Module          = (*Api)(nil)
	_ gotenberg.Provisioner     = (*Api)(nil)
	_ gotenberg.Validator       = (*Api)(nil)
	_ gotenberg.App             = (*Api)(nil)
	_ gotenberg.Debuggable      = (*Api)(nil)
	_ gotenberg.MetricsProvider = (*Api)(nil)
	_ api.HealthChecker         = (*Api)(nil)
	_ Uno                       = (*Api)(nil)
	_ Provider                  = (*Api)(nil)
)
