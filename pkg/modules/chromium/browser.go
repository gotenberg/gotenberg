package chromium

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cdprotobrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/dlclark/regexp2/v2"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// chromiumDisableFeatures is the value of Chromium's --disable-features
// switch.
//
// It restates the "site-per-process,Translate,BlinkGenPropertyTrees" default
// from chromedp.DefaultExecAllocatorOptions (chromedp v0.14.2) on purpose:
// chromedp.Flag keys its flags by switch name, so a second --disable-features
// replaces chromedp's value instead of merging with it. Revisit this list when
// bumping chromedp.
//
// WebUIOmniboxPopup and WebUIOmniboxAimPopup became enabled by default in
// Chromium 151.0.7922.132. Their presenters build the address-bar popup WebUI
// at browser start, headless included, which leaves a renderer process holding
// ~85 MB of anonymous memory for a UI a PDF service can never show. Chromium
// silently ignores feature names it does not know, so both stay harmless on
// older builds (they exist but default to disabled on the Chromium pinned for
// ppc64el) and once upstream eventually removes them.
// See https://github.com/gotenberg/gotenberg/issues/1656.
//
// OptimizationHints, OptimizationTargetPrediction and
// NetworkTimeServiceQuerying make Chromium call Google services at every
// start, which helps no conversion and logs blocked CONNECTs in locked-down
// deployments.
// See https://github.com/gotenberg/gotenberg/issues/1706.
const chromiumDisableFeatures = "site-per-process,Translate,BlinkGenPropertyTrees,WebUIOmniboxPopup,WebUIOmniboxAimPopup,OptimizationHints,OptimizationTargetPrediction,NetworkTimeServiceQuerying"

// chromiumGaiaUrl replaces Chromium's Google sign-in URL with a closed
// loopback port. No feature switch stops the ListAccounts calls Chromium
// makes at every start, so they now fail locally instead of reaching Google.
// See https://github.com/gotenberg/gotenberg/issues/1706.
const chromiumGaiaUrl = "http://127.0.0.1:9"

type browser interface {
	gotenberg.Process
	pdf(ctx context.Context, logger *slog.Logger, url, outputPath string, options PdfOptions, aggregate *networkAggregate) error
	screenshot(ctx context.Context, logger *slog.Logger, url, outputPath string, options ScreenshotOptions, aggregate *networkAggregate) error
}

type browserArguments struct {
	// Executor args.
	binPath                  string
	allowInsecureLocalhost   bool
	ignoreCertificateErrors  bool
	disableWebSecurity       bool
	allowFileAccessFromFiles bool
	hostResolverRules        string
	proxyServer              string
	enableEnvironmentProxy   bool
	wsUrlReadTimeout         time.Duration
	hyphenDataDirPath        string

	// Tasks specific.
	allowList            []*regexp2.Regexp
	denyList             []*regexp2.Regexp
	denyPrivateIPs       bool
	denyPublicIPs        bool
	sharedBrowserContext bool
	clearCache           bool
	clearCookies         bool
	clearStorage         bool
	disableJavaScript    bool
}

// restrictsNetwork reports whether the outbound policy can reject a network
// URL. The default deny-list only rejects file:// URLs, so it does not count.
func (a browserArguments) restrictsNetwork() bool {
	if a.denyPrivateIPs || a.denyPublicIPs || len(a.allowList) > 0 {
		return true
	}
	for _, pattern := range a.denyList {
		if pattern.String() != defaultDenyListPattern {
			return true
		}
	}
	return false
}

type chromiumBrowser struct {
	initialCtx         context.Context
	ctx                context.Context
	cancelFunc         context.CancelFunc
	userProfileDirPath string
	ctxMu              sync.RWMutex
	isStarted          atomic.Bool
	// lifecycleMu serializes Start and Stop, so that a shutdown never
	// interleaves with a restart on the shared pinning proxy.
	// See https://github.com/gotenberg/gotenberg/issues/1599.
	lifecycleMu sync.Mutex

	arguments    browserArguments
	fs           *gotenberg.FileSystem
	pinningProxy *pinningProxy
}

func newChromiumBrowser(arguments browserArguments) browser {
	b := &chromiumBrowser{
		initialCtx:   context.Background(),
		arguments:    arguments,
		fs:           gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll)),
		pinningProxy: newPinningProxy(arguments.allowList, arguments.denyList, arguments.denyPrivateIPs, arguments.denyPublicIPs, arguments.enableEnvironmentProxy),
	}
	b.isStarted.Store(false)

	return b
}

func (b *chromiumBrowser) Start(ctx context.Context, logger *slog.Logger) (err error) {
	b.lifecycleMu.Lock()
	defer b.lifecycleMu.Unlock()

	err = ctx.Err()
	if err != nil {
		return fmt.Errorf("start browser: %w", err)
	}

	if b.isStarted.Load() {
		return errors.New("browser is already started")
	}

	debug := &debugLogger{logger: logger}
	b.userProfileDirPath = b.fs.NewDirPath()

	// See https://github.com/gotenberg/gotenberg/issues/1293.
	err = os.MkdirAll(b.userProfileDirPath, 0o755)
	if err != nil {
		return fmt.Errorf("could not create user profile directory: %w", err)
	}

	// Stop only cleans up after a successful start, and every supervisor retry
	// creates a new directory.
	defer func() {
		if err == nil {
			return
		}

		rmErr := os.RemoveAll(b.userProfileDirPath)
		if rmErr != nil {
			logger.ErrorContext(context.Background(), fmt.Sprintf("remove Chromium's user profile directory after a failed start: %s", rmErr))
		}
		b.userProfileDirPath = ""
	}()

	err = os.Symlink(b.arguments.hyphenDataDirPath, fmt.Sprintf("%s/hyphen-data", b.userProfileDirPath))
	if err != nil {
		return fmt.Errorf("create symlink to hyphen-data directory: %w", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.CombinedOutput(debug),
		chromedp.ExecPath(b.arguments.binPath),
		chromedp.NoSandbox,
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
		// See https://github.com/gotenberg/gotenberg/issues/1293.
		chromedp.Flag("disable-component-update", false),
		// See https://github.com/gotenberg/gotenberg/issues/1656.
		chromedp.Flag("disable-features", chromiumDisableFeatures),
		// See https://github.com/gotenberg/gotenberg/issues/1706.
		chromedp.Flag("gaia-url", chromiumGaiaUrl),
	)

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

	// Default: route Chromium through the internal pinning proxy so that
	// Chromium never performs its own DNS lookup for the navigation URL
	// or any sub-resource. The proxy resolves and validates each URL
	// once per request and dials the pinned IP, closing the DNS
	// rebinding window between Gotenberg's validation and Chromium's
	// connect.
	//
	// Skip when the operator has configured their own egress proxy or
	// custom host-resolver mappings: those deployments take
	// responsibility for outbound safety themselves and routing through
	// an internal proxy would override their configuration.
	if b.arguments.proxyServer == "" && b.arguments.hostResolverRules == "" {
		err = b.pinningProxy.Start(logger)
		if err != nil {
			return fmt.Errorf("start pinning proxy: %w", err)
		}
		opts = append(opts, chromedp.ProxyServer(b.pinningProxy.URL()))

		if b.arguments.restrictsNetwork() {
			// Chromium implicitly bypasses the proxy for loopback and
			// link-local destinations. A WebSocket handshake is never surfaced
			// as a fetch.EventRequestPaused, so listenForEventRequestPaused
			// cannot filter it; the pinning proxy is the only layer that sees
			// it. Left alone, a page could open a WebSocket to 127.0.0.1, ::1,
			// localhost, or the link-local cloud metadata endpoint
			// (169.254.169.254) and reach it unfiltered. "<-loopback>" removes
			// the implicit bypass so those handshakes also traverse the
			// pinning proxy and go through [gotenberg.DecideOutbound] like
			// every other request.
			//
			// Gated on a policy that can reject a network URL: without one,
			// loopback is not restricted, and routing it through the proxy
			// would only turn a refused loopback sub-resource into a 502
			// response instead of a loading failure.
			opts = append(opts, chromedp.Flag("proxy-bypass-list", "<-loopback>"))
		}
	}

	// See https://github.com/gotenberg/gotenberg/issues/524.
	opts = append(opts, chromedp.WSURLReadTimeout(b.arguments.wsUrlReadTimeout))

	// The browser outlives the start, so its context derives from initialCtx.
	// Until the start succeeds, ctx may still abort it.
	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(b.initialCtx, opts...)
	browserCtx, cancel := chromedp.NewContext(allocatorCtx, chromedp.WithDebugf(debug.Printf))
	abortStart := context.AfterFunc(ctx, func() {
		cancel()
		allocatorCancel()
	})

	err = chromedp.Run(browserCtx)
	if err == nil && !abortStart() {
		// ctx was done right as Chromium came up: the browser is going down.
		err = ctx.Err()
	}
	if err != nil {
		cancel()
		allocatorCancel()
		// The pinning proxy started before chromedp; tear it down so a
		// supervisor retry can re-bind. Stop is a no-op when the proxy
		// was never started (operator-configured --chromium-proxy-server
		// or --chromium-host-resolver-rules).
		// See https://github.com/gotenberg/gotenberg/issues/1559.
		stopErr := b.pinningProxy.Stop(logger)
		if stopErr != nil {
			logger.ErrorContext(context.Background(), fmt.Sprintf("stop pinning proxy after failed start: %s", stopErr))
		}
		return fmt.Errorf("run exec allocator: %w; if Chromium is slow to start, raise --chromium-start-timeout (currently %s)", err, b.arguments.wsUrlReadTimeout)
	}

	b.ctxMu.Lock()
	defer b.ctxMu.Unlock()

	// We have to keep the context around, as we need it to create new tabs
	// later.
	b.ctx = browserCtx
	b.cancelFunc = func() {
		cancel()
		allocatorCancel()
	}
	b.isStarted.Store(true)

	return nil
}

func (b *chromiumBrowser) Stop(logger *slog.Logger) error {
	b.lifecycleMu.Lock()
	defer b.lifecycleMu.Unlock()

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
			logger.ErrorContext(context.Background(), fmt.Sprintf("list processes: %v", err))
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
						logger.ErrorContext(context.Background(), fmt.Sprintf("kill process: %v", err))
					} else {
						logger.DebugContext(context.Background(), fmt.Sprintf("Chromium process %d killed", p.Pid))
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
				logger.ErrorContext(context.Background(), fmt.Sprintf("remove Chromium's user profile directory: %s", err))
			} else {
				logger.DebugContext(context.Background(), fmt.Sprintf("'%s' Chromium's user profile directory removed", userProfileDirPath))
			}

			// Also, remove Chromium-specific files in the temporary directory.
			// Chromium names them org.chromium.Chromium.XXXXXX; older
			// releases prefixed a dot.
			err = gotenberg.GarbageCollect(context.Background(), logger, os.TempDir(), []string{"org.chromium.Chromium.*", ".org.chromium.Chromium.*", "com.google.Chrome.*", ".com.google.Chrome.*"}, expirationTime)
			if err != nil {
				logger.ErrorContext(context.Background(), err.Error())
			}
		}()
	}(copyUserProfileDirPath, expirationTime)

	b.ctxMu.Lock()
	defer b.ctxMu.Unlock()

	b.cancelFunc()
	b.ctx = nil
	b.userProfileDirPath = ""
	b.isStarted.Store(false)

	// Stop the pinning proxy after Chromium shutdown so that any
	// in-flight requests Chromium issues during teardown complete. The
	// Stop call is a no-op when the proxy was not started (operator
	// configured --chromium-proxy-server or --chromium-host-resolver-rules).
	err := b.pinningProxy.Stop(logger)
	if err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("stop pinning proxy: %s", err))
	}

	return nil
}

func (b *chromiumBrowser) Healthy(logger *slog.Logger) bool {
	// Good to know: the supervisor does not call this method if no first start
	// or if the process is restarting.

	if !b.isStarted.Load() {
		// Non-started browser but not restarting?
		return false
	}

	b.ctxMu.RLock()
	defer b.ctxMu.RUnlock()

	// Create a timeout based on the existing browser context (b.ctx).
	// IMPORTANT: We do NOT call chromedp.NewContext here.
	// We want to execute this against the main browser connection,
	// avoiding the creation of a new target (tab).
	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()

	// Check if the browser is responsive by asking for its version.
	// This involves a simple JSON payload roundtrip over the websocket.
	// See https://github.com/gotenberg/gotenberg/issues/1169.
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, _, _, _, _, err := cdprotobrowser.GetVersion().Do(ctx)
		return err
	}))
	if err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("browser health check failed: %s", err))
		return false
	}

	return true
}

func (b *chromiumBrowser) pdf(ctx context.Context, logger *slog.Logger, url, outputPath string, options PdfOptions, aggregate *networkAggregate) error {
	// Note: no error wrapping because it leaks on errors we want to display to
	// the end user.
	return b.do(ctx, logger, url, options.Options, aggregate, chromedp.Tasks{
		network.Enable(),
		fetch.Enable(),
		runtime.Enable(),
		clearCacheActionFunc(logger, b.arguments.clearCache),
		clearCookiesActionFunc(logger, b.arguments.clearCookies),
		clearStorageActionFunc(logger, b.arguments.clearStorage, url),
		disableJavaScriptActionFunc(logger, b.arguments.disableJavaScript),
		setCookiesActionFunc(logger, options.Cookies),
		userAgentOverride(logger, options.UserAgent),
		navigateActionFunc(logger, url, options.SkipNetworkIdleEvent, options.SkipNetworkAlmostIdleEvent),
		hideDefaultWhiteBackgroundActionFunc(logger, options.OmitBackground, options.PrintBackground),
		forceExactColorsActionFunc(logger, options.PrintBackground),
		emulateMediaTypeActionFunc(logger, options.EmulatedMediaType, options.EmulatedMediaFeatures),
		waitForExpressionBeforePrintActionFunc(logger, b.arguments.disableJavaScript, windowStatusExpression(options.WaitWindowStatus)),
		waitForExpressionBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitForExpression),
		waitForSelectorVisibleBeforePrintActionFunc(logger, options.WaitForSelector),
		waitDelayBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitDelay),
		// PDF specific.
		printToPdfActionFunc(ctx, logger, outputPath, options),
		// Teardown.
		page.Close(),
	})
}

func (b *chromiumBrowser) screenshot(ctx context.Context, logger *slog.Logger, url, outputPath string, options ScreenshotOptions, aggregate *networkAggregate) error {
	// Note: no error wrapping because it leaks on errors we want to display to
	// the end user.
	return b.do(ctx, logger, url, options.Options, aggregate, chromedp.Tasks{
		network.Enable(),
		fetch.Enable(),
		runtime.Enable(),
		clearCacheActionFunc(logger, b.arguments.clearCache),
		clearCookiesActionFunc(logger, b.arguments.clearCookies),
		clearStorageActionFunc(logger, b.arguments.clearStorage, url),
		disableJavaScriptActionFunc(logger, b.arguments.disableJavaScript),
		setCookiesActionFunc(logger, options.Cookies),
		userAgentOverride(logger, options.UserAgent),
		navigateActionFunc(logger, url, options.SkipNetworkIdleEvent, options.SkipNetworkAlmostIdleEvent),
		hideDefaultWhiteBackgroundActionFunc(logger, options.OmitBackground, true),
		forceExactColorsActionFunc(logger, true),
		emulateMediaTypeActionFunc(logger, options.EmulatedMediaType, options.EmulatedMediaFeatures),
		waitForExpressionBeforePrintActionFunc(logger, b.arguments.disableJavaScript, windowStatusExpression(options.WaitWindowStatus)),
		waitForExpressionBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitForExpression),
		waitForSelectorVisibleBeforePrintActionFunc(logger, options.WaitForSelector),
		waitDelayBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitDelay),
		// Screenshot specific.
		setDeviceMetricsOverride(logger, options.Width, options.Height, options.DeviceScaleFactor),
		captureScreenshotActionFunc(logger, outputPath, options),
		// Teardown.
		page.Close(),
	})
}

// newIsolatedTarget creates a browser context and opens a blank tab in it, so
// cookies, cache, and storage never leak to another conversion, even a
// concurrent one. The caller must dispose the browser context.
//
// chromedp.WithNewBrowserContext does not work here: it opens the tab without
// a new window, which headless Chromium rejects with "no browser is open", and
// it leaks the browser context on that failure.
func (b *chromiumBrowser) newIsolatedTarget(ctx context.Context, logger *slog.Logger) (cdp.BrowserContextID, target.ID, error) {
	c := chromedp.FromContext(b.ctx)
	if c == nil || c.Browser == nil {
		return "", "", errors.New("browser not allocated")
	}
	executor := cdp.WithExecutor(ctx, c.Browser)

	browserContextID, err := target.CreateBrowserContext().WithDisposeOnDetach(true).Do(executor)
	if err != nil {
		return "", "", fmt.Errorf("create browser context: %w", err)
	}

	targetID, err := target.CreateTarget("about:blank").
		WithBrowserContextID(browserContextID).
		WithNewWindow(true).
		Do(executor)
	if err != nil {
		b.disposeBrowserContext(logger, browserContextID)
		return "", "", fmt.Errorf("create target: %w", err)
	}

	return browserContextID, targetID, nil
}

// disposeBrowserContext disposes a browser context created by
// [chromiumBrowser.newIsolatedTarget]. It uses a fresh context, as the
// conversion's one may already be done.
func (b *chromiumBrowser) disposeBrowserContext(logger *slog.Logger, browserContextID cdp.BrowserContextID) {
	c := chromedp.FromContext(b.ctx)
	if c == nil || c.Browser == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := target.DisposeBrowserContext(browserContextID).Do(cdp.WithExecutor(ctx, c.Browser))
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("dispose browser context: %s", err))
	}
}

func (b *chromiumBrowser) do(ctx context.Context, logger *slog.Logger, url string, options Options, aggregate *networkAggregate, tasks chromedp.Tasks) error {
	if !b.isStarted.Load() {
		return errors.New("browser not started, cannot handle tasks")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return errors.New("context has no deadline")
	}

	// We validate the "main" URL against our allowed / deny lists, and
	// against the IP-based outbound URL guard. See [gotenberg.FilterOutboundURL].
	err := gotenberg.FilterOutboundURL(ctx, url, b.arguments.allowList, b.arguments.denyList, deadline,
		gotenberg.WithDenyPrivateIPs(b.arguments.denyPrivateIPs),
		gotenberg.WithDenyPublicIPs(b.arguments.denyPublicIPs),
	)
	if err != nil {
		return fmt.Errorf("filter URL: %w", err)
	}

	b.ctxMu.RLock()
	defer b.ctxMu.RUnlock()

	// The tab derives from the browser, not from the request, but must not
	// outlive it: the supervisor holds the request's slot until do returns.
	timeoutCtx, timeoutCancel := context.WithTimeout(b.ctx, time.Until(deadline))
	defer timeoutCancel()
	stopCancelOnRequest := context.AfterFunc(ctx, timeoutCancel)
	defer stopCancelOnRequest()

	var taskOpts []chromedp.ContextOption
	if !b.arguments.sharedBrowserContext {
		browserContextID, targetID, err := b.newIsolatedTarget(timeoutCtx, logger)
		if err != nil {
			return fmt.Errorf("create isolated browser context: %w", err)
		}
		// Deferred before taskCancel, so it runs after the tab is closed.
		defer b.disposeBrowserContext(logger, browserContextID)
		taskOpts = append(taskOpts, chromedp.WithTargetID(targetID))
	}

	taskCtx, taskCancel := chromedp.NewContext(timeoutCtx, taskOpts...)
	defer taskCancel()

	failures := new(conversionFailures)
	b.listen(taskCtx, taskCancel, logger, url, options, aggregate, failures)

	runErr := chromedp.Run(taskCtx, tasks...)

	// The tab is canceled when the request ends, so the run reports
	// context.Canceled even when the request ran out of time. Report the
	// request's own error, so that a deadline still maps to a 503 and a client
	// that went away to a 499.
	if runErr != nil && ctx.Err() != nil {
		runErr = fmt.Errorf("%w: %w", ctx.Err(), runErr)
	}

	return failures.err(runErr, b.ctx.Err() != nil, options.FailOnResourceLoadingFailed)
}

// listen registers the event listeners of one conversion on taskCtx. They
// record what fails the conversion in failures, and some cancel taskCtx so
// that the conversion stops at once.
func (b *chromiumBrowser) listen(taskCtx context.Context, taskCancel context.CancelFunc, logger *slog.Logger, url string, options Options, aggregate *networkAggregate, failures *conversionFailures) {
	// Accumulate per-conversion network activity for telemetry.
	listenForNetworkActivity(taskCtx, aggregate)

	// We validate all other requests against our allowed / deny lists.
	// If a request does not pass the validation, we make it fail. It also set
	// the extra HTTP headers, if any.
	// See https://github.com/gotenberg/gotenberg/issues/1011.
	listenForEventRequestPaused(taskCtx, logger, eventRequestPausedOptions{
		allowList:           b.arguments.allowList,
		denyList:            b.arguments.denyList,
		denyPrivateIPs:      b.arguments.denyPrivateIPs,
		denyPublicIPs:       b.arguments.denyPublicIPs,
		allowedFilePrefixes: options.AllowedFilePrefixes,
		extraHttpHeaders:    options.ExtraHttpHeaders,
	})

	// WebSocket handshakes never surface as fetch.EventRequestPaused, so
	// listenForEventRequestPaused above cannot filter them. Validate them
	// against the same allow / deny lists and IP-class policy.
	// See https://github.com/gotenberg/gotenberg/issues/1011.
	webSocketOptions := eventWebSocketCreatedOptions{
		allowList:      b.arguments.allowList,
		denyList:       b.arguments.denyList,
		denyPrivateIPs: b.arguments.denyPrivateIPs,
		denyPublicIPs:  b.arguments.denyPublicIPs,
		failures:       failures,
		cancel:         taskCancel,
	}
	if b.arguments.proxyServer == "" && b.arguments.hostResolverRules == "" {
		webSocketOptions.decideTunnel = b.pinningProxy.decideTunnel
	}

	listenForEventWebSocketCreated(taskCtx, logger, webSocketOptions)

	// See:
	// https://github.com/gotenberg/gotenberg/issues/613.
	// https://github.com/gotenberg/gotenberg/issues/1021.
	if len(options.FailOnHttpStatusCodes) != 0 || len(options.FailOnResourceHttpStatusCodes) != 0 {
		listenForEventResponseReceived(taskCtx, logger, eventResponseReceivedOptions{
			mainPageUrl:                     url,
			failOnHttpStatusCodes:           options.FailOnHttpStatusCodes,
			failOnResourceOnHttpStatusCode:  options.FailOnResourceHttpStatusCodes,
			ignoreResourceHttpStatusDomains: options.IgnoreResourceHttpStatusDomains,
			failures:                        failures,
			cancelOnMainPageError:           taskCancel,
		})
	}

	// See https://github.com/gotenberg/gotenberg/issues/262.
	if options.FailOnConsoleExceptions && !b.arguments.disableJavaScript {
		listenForEventExceptionThrown(taskCtx, logger, failures)
	}

	// See https://github.com/gotenberg/gotenberg/issues/1667.
	if len(options.FailOnConsoleMessageTypes) != 0 && !b.arguments.disableJavaScript {
		listenForEventConsoleAPICalled(taskCtx, logger, eventConsoleAPICalledOptions{
			types:    options.FailOnConsoleMessageTypes,
			failures: failures,
		})
	}

	// See:
	// https://github.com/gotenberg/gotenberg/issues/913.
	// https://github.com/gotenberg/gotenberg/issues/959.
	// https://github.com/gotenberg/gotenberg/issues/1021.
	listenForEventLoadingFailed(taskCtx, logger, eventLoadingFailedOptions{
		failures:              failures,
		cancelOnMainPageError: taskCancel,
	})

	// See https://github.com/gotenberg/gotenberg/issues/1640.
	listenForEventTargetCrashed(taskCtx, logger, eventTargetCrashedOptions{
		failures: failures,
		cancel:   taskCancel,
	})
}

// invalidPrinterSettingsMessages are the errors Chromium returns when it
// rejects the Page.printToPDF parameters. The InvalidParams (-32602) messages
// come from GetPrintPagesParams in Chromium's
// components/printing/browser/print_to_pdf/pdf_print_utils.cc.
var invalidPrinterSettingsMessages = []string{
	"Show invalid printer settings error (-32000)",
	"scale is outside of [0.1 - 2] range (-32602)",
	"margin is negative (-32602)",
	"paper width is zero or negative (-32602)",
	"paper height is zero or negative (-32602)",
	"printable area is empty (-32602)",
	"content area is empty (-32602)",
	"invalid print parameters (-32602)",
}

// isInvalidPrinterSettingsError reports whether errMessage carries one of the
// errors Chromium returns for aberrant Page.printToPDF parameters.
func isInvalidPrinterSettingsError(errMessage string) bool {
	return slices.ContainsFunc(invalidPrinterSettingsMessages, func(message string) bool {
		return strings.Contains(errMessage, message)
	})
}

// Interface guards.
var (
	_ gotenberg.Process = (*chromiumBrowser)(nil)
	_ browser           = (*chromiumBrowser)(nil)
)
