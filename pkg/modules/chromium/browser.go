package chromium

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
)

type browser interface {
	gotenberg.Process
	pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error
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
	allowList         *regexp.Regexp
	denyList          *regexp.Regexp
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
		fs:         gotenberg.NewFileSystem(),
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

	// We have to keep the context around, as we need it to create a new tabs
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
		return errors.New("browser is already stopped")
	}

	// Always remove the user profile directory created by Chromium.
	copyUserProfileDirPath := b.userProfileDirPath
	defer func(userProfileDirPath string) {
		go func() {
			// FIXME: Chromium seems to recreate the user profile directory
			//  right after its deletion if we do not wait a certain amount
			//  of time before re-deleting it.
			<-time.After(10 * time.Second)

			err := os.RemoveAll(userProfileDirPath)
			if err != nil {
				logger.Error(fmt.Sprintf("remove Chromium's user profile directory: %s", err))
			}

			logger.Debug(fmt.Sprintf("'%s' Chromium's user profile directory removed", userProfileDirPath))
		}()
	}(copyUserProfileDirPath)

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

	taskCtx, cancel := chromedp.NewContext(b.ctx)
	defer cancel()

	err := chromedp.Run(taskCtx, chromedp.Navigate("about:blank"))
	if err != nil {
		logger.Error(fmt.Sprintf("browser health check failed: %s", err))
		return false
	}

	return true
}

func (b *chromiumBrowser) pdf(ctx context.Context, logger *zap.Logger, url, outputPath string, options Options) error {
	if !b.isStarted.Load() {
		return errors.New("browser not started, cannot handle PDF conversion")
	}

	// We validate the "main" URL against our allow / deny lists.
	if !b.arguments.allowList.MatchString(url) {
		return fmt.Errorf("'%s' does not match the expression from the allowed list: %w", url, ErrUrlNotAuthorized)
	}

	if b.arguments.denyList.String() != "" && b.arguments.denyList.MatchString(url) {
		return fmt.Errorf("'%s' matches the expression from the denied list: %w", url, ErrUrlNotAuthorized)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return errors.New("context has no deadline")
	}

	b.ctxMu.RLock()
	defer b.ctxMu.RUnlock()

	timeoutCtx, timeoutCancel := context.WithTimeout(b.ctx, time.Until(deadline))
	defer timeoutCancel()

	taskCtx, taskCancel := chromedp.NewContext(timeoutCtx)
	defer taskCancel()

	// We validate all others requests against our allow / deny lists.
	// If a request does not pass the validation, we make it fail.
	listenForEventRequestPaused(taskCtx, logger, b.arguments.allowList, b.arguments.denyList)

	var (
		consoleExceptions   error
		consoleExceptionsMu sync.RWMutex
	)

	// See https://github.com/gotenberg/gotenberg/issues/262.
	if options.FailOnConsoleExceptions && !b.arguments.disableJavaScript {
		listenForEventExceptionThrown(taskCtx, logger, &consoleExceptions, &consoleExceptionsMu)
	}

	tasks := chromedp.Tasks{
		network.Enable(),
		fetch.Enable(),
		runtime.Enable(),
		disableJavaScriptActionFunc(logger, b.arguments.disableJavaScript),
		extraHttpHeadersActionFunc(logger, options.ExtraHttpHeaders),
		navigateActionFunc(logger, url),
		hideDefaultWhiteBackgroundActionFunc(logger, options.OmitBackground, options.PrintBackground),
		forceExactColorsActionFunc(),
		emulateMediaTypeActionFunc(logger, options.EmulatedMediaType),
		waitDelayBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitDelay),
		waitForExpressionBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitForExpression),
		printToPdfActionFunc(logger, outputPath, options),
	}

	err := chromedp.Run(taskCtx, tasks...)
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

		return fmt.Errorf("print to PDF: %w", err)
	}

	// See https://github.com/gotenberg/gotenberg/issues/262.
	consoleExceptionsMu.RLock()
	defer consoleExceptionsMu.RUnlock()

	if consoleExceptions != nil {
		return fmt.Errorf("%v: %w", consoleExceptions, ErrConsoleExceptions)
	}

	return nil
}

// Interface guards.
var (
	_ gotenberg.Process = (*chromiumBrowser)(nil)
	_ browser           = (*chromiumBrowser)(nil)
)
