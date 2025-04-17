package chromium

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/dlclark/regexp2"
	"github.com/shirou/gopsutil/v4/process"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

type browser interface {
	gotenberg.Process
	pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error
	screenshot(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error
}

type browserArguments struct {
	// Executor args.
	binPath                  string
	incognito                bool
	allowInsecureLocalhost   bool
	ignoreCertificateErrors  bool
	disableWebSecurity       bool
	allowFileAccessFromFiles bool
	hostResolverRules        string
	proxyServer              string
	wsUrlReadTimeout         time.Duration

	// Tasks specific.
	allowList         *regexp2.Regexp
	denyList          *regexp2.Regexp
	clearCache        bool
	clearCookies      bool
	disableJavaScript bool
}

type chromiumBrowser struct {
	initialCtx         context.Context
	ctx                context.Context
	cancelFunc         context.CancelFunc
	userProfileDirPath string
	ctxMu              sync.RWMutex
	isStarted          atomic.Bool

	arguments browserArguments
	fs        *gotenberg.FileSystem
}

func newChromiumBrowser(arguments browserArguments) browser {
	b := &chromiumBrowser{
		initialCtx: context.Background(),
		arguments:  arguments,
		fs:         gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll)),
	}
	b.isStarted.Store(false)

	return b
}

func (b *chromiumBrowser) Start(logger *zap.Logger) error {
	if b.isStarted.Load() {
		return errors.New("browser is already started")
	}

	debug := &debugLogger{logger: logger}
	b.userProfileDirPath = b.fs.NewDirPath()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.CombinedOutput(debug),
		chromedp.ExecPath(b.arguments.binPath),
		chromedp.NoSandbox,
		// See:
		// https://github.com/gotenberg/gotenberg/issues/327
		// https://github.com/chromedp/chromedp/issues/904
		chromedp.DisableGPU,
		// See:
		// https://github.com/puppeteer/puppeteer/issues/661
		// https://github.com/puppeteer/puppeteer/issues/2410
		chromedp.Flag("font-render-hinting", "none"),
		chromedp.UserDataDir(b.userProfileDirPath),
		// See https://github.com/gotenberg/gotenberg/issues/831.
		chromedp.Flag("disable-pdf-tagging", true),
		// See https://github.com/gotenberg/gotenberg/issues/1177.
		chromedp.Flag("no-zygote", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	if b.arguments.incognito {
		opts = append(opts, chromedp.Flag("incognito", b.arguments.incognito))
	}

	if b.arguments.allowInsecureLocalhost {
		// See https://github.com/gotenberg/gotenberg/issues/488.
		opts = append(opts, chromedp.Flag("allow-insecure-localhost", true))
	}

	if b.arguments.ignoreCertificateErrors {
		opts = append(opts, chromedp.IgnoreCertErrors)
	}

	if b.arguments.disableWebSecurity {
		opts = append(opts, chromedp.Flag("disable-web-security", true))
	}

	if b.arguments.allowFileAccessFromFiles {
		// See https://github.com/gotenberg/gotenberg/issues/356.
		opts = append(opts, chromedp.Flag("allow-file-access-from-files", true))
	}

	if b.arguments.hostResolverRules != "" {
		// See https://github.com/gotenberg/gotenberg/issues/488.
		opts = append(opts, chromedp.Flag("host-resolver-rules", b.arguments.hostResolverRules))
	}

	if b.arguments.proxyServer != "" {
		// See https://github.com/gotenberg/gotenberg/issues/376.
		opts = append(opts, chromedp.ProxyServer(b.arguments.proxyServer))
	}

	// See https://github.com/gotenberg/gotenberg/issues/524.
	opts = append(opts, chromedp.WSURLReadTimeout(b.arguments.wsUrlReadTimeout))

	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(b.initialCtx, opts...)
	ctx, cancel := chromedp.NewContext(allocatorCtx, chromedp.WithDebugf(debug.Printf))

	err := chromedp.Run(ctx)
	if err != nil {
		cancel()
		allocatorCancel()
		return fmt.Errorf("run exec allocator: %w", err)
	}

	b.ctxMu.Lock()
	defer b.ctxMu.Unlock()

	// We have to keep the context around, as we need it to create new tabs
	// later.
	b.ctx = ctx
	b.cancelFunc = func() {
		cancel()
		allocatorCancel()
	}
	b.isStarted.Store(true)

	return nil
}

func (b *chromiumBrowser) Stop(logger *zap.Logger) error {
	if !b.isStarted.Load() {
		// No big deal? Like calling cancel twice.
		return nil
	}

	// Always remove the user profile directory created by Chromium.
	copyUserProfileDirPath := b.userProfileDirPath
	expirationTime := time.Now()
	defer func(userProfileDirPath string, expirationTime time.Time) {
		// See:
		// https://github.com/SeleniumHQ/docker-selenium/blob/7216d060d86872afe853ccda62db0dfab5118dc7/NodeChrome/chrome-cleanup.sh
		// https://github.com/SeleniumHQ/docker-selenium/blob/7216d060d86872afe853ccda62db0dfab5118dc7/NodeChromium/chrome-cleanup.sh

		// Clean up stuck processes.
		ps, err := process.Processes()
		if err != nil {
			logger.Error(fmt.Sprintf("list processes: %v", err))
		} else {
			for _, p := range ps {
				func() {
					cmdline, err := p.Cmdline()
					if err != nil {
						return
					}

					if !strings.Contains(cmdline, "chromium/chromium") && !strings.Contains(cmdline, "chrome/chrome") {
						return
					}

					killCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
					defer cancel()

					err = p.KillWithContext(killCtx)
					if err != nil {
						logger.Error(fmt.Sprintf("kill process: %v", err))
					} else {
						logger.Debug(fmt.Sprintf("Chromium process %d killed", p.Pid))
					}
				}()
			}
		}

		go func() {
			// FIXME: Chromium seems to recreate the user profile directory
			//  right after its deletion if we do not wait a certain amount
			//  of time before deleting it.
			<-time.After(10 * time.Second)

			err = os.RemoveAll(userProfileDirPath)
			if err != nil {
				logger.Error(fmt.Sprintf("remove Chromium's user profile directory: %s", err))
			} else {
				logger.Debug(fmt.Sprintf("'%s' Chromium's user profile directory removed", userProfileDirPath))
			}

			// Also, remove Chromium-specific files in the temporary directory.
			err = gotenberg.GarbageCollect(logger, os.TempDir(), []string{".org.chromium.Chromium", ".com.google.Chrome"}, expirationTime)
			if err != nil {
				logger.Error(err.Error())
			}
		}()
	}(copyUserProfileDirPath, expirationTime)

	b.ctxMu.Lock()
	defer b.ctxMu.Unlock()

	b.cancelFunc()
	b.ctx = nil
	b.userProfileDirPath = ""
	b.isStarted.Store(false)

	return nil
}

func (b *chromiumBrowser) Healthy(logger *zap.Logger) bool {
	// Good to know: the supervisor does not call this method if no first start
	// or if the process is restarting.

	if !b.isStarted.Load() {
		// Non-started browser but not restarting?
		return false
	}

	b.ctxMu.RLock()
	defer b.ctxMu.RUnlock()

	timeoutCtx, timeoutCancel := context.WithTimeout(b.ctx, time.Duration(10)*time.Second)
	defer timeoutCancel()

	taskCtx, taskCancel := chromedp.NewContext(timeoutCtx)
	defer taskCancel()

	err := chromedp.Run(taskCtx, chromedp.Navigate("about:blank"))
	if err != nil {
		logger.Error(fmt.Sprintf("browser health check failed: %s", err))
		return false
	}

	return true
}

func (b *chromiumBrowser) pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options PdfOptions) error {
	// Note: no error wrapping because it leaks on errors we want to display to
	// the end user.
	return b.do(ctx, logger, url, options.Options, chromedp.Tasks{
		network.Enable(),
		fetch.Enable(),
		runtime.Enable(),
		clearCacheActionFunc(logger, b.arguments.clearCache),
		clearCookiesActionFunc(logger, b.arguments.clearCookies),
		disableJavaScriptActionFunc(logger, b.arguments.disableJavaScript),
		setCookiesActionFunc(logger, options.Cookies),
		userAgentOverride(logger, options.UserAgent),
		navigateActionFunc(logger, url, options.SkipNetworkIdleEvent),
		hideDefaultWhiteBackgroundActionFunc(logger, options.OmitBackground, options.PrintBackground),
		forceExactColorsActionFunc(logger, options.PrintBackground),
		emulateMediaTypeActionFunc(logger, options.EmulatedMediaType),
		waitDelayBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitDelay),
		waitForExpressionBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitForExpression),
		// PDF specific.
		printToPdfActionFunc(logger, outputPath, options),
	})
}

func (b *chromiumBrowser) screenshot(ctx context.Context, logger *zap.Logger, url, outputPath string, options ScreenshotOptions) error {
	// Note: no error wrapping because it leaks on errors we want to display to
	// the end user.
	return b.do(ctx, logger, url, options.Options, chromedp.Tasks{
		network.Enable(),
		fetch.Enable(),
		runtime.Enable(),
		clearCacheActionFunc(logger, b.arguments.clearCache),
		clearCookiesActionFunc(logger, b.arguments.clearCookies),
		disableJavaScriptActionFunc(logger, b.arguments.disableJavaScript),
		setCookiesActionFunc(logger, options.Cookies),
		userAgentOverride(logger, options.UserAgent),
		navigateActionFunc(logger, url, options.SkipNetworkIdleEvent),
		hideDefaultWhiteBackgroundActionFunc(logger, options.OmitBackground, true),
		forceExactColorsActionFunc(logger, true),
		emulateMediaTypeActionFunc(logger, options.EmulatedMediaType),
		waitDelayBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitDelay),
		waitForExpressionBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitForExpression),
		// Screenshot specific.
		setDeviceMetricsOverride(logger, options.Width, options.Height),
		captureScreenshotActionFunc(logger, outputPath, options),
	})
}

func (b *chromiumBrowser) do(ctx context.Context, logger *zap.Logger, url string, options Options, tasks chromedp.Tasks) error {
	if !b.isStarted.Load() {
		return errors.New("browser not started, cannot handle tasks")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return errors.New("context has no deadline")
	}

	// We validate the "main" URL against our allowed / deny lists.
	err := gotenberg.FilterDeadline(b.arguments.allowList, b.arguments.denyList, url, deadline)
	if err != nil {
		return fmt.Errorf("filter URL: %w", err)
	}

	b.ctxMu.RLock()
	defer b.ctxMu.RUnlock()

	timeoutCtx, timeoutCancel := context.WithTimeout(b.ctx, time.Until(deadline))
	defer timeoutCancel()

	taskCtx, taskCancel := chromedp.NewContext(timeoutCtx)
	defer taskCancel()

	// We validate all other requests against our allowed / deny lists.
	// If a request does not pass the validation, we make it fail. It also set
	// the extra HTTP headers, if any.
	// See https://github.com/gotenberg/gotenberg/issues/1011.
	listenForEventRequestPaused(taskCtx, logger, eventRequestPausedOptions{
		allowList:        b.arguments.allowList,
		denyList:         b.arguments.denyList,
		extraHttpHeaders: options.ExtraHttpHeaders,
	})

	var (
		invalidHttpStatusCode           error
		invalidHttpStatusCodeMu         sync.RWMutex
		invalidResourceHttpStatusCode   error
		invalidResourceHttpStatusCodeMu sync.RWMutex
	)

	// See:
	// https://github.com/gotenberg/gotenberg/issues/613.
	// https://github.com/gotenberg/gotenberg/issues/1021.
	if len(options.FailOnHttpStatusCodes) != 0 || len(options.FailOnResourceHttpStatusCodes) != 0 {
		listenForEventResponseReceived(taskCtx, logger, eventResponseReceivedOptions{
			mainPageUrl:                     url,
			failOnHttpStatusCodes:           options.FailOnHttpStatusCodes,
			invalidHttpStatusCode:           &invalidHttpStatusCode,
			invalidHttpStatusCodeMu:         &invalidHttpStatusCodeMu,
			failOnResourceOnHttpStatusCode:  options.FailOnResourceHttpStatusCodes,
			invalidResourceHttpStatusCode:   &invalidResourceHttpStatusCode,
			invalidResourceHttpStatusCodeMu: &invalidResourceHttpStatusCodeMu,
		})
	}

	var (
		consoleExceptions   error
		consoleExceptionsMu sync.RWMutex
	)

	// See https://github.com/gotenberg/gotenberg/issues/262.
	if options.FailOnConsoleExceptions && !b.arguments.disableJavaScript {
		listenForEventExceptionThrown(taskCtx, logger, &consoleExceptions, &consoleExceptionsMu)
	}

	var (
		loadingFailed           error
		loadingFailedMu         sync.RWMutex
		resourceLoadingFailed   error
		resourceLoadingFailedMu sync.RWMutex
	)

	// See:
	// https://github.com/gotenberg/gotenberg/issues/913.
	// https://github.com/gotenberg/gotenberg/issues/959.
	// https://github.com/gotenberg/gotenberg/issues/1021.
	listenForEventLoadingFailed(taskCtx, logger, eventLoadingFailedOptions{
		loadingFailed:           &loadingFailed,
		loadingFailedMu:         &loadingFailedMu,
		resourceLoadingFailed:   &resourceLoadingFailed,
		resourceLoadingFailedMu: &resourceLoadingFailedMu,
	})

	err = chromedp.Run(taskCtx, tasks...)
	if err != nil {
		errMessage := err.Error()

		if strings.Contains(errMessage, "Show invalid printer settings error (-32000)") || strings.Contains(errMessage, "content area is empty (-32602)") {
			return ErrInvalidPrinterSettings
		}

		if strings.Contains(errMessage, "Page range syntax error") {
			return ErrPageRangesSyntaxError
		}

		if strings.Contains(errMessage, "rpcc: message too large") {
			return ErrRpccMessageTooLarge
		}

		return fmt.Errorf("handle tasks: %w", err)
	}

	// See https://github.com/gotenberg/gotenberg/issues/613.
	invalidHttpStatusCodeMu.RLock()
	defer invalidHttpStatusCodeMu.RUnlock()

	if invalidHttpStatusCode != nil {
		return fmt.Errorf("%v: %w", invalidHttpStatusCode, ErrInvalidHttpStatusCode)
	}

	// See https://github.com/gotenberg/gotenberg/issues/1021.
	invalidResourceHttpStatusCodeMu.RLock()
	defer invalidResourceHttpStatusCodeMu.RUnlock()

	if invalidResourceHttpStatusCode != nil {
		return fmt.Errorf("%v: %w", invalidResourceHttpStatusCode, ErrInvalidResourceHttpStatusCode)
	}

	// See https://github.com/gotenberg/gotenberg/issues/262.
	consoleExceptionsMu.RLock()
	defer consoleExceptionsMu.RUnlock()

	if consoleExceptions != nil {
		return fmt.Errorf("%v: %w", consoleExceptions, ErrConsoleExceptions)
	}

	// See:
	// https://github.com/gotenberg/gotenberg/issues/913.
	// https://github.com/gotenberg/gotenberg/issues/959.
	loadingFailedMu.RLock()
	defer loadingFailedMu.RUnlock()

	if loadingFailed != nil {
		return fmt.Errorf("%v: %w", loadingFailed, ErrLoadingFailed)
	}

	// See https://github.com/gotenberg/gotenberg/issues/1021.
	if options.FailOnResourceLoadingFailed {
		if resourceLoadingFailed != nil {
			return fmt.Errorf("%v: %w", resourceLoadingFailed, ErrResourceLoadingFailed)
		}
	}

	return nil
}

// Interface guards.
var (
	_ gotenberg.Process = (*chromiumBrowser)(nil)
	_ browser           = (*chromiumBrowser)(nil)
)
