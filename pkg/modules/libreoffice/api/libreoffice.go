package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

type libreOffice interface {
	gotenberg.Process
	pdf(ctx context.Context, logger *slog.Logger, inputPath, outputPath string, options Options) error
}

type libreOfficeArguments struct {
	// binPath is the soffice binary. Its directory is the LibreOffice program
	// directory lok loads LibreOffice from.
	binPath      string
	lokBinPath   string
	startTimeout time.Duration
	proxyOptions outboundProxyOptions
}

// libreOfficeProcess runs LibreOffice in a long-running lok process, which
// converts the documents it reads as JSON lines on its standard input, one at
// a time, and answers each with a JSON line on its standard output.
type libreOfficeProcess struct {
	userProfileDirPath string
	cmd                *gotenberg.Cmd
	stdin              io.WriteCloser
	stdout             *bufio.Reader
	exited             chan struct{}
	proxy              *libreOfficeProxy
	cfgMu              sync.RWMutex
	convMu             sync.Mutex
	isStarted          atomic.Bool

	arguments libreOfficeArguments
	fs        *gotenberg.FileSystem
}

func newLibreOfficeProcess(arguments libreOfficeArguments) libreOffice {
	p := &libreOfficeProcess{
		arguments: arguments,
		fs:        gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll)),
	}
	p.isStarted.Store(false)

	return p
}

// lokRequest is a conversion request of lok's long-running mode. Every option
// is sent, so lok's own defaults never apply.
type lokRequest struct {
	InputPath                       string `json:"inputPath"`
	OutputPath                      string `json:"outputPath"`
	Password                        string `json:"password,omitempty"` //nolint:gosec // document open password, sent to the local lok process only
	Landscape                       bool   `json:"landscape"`
	PaperFormat                     int    `json:"paperFormat"`
	PaperWidth                      int    `json:"paperWidth"`
	PaperHeight                     int    `json:"paperHeight"`
	PageRanges                      string `json:"pageRanges"`
	UpdateIndexes                   bool   `json:"updateIndexes"`
	DisableCalcHeader               bool   `json:"disableCalcHeader"`
	ExportFormFields                bool   `json:"exportFormFields"`
	AllowDuplicateFieldNames        bool   `json:"allowDuplicateFieldNames"`
	ExportBookmarks                 bool   `json:"exportBookmarks"`
	ExportBookmarksToPdfDestination bool   `json:"exportBookmarksToPdfDestination"`
	ExportPlaceholders              bool   `json:"exportPlaceholders"`
	ExportNotes                     bool   `json:"exportNotes"`
	ExportNotesPages                bool   `json:"exportNotesPages"`
	ExportOnlyNotesPages            bool   `json:"exportOnlyNotesPages"`
	ExportNotesInMargin             bool   `json:"exportNotesInMargin"`
	ConvertOooTargetToPdfTarget     bool   `json:"convertOooTargetToPdfTarget"`
	ExportLinksRelativeFsys         bool   `json:"exportLinksRelativeFsys"`
	ExportHiddenSlides              bool   `json:"exportHiddenSlides"`
	SkipEmptyPages                  bool   `json:"skipEmptyPages"`
	AddOriginalDocumentAsStream     bool   `json:"addOriginalDocumentAsStream"`
	SinglePageSheets                bool   `json:"singlePageSheets"`
	InitialView                     int    `json:"initialView"`
	InitialPage                     int    `json:"initialPage"`
	Magnification                   int    `json:"magnification"`
	Zoom                            int    `json:"zoom"`
	PageLayout                      int    `json:"pageLayout"`
	FirstPageOnLeft                 bool   `json:"firstPageOnLeft"`
	ResizeWindowToInitialPage       bool   `json:"resizeWindowToInitialPage"`
	CenterWindow                    bool   `json:"centerWindow"`
	OpenInFullScreenMode            bool   `json:"openInFullScreenMode"`
	DisplayPDFDocumentTitle         bool   `json:"displayPDFDocumentTitle"`
	HideViewerMenubar               bool   `json:"hideViewerMenubar"`
	HideViewerToolbar               bool   `json:"hideViewerToolbar"`
	HideViewerWindowControls        bool   `json:"hideViewerWindowControls"`
	UseTransitionEffects            bool   `json:"useTransitionEffects"`
	OpenBookmarkLevels              int    `json:"openBookmarkLevels"`
	LosslessImageCompression        bool   `json:"losslessImageCompression"`
	Quality                         int    `json:"quality"`
	ReduceImageResolution           bool   `json:"reduceImageResolution"`
	MaxImageResolution              int    `json:"maxImageResolution"`
	NativeWatermarkText             string `json:"nativeWatermarkText"`
	NativeWatermarkColor            int    `json:"nativeWatermarkColor"`
	NativeWatermarkFontHeight       int    `json:"nativeWatermarkFontHeight"`
	NativeWatermarkRotateAngle      int    `json:"nativeWatermarkRotateAngle"`
	NativeWatermarkFontName         string `json:"nativeWatermarkFontName"`
	NativeTiledWatermarkText        string `json:"nativeTiledWatermarkText"`
	PdfVersion                      int    `json:"pdfVersion"`
	PdfUniversalAccess              bool   `json:"pdfUniversalAccess"`
	TaggedPdf                       bool   `json:"taggedPdf"`
	EmbedStandardFonts              bool   `json:"embedStandardFonts"`
}

// lokResponse is the answer of lok's long-running mode to a request.
type lokResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Error   string `json:"error"`
}

// lok's paper formats: unset keeps the document's size, user takes the width
// and height of the request.
const (
	lokPaperFormatUnset = -1
	lokPaperFormatUser  = 8
)

// pdfVersions maps the native PDF/A formats to lok's SelectPdfVersion values.
var pdfVersions = map[string]int{
	"":     0,
	PdfA1b: 1,
	PdfA2b: 2,
	PdfA3b: 3,
	PdfA4:  4,
}

// newLokRequest builds the lok request for a conversion.
func newLokRequest(inputPath, outputPath string, options Options) (lokRequest, error) {
	pdfVersion, ok := pdfVersions[options.PdfFormats.PdfA]
	if !ok {
		return lokRequest{}, ErrInvalidPdfFormats
	}

	// LibreOffice's default watermark font. Leaving it out keeps lok's default
	// identical to it.
	fontName := options.NativeWatermarkFontName
	if fontName == "Helvetica" {
		fontName = ""
	}

	// lok's custom paper format takes the size in 1/100 mm; -1 keeps the
	// document's own size.
	paperFormat, paperWidth, paperHeight := lokPaperFormatUnset, 0, 0
	if options.PaperWidth > 0 && options.PaperHeight > 0 {
		paperFormat = lokPaperFormatUser
		paperWidth = int(math.Round(options.PaperWidth * 2540))
		paperHeight = int(math.Round(options.PaperHeight * 2540))
	}

	return lokRequest{
		InputPath:   inputPath,
		OutputPath:  outputPath,
		Password:    options.Password,
		Landscape:   options.Landscape,
		PaperFormat: paperFormat,
		PaperWidth:  paperWidth,
		PaperHeight: paperHeight,
		PageRanges:  options.PageRanges,
		// See https://github.com/gotenberg/gotenberg/issues/1149.
		UpdateIndexes: options.UpdateIndexes,
		// A CSV becomes a single Calc sheet named after the input file, and
		// Calc's default page style prints that sheet name as a centered
		// header. Uploads are stored under a UUID-based filename, so the UUID
		// would otherwise leak into the rendered PDF. Spreadsheets that carry
		// their own page styles (XLSX, ODS) are left untouched.
		// See https://github.com/gotenberg/gotenberg/issues/1568.
		DisableCalcHeader:               strings.EqualFold(filepath.Ext(inputPath), ".csv"),
		ExportFormFields:                options.ExportFormFields,
		AllowDuplicateFieldNames:        options.AllowDuplicateFieldNames,
		ExportBookmarks:                 options.ExportBookmarks,
		ExportBookmarksToPdfDestination: options.ExportBookmarksToPdfDestination,
		ExportPlaceholders:              options.ExportPlaceholders,
		ExportNotes:                     options.ExportNotes,
		ExportNotesPages:                options.ExportNotesPages,
		ExportOnlyNotesPages:            options.ExportOnlyNotesPages,
		ExportNotesInMargin:             options.ExportNotesInMargin,
		ConvertOooTargetToPdfTarget:     options.ConvertOooTargetToPdfTarget,
		ExportLinksRelativeFsys:         options.ExportLinksRelativeFsys,
		ExportHiddenSlides:              options.ExportHiddenSlides,
		SkipEmptyPages:                  options.SkipEmptyPages,
		AddOriginalDocumentAsStream:     options.AddOriginalDocumentAsStream,
		SinglePageSheets:                options.SinglePageSheets,
		InitialView:                     options.InitialView,
		InitialPage:                     options.InitialPage,
		Magnification:                   options.Magnification,
		Zoom:                            options.Zoom,
		PageLayout:                      options.PageLayout,
		FirstPageOnLeft:                 options.FirstPageOnLeft,
		ResizeWindowToInitialPage:       options.ResizeWindowToInitialPage,
		CenterWindow:                    options.CenterWindow,
		OpenInFullScreenMode:            options.OpenInFullScreenMode,
		DisplayPDFDocumentTitle:         options.DisplayPDFDocumentTitle,
		HideViewerMenubar:               options.HideViewerMenubar,
		HideViewerToolbar:               options.HideViewerToolbar,
		HideViewerWindowControls:        options.HideViewerWindowControls,
		UseTransitionEffects:            options.UseTransitionEffects,
		OpenBookmarkLevels:              options.OpenBookmarkLevels,
		LosslessImageCompression:        options.LosslessImageCompression,
		Quality:                         options.Quality,
		ReduceImageResolution:           options.ReduceImageResolution,
		MaxImageResolution:              options.MaxImageResolution,
		NativeWatermarkText:             options.NativeWatermarkText,
		NativeWatermarkColor:            options.NativeWatermarkColor,
		NativeWatermarkFontHeight:       options.NativeWatermarkFontHeight,
		NativeWatermarkRotateAngle:      options.NativeWatermarkRotateAngle,
		NativeWatermarkFontName:         fontName,
		NativeTiledWatermarkText:        options.NativeTiledWatermarkText,
		PdfVersion:                      pdfVersion,
		PdfUniversalAccess:              options.PdfFormats.PdfUa,
		TaggedPdf:                       options.GenerateTaggedPdf,
		// PDF/A and PDF/UA require every font embedded.
		EmbedStandardFonts: pdfVersion != 0 || options.PdfFormats.PdfUa,
	}, nil
}

func (p *libreOfficeProcess) Start(ctx context.Context, logger *slog.Logger) error {
	if p.isStarted.Load() {
		return errors.New("LibreOffice is already started")
	}

	proxy, err := newLibreOfficeProxy(logger, p.arguments.proxyOptions)
	if err != nil {
		return fmt.Errorf("create LibreOffice outbound proxy: %w", err)
	}
	proxy.Start()

	userProfileDirPath := p.fs.NewDirPath()

	var (
		cmd     *gotenberg.Cmd
		stdin   io.WriteCloser
		stdout  *bufio.Reader
		exited  chan struct{}
		success bool
	)

	// Registered here, right after the proxy starts listening, so that every
	// failure below tears it down. A return between the proxy start and this
	// point strands it: its listener stays bound, its Serve goroutine and HTTP
	// client stay alive, and p.proxy is only assigned on success, so nothing
	// could ever reach it to stop it. exec.Cmd.Start fails precisely under fd
	// or memory pressure, and the supervisor retries the launch on the next
	// request, so each stranded proxy compounds the condition that caused it.
	defer func() {
		if success {
			p.cfgMu.Lock()
			defer p.cfgMu.Unlock()

			p.userProfileDirPath = userProfileDirPath
			p.cmd = cmd
			p.stdin = stdin
			p.stdout = stdout
			p.exited = exited
			p.proxy = proxy
			p.isStarted.Store(true)

			return
		}

		// LibreOffice failed to start; tear the proxy down too.
		stopErr := proxy.Stop(context.Background())
		if stopErr != nil {
			logger.WarnContext(context.Background(), fmt.Sprintf("stop LibreOffice outbound proxy after failed start: %s", stopErr))
		}

		// Let's make sure the process is killed. It is nil when the failure
		// happened before the command was built.
		if cmd != nil {
			killErr := cmd.Kill()
			if killErr != nil {
				logger.DebugContext(context.Background(), fmt.Sprintf("kill LibreOffice process: %v", killErr))
			}
		}

		// And the user profile directory is deleted. It may never have been
		// created, which RemoveAll reports as success.
		removeErr := os.RemoveAll(userProfileDirPath)
		if removeErr != nil {
			logger.ErrorContext(context.Background(), fmt.Sprintf("remove LibreOffice's user profile directory: %v", removeErr))

			return
		}

		logger.DebugContext(context.Background(), fmt.Sprintf("'%s' LibreOffice's user profile directory removed", userProfileDirPath))
	}()

	// LibreOffice fetches external content (OOXML images via
	// TargetMode=External, RTF INCLUDEPICTURE, ODT linked images) inside
	// its own libcurl. The profile config routes those fetches through the
	// in-process proxy so the chromium/webhook SSRF filters apply, and
	// blocks content linked from untrusted locations so absolute-path
	// (file://) and direct fetches are dropped at the source. lok loads
	// LibreOffice in its own process, against this profile.
	err = writeProfileConfig(userProfileDirPath, proxy.Addr())
	if err != nil {
		return fmt.Errorf("write LibreOffice profile config: %w", err)
	}

	cmd = gotenberg.Command(
		logger,
		p.arguments.lokBinPath,
		"--long-running",
		"--libreoffice-program-path", filepath.Dir(p.arguments.binPath),
		"--user-profile-path", userProfileDirPath,
	)
	cmd.SetEnv(proxyEnv(os.Environ(), proxy.Addr()))

	in, out, err := cmd.StartWithPipes()
	if err != nil {
		return fmt.Errorf("start lok: %w", err)
	}

	stdin = in
	stdout = bufio.NewReader(out)
	exited = make(chan struct{})

	go func() { //nolint:gosec // G118: the process outlives the start request
		// By waiting the process, we avoid the creation of a zombie process
		// and learn when it exits, which makes it unhealthy.
		waitErr := cmd.Wait()
		if waitErr != nil {
			logger.DebugContext(context.Background(), fmt.Sprintf("lok exited: %s", waitErr))
		}

		close(exited)
	}()

	// lok reads its standard input only once LibreOffice is loaded. A request
	// without paths is answered right away, so its answer tells that the
	// process is ready.
	readyCtx, cancel := context.WithTimeout(ctx, p.arguments.startTimeout)
	defer cancel()

	logger.DebugContext(readyCtx, "waiting for LibreOffice to be loaded...")

	resp, err := exchange(readyCtx, stdin, stdout, exited, struct{}{})
	if err != nil && ctx.Err() != nil {
		// The request gave up first: the start timeout is not to blame.
		return fmt.Errorf("start LibreOffice: %w", err)
	}
	if err != nil {
		return fmt.Errorf("LibreOffice did not load within the start timeout; increase --libreoffice-start-timeout (env LIBREOFFICE_START_TIMEOUT) or check system resources: %w", err)
	}

	if resp.Code != lokCodeInvalidRequest {
		return fmt.Errorf("unexpected lok answer to the readiness request: %+v", resp)
	}

	logger.DebugContext(readyCtx, "LibreOffice loaded")
	success = true

	return nil
}

func (p *libreOfficeProcess) Stop(logger *slog.Logger) error {
	if !p.isStarted.Load() {
		// No big deal? Like calling cancel twice.
		return nil
	}

	// Always remove the user profile directory created for LibreOffice.
	copyUserProfileDirPath := p.userProfileDirPath
	expirationTime := time.Now()
	defer func(userProfileDirPath string, expirationTime time.Time) {
		go func() {
			err := os.RemoveAll(userProfileDirPath)
			if err != nil {
				logger.ErrorContext(context.Background(), fmt.Sprintf("remove LibreOffice's user profile directory: %v", err))
			} else {
				logger.DebugContext(context.Background(), fmt.Sprintf("'%s' LibreOffice's user profile directory removed", userProfileDirPath))
			}

			// Also, remove LibreOffice specific files in the temporary directory.
			err = gotenberg.GarbageCollect(context.Background(), logger, os.TempDir(), []string{"OSL_PIPE", ".tmp", "lok-"}, expirationTime)
			if err != nil {
				logger.ErrorContext(context.Background(), err.Error())
			}
		}()
	}(copyUserProfileDirPath, expirationTime)

	p.cfgMu.Lock()
	defer p.cfgMu.Unlock()

	err := p.cmd.Kill()
	if err != nil {
		return fmt.Errorf("kill lok process: %w", err)
	}

	if p.proxy != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		stopErr := p.proxy.Stop(shutdownCtx)
		cancel()
		if stopErr != nil {
			logger.WarnContext(context.Background(), fmt.Sprintf("stop LibreOffice outbound proxy: %s", stopErr))
		}
		p.proxy = nil
	}

	p.userProfileDirPath = ""
	p.cmd = nil
	p.stdin = nil
	p.stdout = nil
	p.isStarted.Store(false)

	return nil
}

func (p *libreOfficeProcess) Healthy(_ *slog.Logger) bool {
	// Good to know: the supervisor does not call this method if no first start
	// or if the process is restarting.

	if !p.isStarted.Load() {
		return false
	}

	p.cfgMu.RLock()
	defer p.cfgMu.RUnlock()

	select {
	case <-p.exited:
		return false
	default:
		return true
	}
}

func (p *libreOfficeProcess) pdf(ctx context.Context, logger *slog.Logger, inputPath, outputPath string, options Options) error {
	if !p.isStarted.Load() {
		return errors.New("LibreOffice not started, cannot handle PDF conversion")
	}

	req, err := newLokRequest(inputPath, outputPath, options)
	if err != nil {
		return err
	}

	// lok converts one document at a time, and a request must read its own
	// answer.
	p.convMu.Lock()
	defer p.convMu.Unlock()

	p.cfgMu.RLock()
	cmd, stdin, stdout, exited := p.cmd, p.stdin, p.stdout, p.exited
	p.cfgMu.RUnlock()

	logger.DebugContext(ctx, fmt.Sprintf("print to PDF with: %+v", options))

	resp, err := exchange(ctx, stdin, stdout, exited, req)
	if err != nil {
		if ctx.Err() != nil {
			// LibreOfficeKit calls cannot be cancelled: kill lok so that its
			// late answer never reaches the next request. The process is then
			// unhealthy, and the supervisor restarts it.
			killErr := cmd.Kill()
			if killErr != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("kill lok after cancellation: %s", killErr))
			}

			return ctx.Err()
		}

		// lok exited mid-conversion: LibreOffice crashed. The same
		// conversion often succeeds once restarted.
		// See https://github.com/gotenberg/gotenberg/issues/639.
		//
		// The read can fail before the process is reaped, while lok still
		// reports healthy. Make sure it is gone, so that the supervisor
		// restarts it instead of handing it the next request.
		killErr := cmd.Kill()
		if killErr != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("kill lok after a failed exchange: %s", killErr))
		}

		select {
		case <-exited:
		case <-time.After(reapTimeout):
			logger.ErrorContext(ctx, "lok not reaped after a failed exchange")
		}

		return fmt.Errorf("%w: %w", ErrCrashed, err)
	}

	if resp.Success {
		return nil
	}

	return lokError(resp)
}

// reapTimeout bounds the wait for a killed lok process to be reaped.
const reapTimeout = 5 * time.Second

// Codes of lok's long-running answers.
const (
	lokCodeInvalidRequest  = "invalid_request"
	lokCodeInvalidPassword = "invalid_password"
	lokCodeLoadFailed      = "load_failed"
	lokCodeSaveFailed      = "save_failed"
)

// lokError maps a failed lok answer to the matching sentinel error.
func lokError(resp lokResponse) error {
	detail := errors.New(resp.Error)

	switch resp.Code {
	case lokCodeInvalidPassword:
		return fmt.Errorf("%w: %w", ErrInvalidPassword, detail)
	case lokCodeLoadFailed:
		return fmt.Errorf("%w: %w", ErrLoadFailed, detail)
	case lokCodeSaveFailed:
		return fmt.Errorf("%w: %w", ErrSaveFailed, detail)
	default:
		return fmt.Errorf("lok %s: %w", resp.Code, detail)
	}
}

// exchange writes a request line to lok and reads its answer line. It returns
// early when ctx is done or the process exits, and then leaves the read
// behind: only killing the process ends it.
func exchange(ctx context.Context, stdin io.Writer, stdout *bufio.Reader, exited <-chan struct{}, req any) (lokResponse, error) {
	line, err := json.Marshal(req)
	if err != nil {
		return lokResponse{}, fmt.Errorf("encode lok request: %w", err)
	}

	type result struct {
		resp lokResponse
		err  error
	}

	done := make(chan result, 1)

	go func() {
		_, err := stdin.Write(append(line, '\n'))
		if err != nil {
			done <- result{err: fmt.Errorf("write lok request: %w", err)}
			return
		}

		answer, err := stdout.ReadBytes('\n')
		if err != nil {
			done <- result{err: fmt.Errorf("read lok answer: %w", err)}
			return
		}

		var resp lokResponse
		err = json.Unmarshal(answer, &resp)
		if err != nil {
			done <- result{err: fmt.Errorf("decode lok answer %q: %w", answer, err)}
			return
		}

		done <- result{resp: resp}
	}()

	select {
	case r := <-done:
		return r.resp, r.err
	case <-ctx.Done():
		return lokResponse{}, ctx.Err()
	case <-exited:
		// The answer may have arrived just before the exit.
		select {
		case r := <-done:
			return r.resp, r.err
		default:
			return lokResponse{}, errors.New("lok exited")
		}
	}
}

// Interface guards.
var (
	_ gotenberg.Process = (*libreOfficeProcess)(nil)
	_ libreOffice       = (*libreOfficeProcess)(nil)
)
