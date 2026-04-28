package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

type libreOffice interface {
	gotenberg.Process
	pdf(ctx context.Context, logger *slog.Logger, inputPath, outputPath string, options Options) error
}

type libreOfficeArguments struct {
	binPath      string
	unoBinPath   string
	startTimeout time.Duration
	proxyOptions outboundProxyOptions
}

type libreOfficeProcess struct {
	socketPort         int
	userProfileDirPath string
	cmd                *gotenberg.Cmd
	proxy              *libreOfficeProxy
	cfgMu              sync.RWMutex
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

func (p *libreOfficeProcess) Start(logger *slog.Logger) error {
	if p.isStarted.Load() {
		return errors.New("LibreOffice is already started")
	}

	port, err := freePort(logger)
	if err != nil {
		return fmt.Errorf("get free port: %w", err)
	}

	proxy, err := newLibreOfficeProxy(logger, p.arguments.proxyOptions)
	if err != nil {
		return fmt.Errorf("create LibreOffice outbound proxy: %w", err)
	}
	proxy.Start()

	userProfileDirPath := p.fs.NewDirPath()

	// LibreOffice fetches external content (OOXML images via
	// TargetMode=External, RTF INCLUDEPICTURE, ODT linked images) inside
	// its own libcurl. Route those fetches through the in-process proxy
	// so the chromium/webhook SSRF filters apply.
	if err := writeSofficeProxyConfig(userProfileDirPath, proxy.Addr()); err != nil {
		_ = proxy.Stop(context.Background())
		return fmt.Errorf("write soffice proxy config: %w", err)
	}
	sofficeEnv := sofficeProxyEnv(os.Environ(), proxy.Addr())

	args := []string{
		"--headless",
		"--invisible",
		"--nocrashreport",
		"--nodefault",
		"--nologo",
		"--nofirststartwizard",
		"--norestore",
		fmt.Sprintf("-env:UserInstallation=file://%s", userProfileDirPath),
		fmt.Sprintf("--accept=socket,host=127.0.0.1,port=%d,tcpNoDelay=1;urp;StarOffice.ComponentContext", port),
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.arguments.startTimeout)
	defer cancel()

	cmd, err := gotenberg.CommandContext(ctx, logger, p.arguments.binPath, args...)
	if err != nil {
		_ = proxy.Stop(context.Background())
		return fmt.Errorf("create LibreOffice command: %w", err)
	}
	cmd.SetEnv(sofficeEnv)

	// For whatever reason, LibreOffice requires a first start before being
	// able to run as a daemon.
	exitCode, err := cmd.Exec()
	if err != nil && exitCode != 81 {
		_ = proxy.Stop(context.Background())
		return fmt.Errorf("execute LibreOffice: %w", err)
	}

	logger.DebugContext(context.Background(), "got exit code 81, e.g., LibreOffice first start")

	// Second start (daemon).
	cmd = gotenberg.Command(logger, p.arguments.binPath, args...)
	cmd.SetEnv(sofficeEnv)

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start LibreOffice: %w", err)
	}

	waitChan := make(chan error, 1)

	go func() {
		// By waiting the process, we avoid the creation of a zombie process
		// and make sure we catch an early exit if any.
		waitChan <- cmd.Wait()
	}()

	connChan := make(chan error, 1)

	go func() {
		// As the LibreOffice socket may take some time to be available, we
		// have to ensure that it is indeed accepting connections.
		for {
			if ctx.Err() != nil {
				connChan <- ctx.Err()
				break
			}

			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Duration(1)*time.Second)
			if err != nil {
				continue
			}

			connChan <- nil
			err = conn.Close()
			if err != nil {
				logger.DebugContext(context.Background(), fmt.Sprintf("close connection after health checking the LibreOffice: %v", err))
			}

			break
		}
	}()

	var success bool

	defer func() {
		if success {
			p.cfgMu.Lock()
			defer p.cfgMu.Unlock()

			p.socketPort = port
			p.userProfileDirPath = userProfileDirPath
			p.cmd = cmd
			p.proxy = proxy
			p.isStarted.Store(true)

			return
		}

		// LibreOffice failed to start; tear the proxy down too.
		stopErr := proxy.Stop(context.Background())
		if stopErr != nil {
			logger.WarnContext(context.Background(), fmt.Sprintf("stop LibreOffice outbound proxy after failed start: %s", stopErr))
		}

		// Let's make sure the process is killed.
		err = cmd.Kill()
		if err != nil {
			logger.DebugContext(context.Background(), fmt.Sprintf("kill LibreOffice process: %v", err))
		}

		// And the user profile directory is deleted.
		err = os.RemoveAll(userProfileDirPath)
		if err != nil {
			logger.ErrorContext(context.Background(), fmt.Sprintf("remove LibreOffice's user profile directory: %v", err))
		}

		logger.DebugContext(context.Background(), fmt.Sprintf("'%s' LibreOffice's user profile directory removed", userProfileDirPath))
	}()

	logger.DebugContext(context.Background(), "waiting for the LibreOffice socket to be available...")

	for {
		select {
		case err = <-connChan:
			if err != nil {
				return fmt.Errorf("LibreOffice socket not available: %w", err)
			}

			logger.DebugContext(context.Background(), "LibreOffice socket available")
			success = true

			return nil
		case err = <-waitChan:
			return fmt.Errorf("LibreOffice process exited: %w", err)
		}
	}
}

func (p *libreOfficeProcess) Stop(logger *slog.Logger) error {
	if !p.isStarted.Load() {
		// No big deal? Like calling cancel twice.
		return nil
	}

	// Always remove the user profile directory created by LibreOffice.
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
			err = gotenberg.GarbageCollect(context.Background(), logger, os.TempDir(), []string{"OSL_PIPE", ".tmp"}, expirationTime)
			if err != nil {
				logger.ErrorContext(context.Background(), err.Error())
			}
		}()
	}(copyUserProfileDirPath, expirationTime)

	p.cfgMu.Lock()
	defer p.cfgMu.Unlock()

	err := p.cmd.Kill()
	if err != nil {
		return fmt.Errorf("kill LibreOffice process: %w", err)
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

	p.socketPort = 0
	p.userProfileDirPath = ""
	p.cmd = nil
	p.isStarted.Store(false)

	return nil
}

func (p *libreOfficeProcess) Healthy(logger *slog.Logger) bool {
	// Good to know: the supervisor does not call this method if no first start
	// or if the process is restarting.

	if !p.isStarted.Load() {
		// Non-started browser but not restarting?
		return false
	}

	p.cfgMu.RLock()
	defer p.cfgMu.RUnlock()

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p.socketPort), time.Duration(10)*time.Second)
	if err == nil {
		err = conn.Close()
		if err != nil {
			logger.DebugContext(context.Background(), fmt.Sprintf("close connection after health checking LibreOffice: %v", err))
		}

		return true
	}

	return false
}

func (p *libreOfficeProcess) pdf(ctx context.Context, logger *slog.Logger, inputPath, outputPath string, options Options) error {
	if !p.isStarted.Load() {
		return errors.New("LibreOffice not started, cannot handle PDF conversion")
	}

	args := []string{
		"--no-launch",
		"--format",
		"pdf",
	}

	args = append(args, "--port", fmt.Sprintf("%d", p.socketPort))

	if logger.Enabled(ctx, slog.LevelDebug) {
		args = append(args, "-vvv")
	}

	if options.Password != "" {
		args = append(args, "--password", options.Password)
	}

	if options.Landscape {
		args = append(args, "--printer", "PaperOrientation=landscape")
	}

	// See: https://github.com/gotenberg/gotenberg/issues/1149.
	if options.PageRanges != "" {
		args = append(args, "--export", fmt.Sprintf("PageRange=%s", options.PageRanges))
	}

	if !options.UpdateIndexes {
		args = append(args, "--disable-update-indexes")
	}

	args = append(args, "--export", fmt.Sprintf("ExportFormFields=%t", options.ExportFormFields))
	args = append(args, "--export", fmt.Sprintf("AllowDuplicateFieldNames=%t", options.AllowDuplicateFieldNames))
	args = append(args, "--export", fmt.Sprintf("ExportBookmarks=%t", options.ExportBookmarks))
	args = append(args, "--export", fmt.Sprintf("ExportBookmarks=%t", options.ExportBookmarks))
	args = append(args, "--export", fmt.Sprintf("ExportBookmarksToPDFDestination=%t", options.ExportBookmarksToPdfDestination))
	args = append(args, "--export", fmt.Sprintf("ExportPlaceholders=%t", options.ExportPlaceholders))
	args = append(args, "--export", fmt.Sprintf("ExportNotes=%t", options.ExportNotes))
	args = append(args, "--export", fmt.Sprintf("ExportNotesPages=%t", options.ExportNotesPages))
	args = append(args, "--export", fmt.Sprintf("ExportOnlyNotesPages=%t", options.ExportOnlyNotesPages))
	args = append(args, "--export", fmt.Sprintf("ExportNotesInMargin=%t", options.ExportNotesInMargin))
	args = append(args, "--export", fmt.Sprintf("ConvertOOoTargetToPDFTarget=%t", options.ConvertOooTargetToPdfTarget))
	args = append(args, "--export", fmt.Sprintf("ExportLinksRelativeFsys=%t", options.ExportLinksRelativeFsys))
	args = append(args, "--export", fmt.Sprintf("ExportHiddenSlides=%t", options.ExportHiddenSlides))
	args = append(args, "--export", fmt.Sprintf("IsSkipEmptyPages=%t", options.SkipEmptyPages))
	args = append(args, "--export", fmt.Sprintf("IsAddStream=%t", options.AddOriginalDocumentAsStream))
	args = append(args, "--export", fmt.Sprintf("SinglePageSheets=%t", options.SinglePageSheets))
	args = append(args, "--export", fmt.Sprintf("InitialView=%d", options.InitialView))
	args = append(args, "--export", fmt.Sprintf("InitialPage=%d", options.InitialPage))
	args = append(args, "--export", fmt.Sprintf("Magnification=%d", options.Magnification))
	args = append(args, "--export", fmt.Sprintf("Zoom=%d", options.Zoom))
	args = append(args, "--export", fmt.Sprintf("PageLayout=%d", options.PageLayout))
	args = append(args, "--export", fmt.Sprintf("FirstPageOnLeft=%t", options.FirstPageOnLeft))
	args = append(args, "--export", fmt.Sprintf("ResizeWindowToInitialPage=%t", options.ResizeWindowToInitialPage))
	args = append(args, "--export", fmt.Sprintf("CenterWindow=%t", options.CenterWindow))
	args = append(args, "--export", fmt.Sprintf("OpenInFullScreenMode=%t", options.OpenInFullScreenMode))
	args = append(args, "--export", fmt.Sprintf("DisplayPDFDocumentTitle=%t", options.DisplayPDFDocumentTitle))
	args = append(args, "--export", fmt.Sprintf("HideViewerMenubar=%t", options.HideViewerMenubar))
	args = append(args, "--export", fmt.Sprintf("HideViewerToolbar=%t", options.HideViewerToolbar))
	args = append(args, "--export", fmt.Sprintf("HideViewerWindowControls=%t", options.HideViewerWindowControls))
	args = append(args, "--export", fmt.Sprintf("UseTransitionEffects=%t", options.UseTransitionEffects))
	args = append(args, "--export", fmt.Sprintf("OpenBookmarkLevels=%d", options.OpenBookmarkLevels))
	args = append(args, "--export", fmt.Sprintf("UseLosslessCompression=%t", options.LosslessImageCompression))
	args = append(args, "--export", fmt.Sprintf("Quality=%d", options.Quality))
	args = append(args, "--export", fmt.Sprintf("ReduceImageResolution=%t", options.ReduceImageResolution))
	args = append(args, "--export", fmt.Sprintf("MaxImageResolution=%d", options.MaxImageResolution))

	if options.NativeWatermarkText != "" {
		args = append(args, "--export", fmt.Sprintf("Watermark=%s", options.NativeWatermarkText))
	}

	if options.NativeWatermarkColor != 0 {
		args = append(args, "--export", fmt.Sprintf("WatermarkColor=%d", options.NativeWatermarkColor))
	}

	if options.NativeWatermarkFontHeight > 0 {
		args = append(args, "--export", fmt.Sprintf("WatermarkFontHeight=%d", options.NativeWatermarkFontHeight))
	}

	if options.NativeWatermarkRotateAngle != 0 {
		args = append(args, "--export", fmt.Sprintf("WatermarkRotateAngle=%d", options.NativeWatermarkRotateAngle))
	}

	if options.NativeWatermarkFontName != "" && options.NativeWatermarkFontName != "Helvetica" {
		args = append(args, "--export", fmt.Sprintf("WatermarkFontName=%s", options.NativeWatermarkFontName))
	}

	if options.NativeTiledWatermarkText != "" {
		args = append(args, "--export", fmt.Sprintf("TiledWatermark=%s", options.NativeTiledWatermarkText))
	}

	switch options.PdfFormats.PdfA {
	case "":
	case gotenberg.PdfA1b:
		args = append(args, "--export", "SelectPdfVersion=1", "--export", "EmbedStandardFonts=true")
	case gotenberg.PdfA2b:
		args = append(args, "--export", "SelectPdfVersion=2", "--export", "EmbedStandardFonts=true")
	case gotenberg.PdfA3b:
		args = append(args, "--export", "SelectPdfVersion=3", "--export", "EmbedStandardFonts=true")
	default:
		return ErrInvalidPdfFormats
	}

	if options.PdfFormats.PdfUa {
		args = append(
			args,
			"--export", "PDFUACompliance=true",
			"--export", "UseTaggedPDF=true",
			"--export", "EnableTextAccessForAccessibilityTools=true",
			"--export", "EmbedStandardFonts=true",
		)
	} else {
		args = append(
			args,
			"--export", "PDFUACompliance=false",
			"--export", "UseTaggedPDF=false",
			"--export", "EnableTextAccessForAccessibilityTools=false",
		)
	}

	args = append(args, "--output", outputPath, inputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, p.arguments.unoBinPath, args...)
	if err != nil {
		return fmt.Errorf("create uno command: %w", err)
	}

	logger.DebugContext(ctx, fmt.Sprintf("print to PDF with: %+v", options))

	exitCode, err := cmd.Exec()
	if err == nil {
		return nil
	}

	// LibreOffice's errors are not explicit.
	// For instance, exit code 5 may be explained by a malformed page range
	// but also by a not required password.

	// We may want to retry in case of a core-dumped event.
	// See https://github.com/gotenberg/gotenberg/issues/639.
	if strings.Contains(err.Error(), "core dumped") {
		return ErrCoreDumped
	}

	if exitCode == 5 {
		// Potentially malformed page ranges or password not required.
		return ErrUnoException
	}
	if exitCode == 6 {
		// Password potentially required or invalid.
		return ErrRuntimeException
	}

	return fmt.Errorf("convert to PDF: %w", err)
}

// Interface guards.
var (
	_ gotenberg.Process = (*libreOfficeProcess)(nil)
	_ libreOffice       = (*libreOfficeProcess)(nil)
)
