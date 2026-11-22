package chromium

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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
// [chromiumSwitches] keys the switches by name, as chromedp does, so this
// value replaces chromedp's instead of merging with it. Revisit this list when
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
	binPath string
	// maxMemory is the anonymous resident memory, in bytes, Chromium may
	// hold while it converts, 0 for no limit (see --chromium-max-memory).
	maxMemory                uint64
	allowInsecureLocalhost   bool
	ignoreCertificateErrors  bool
	disableWebSecurity       bool
	allowFileAccessFromFiles bool
	hostResolverRules        string
	hostRules                hostResolverRules
	proxyServer              string
	enableEnvironmentProxy   bool
	wsUrlReadTimeout         time.Duration
	hyphenDataDirPath        string
	// debugProtocol logs every DevTools Protocol message, which carries URLs,
	// cookies and header values whole.
	debugProtocol bool

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

// usesPinningProxy reports whether Chromium sends its traffic through the
// pinning proxy. An operator egress proxy set with --chromium-proxy-server
// takes its place.
func (a browserArguments) usesPinningProxy() bool {
	return a.proxyServer == ""
}

// hostMapping returns the --chromium-host-resolver-rules mapping that the
// outbound checks apply, or nil. Chromium hands every destination to the
// pinning proxy, so the rules must apply there. Behind the operator's proxy,
// Chromium keeps applying them itself.
func (a browserArguments) hostMapping() gotenberg.HostMapping {
	if !a.usesPinningProxy() || (len(a.hostRules.maps) == 0 && len(a.hostRules.excludes) == 0) {
		return nil
	}
	return a.hostRules.rewrite
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
	pinningProxy *gotenberg.OutboundProxy

	// pid is the browser process, which the memory watch reads with its
	// descendants, the renderers. Set by Start, under ctxMu.
	pid int
	// watchMu guards inFlight and stopWatching: the memory of Chromium is
	// watched while at least one conversion runs.
	watchMu      sync.Mutex
	inFlight     int
	stopWatching context.CancelFunc
	// memoryExceeded holds the memory Chromium held when the watch killed
	// it, 0 until then. Start resets it.
	memoryExceeded atomic.Uint64
}

func newChromiumBrowser(arguments browserArguments) browser {
	b := &chromiumBrowser{
		initialCtx: context.Background(),
		arguments:  arguments,
		fs:         gotenberg.NewFileSystem(new(gotenberg.OsMkdirAll)),
		pinningProxy: gotenberg.NewOutboundProxy(gotenberg.OutboundProxyOptions{
			AllowList:              arguments.allowList,
			DenyList:               arguments.denyList,
			DenyPrivateIPs:         arguments.denyPrivateIPs,
			DenyPublicIPs:          arguments.denyPublicIPs,
			EnableEnvironmentProxy: arguments.enableEnvironmentProxy,
			HostMapping:            arguments.hostMapping(),
		}),
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

	// The pinning proxy starts first, as Chromium's switches name its address.
	if b.arguments.usesPinningProxy() {
		err = b.pinningProxy.Start(logger.With(slog.String("logger", "pinning-proxy")))
		if err != nil {
			return fmt.Errorf("start pinning proxy: %w", err)
		}

		// Tear it down on a failed start so a supervisor retry can re-bind.
		// See https://github.com/gotenberg/gotenberg/issues/1559.
		defer func() {
			if err == nil {
				return
			}

			stopErr := b.stopPinningProxy()
			if stopErr != nil {
				logger.ErrorContext(context.Background(), fmt.Sprintf("stop pinning proxy after failed start: %s", stopErr))
			}
		}()
	}

	// --chromium-start-timeout bounds the start, from the launch to the
	// DevTools handshake.
	// See https://github.com/gotenberg/gotenberg/issues/524.
	startCtx, cancelStartTimeout := context.WithTimeout(ctx, b.arguments.wsUrlReadTimeout)
	defer cancelStartTimeout()

	process, err := b.launch(logger, debug)
	if err != nil {
		return fmt.Errorf("launch Chromium: %w", err)
	}

	// Until the start succeeds, its context may abort it.
	abortStart := context.AfterFunc(startCtx, process.stop)
	defer func() {
		if err == nil {
			return
		}

		abortStart()
		process.stop()
		<-process.exited

		if ctx.Err() != nil {
			err = fmt.Errorf("start Chromium: %w: %w", ctx.Err(), err)
			return
		}
		err = fmt.Errorf("start Chromium: %w; if Chromium is slow to start, raise --chromium-start-timeout (CHROMIUM_START_TIMEOUT), currently %s", err, b.arguments.wsUrlReadTimeout)
	}()

	initialPage, err := process.pipe.initialPage()
	if err != nil {
		return fmt.Errorf("discover the initial page: %w", process.exitError(startCtx, err))
	}

	deadline, _ := startCtx.Deadline()
	bridge, err := newDevToolsBridge(process.pipe, deadline)
	if err != nil {
		return fmt.Errorf("bridge DevTools: %w", err)
	}
	// Nothing listens anymore once the start is over.
	defer bridge.closeListener()

	// The browser outlives the start, so its context derives from the
	// process's, which derives from initialCtx.
	allocatorCtx, allocatorCancel := chromedp.NewRemoteAllocator(process.ctx, bridge.url(), chromedp.NoModifyURL)
	contextOpts := []chromedp.ContextOption{
		chromedp.WithTargetID(initialPage),
		chromedp.WithErrorf(func(format string, v ...any) {
			if format == "executor for %q doesn't exist" {
				// The end of a session chromedp does not drive: the browser
				// session auto-attaches shared and service workers, see
				// lockBrowserWorkers, and chromedp drives a session of its
				// own to each.
				debug.Printf(format, v...)
				return
			}
			// chromedp's default.
			log.Printf("ERROR: "+format, v...)
		}),
	}
	if b.arguments.debugProtocol {
		contextOpts = append(contextOpts, chromedp.WithDebugf(debug.Printf))
	}
	browserCtx, cancel := chromedp.NewContext(allocatorCtx, contextOpts...)
	// chromedp cancels browserCtx when it loses the connection, but does not
	// kill a process it did not start.
	stopKillOnLostConnection := context.AfterFunc(browserCtx, process.stop)
	stop := func() {
		// chromedp first detaches from the tab over the live connection.
		stopKillOnLostConnection()
		cancel()
		allocatorCancel()
		process.stop()
		<-process.exited
	}

	err = chromedp.Run(browserCtx)
	if err == nil {
		err = lockBrowserWorkers(startCtx, browserCtx, logger)
	}
	if err == nil && !abortStart() {
		// startCtx was done right as Chromium came up: the browser is going
		// down.
		err = startCtx.Err()
	}
	if err != nil {
		stop()
		return fmt.Errorf("connect to Chromium: %w", process.exitError(startCtx, err))
	}

	b.ctxMu.Lock()
	defer b.ctxMu.Unlock()

	// We have to keep the context around, as we need it to create new tabs
	// later.
	b.ctx = browserCtx
	b.cancelFunc = stop
	b.pid = process.cmd.Process.Pid
	b.memoryExceeded.Store(0)
	b.isStarted.Store(true)

	return nil
}

// chromiumProcess is a Chromium process Gotenberg drives over
// --remote-debugging-pipe.
type chromiumProcess struct {
	cmd  *exec.Cmd
	pipe *devToolsPipe
	// ctx is done once the process must go: stop kills it then.
	ctx  context.Context
	stop context.CancelFunc
	// exited is closed once the process has exited, after which waitErr
	// holds how.
	exited  chan struct{}
	waitErr error
}

// launch starts Chromium with the switches of [chromiumSwitches]. Its output
// goes to the debug logs. The process lives until its stop function is
// called, or Gotenberg exits.
func (b *chromiumBrowser) launch(logger *slog.Logger, debug *debugLogger) (*chromiumProcess, error) {
	proxyServer := b.arguments.proxyServer
	if b.arguments.usesPinningProxy() {
		proxyServer = "http://" + b.pinningProxy.Addr()
	}

	pipe, chromiumEnds, err := newDevToolsPipe()
	if err != nil {
		return nil, fmt.Errorf("create DevTools pipe: %w", err)
	}

	output, chromiumOutput, err := os.Pipe()
	if err != nil {
		pipe.close()
		closeFiles(chromiumEnds...)
		return nil, fmt.Errorf("create output pipe: %w", err)
	}

	cmd := exec.Command(b.arguments.binPath, chromiumSwitches(b.arguments, b.userProfileDirPath, proxyServer)...) //nolint:gosec
	cmd.ExtraFiles = chromiumEnds
	cmd.Stdout = chromiumOutput
	cmd.Stderr = chromiumOutput
	setParentDeathSignal(cmd)

	logger.DebugContext(context.Background(), fmt.Sprintf("start Chromium: %s", strings.Join(cmd.Args, " ")))

	err = cmd.Start()
	// Chromium holds its own copies now.
	closeFiles(append(chromiumEnds, chromiumOutput)...)
	if err != nil {
		pipe.close()
		closeFiles(output)
		return nil, fmt.Errorf("start Chromium: %w", err)
	}

	go func() {
		_, _ = io.Copy(debug, output)
		closeFiles(output)
	}()

	processCtx, stop := context.WithCancel(b.initialCtx)
	process := &chromiumProcess{
		cmd:    cmd,
		pipe:   pipe,
		ctx:    processCtx,
		stop:   stop,
		exited: make(chan struct{}),
	}

	go func() {
		process.waitErr = cmd.Wait()
		pipe.close()
		close(process.exited)
	}()

	context.AfterFunc(processCtx, func() {
		err := cmd.Process.Kill()
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			logger.ErrorContext(context.Background(), fmt.Sprintf("kill Chromium: %s", err))
		}
		pipe.close()
	})

	return process, nil
}

// exitError adds to err, an error the start ran into, its cause: the start
// ran out of time, or Chromium exited.
func (p *chromiumProcess) exitError(startCtx context.Context, err error) error {
	select {
	case <-p.exited:
		if startCtx.Err() == nil {
			return fmt.Errorf("Chromium exited (%v): %w", p.waitErr, err)
		}
	case <-startCtx.Done():
	}

	return fmt.Errorf("%w: %w", startCtx.Err(), err)
}

// chromiumSwitches returns the command line Gotenberg starts Chromium with.
// proxyServer is the address of the proxy Chromium sends its traffic to.
func chromiumSwitches(arguments browserArguments, userProfileDirPath, proxyServer string) []string {
	// A string value gives --name=value, true gives --name, and false leaves
	// the switch out.
	switches := map[string]any{
		// The switches of chromedp.DefaultExecAllocatorOptions (chromedp
		// v0.14.2), which Gotenberg keeps while it starts Chromium itself.
		// Revisit this list when bumping chromedp.
		"no-first-run":                           true,
		"no-default-browser-check":               true,
		"headless":                               true,
		"hide-scrollbars":                        true,
		"mute-audio":                             true,
		"disable-background-networking":          true,
		"enable-features":                        "NetworkService,NetworkServiceInProcess",
		"disable-background-timer-throttling":    true,
		"disable-backgrounding-occluded-windows": true,
		"disable-breakpad":                       true,
		"disable-client-side-phishing-detection": true,
		"disable-default-apps":                   true,
		"disable-dev-shm-usage":                  true,
		"disable-extensions":                     true,
		"disable-hang-monitor":                   true,
		"disable-ipc-flooding-protection":        true,
		"disable-popup-blocking":                 true,
		"disable-prompt-on-repost":               true,
		"disable-renderer-backgrounding":         true,
		"disable-sync":                           true,
		"force-color-profile":                    "srgb",
		"metrics-recording-only":                 true,
		"safebrowsing-disable-auto-update":       true,
		"enable-automation":                      true,
		"password-store":                         "basic",
		"use-mock-keychain":                      true,

		// Gotenberg's own.
		"no-sandbox": true,
		// See:
		// https://github.com/puppeteer/puppeteer/issues/661
		// https://github.com/puppeteer/puppeteer/issues/2410
		"font-render-hinting": "none",
		"user-data-dir":       userProfileDirPath,
		// See https://github.com/gotenberg/gotenberg/issues/831.
		"disable-pdf-tagging": true,
		// See https://github.com/gotenberg/gotenberg/issues/1177.
		"no-zygote": true,
		// See https://github.com/gotenberg/gotenberg/issues/1293.
		"disable-component-update": false,
		// See https://github.com/gotenberg/gotenberg/issues/1656.
		"disable-features": chromiumDisableFeatures,
		// See https://github.com/gotenberg/gotenberg/issues/1706.
		"gaia-url": chromiumGaiaUrl,
		// DevTools over file descriptors 3 and 4, and no port: see
		// [devToolsPipe].
		"remote-debugging-pipe": true,
	}

	if arguments.allowInsecureLocalhost {
		// See https://github.com/gotenberg/gotenberg/issues/488.
		switches["allow-insecure-localhost"] = true
	}

	if arguments.ignoreCertificateErrors {
		switches["ignore-certificate-errors"] = true
	}

	if arguments.disableWebSecurity {
		switches["disable-web-security"] = true
	}

	if arguments.allowFileAccessFromFiles {
		// See https://github.com/gotenberg/gotenberg/issues/356.
		switches["allow-file-access-from-files"] = true
	}

	if arguments.hostResolverRules != "" && !arguments.usesPinningProxy() {
		// Behind the pinning proxy, Chromium resolves nothing but the proxy
		// itself, and would apply a rule such as "MAP * ~NOTFOUND" to its
		// 127.0.0.1 address. The pinning proxy applies the rules instead.
		// See https://github.com/gotenberg/gotenberg/issues/488.
		switches["host-resolver-rules"] = arguments.hostResolverRules
	}

	if proxyServer != "" {
		// Default: route Chromium through the internal pinning proxy so that
		// Chromium never performs its own DNS lookup for the navigation URL
		// or any sub-resource. The proxy resolves and validates each URL
		// once per request and dials the pinned IP, closing the DNS
		// rebinding window between Gotenberg's validation and Chromium's
		// connect.
		//
		// The operator's egress proxy, set with --chromium-proxy-server,
		// takes its place: that proxy owns outbound safety, and routing
		// through an internal proxy would override it. Custom host-resolver
		// mappings keep the pinning proxy, which applies them, so that the
		// outbound policy still covers WebSocket handshakes. Without it,
		// nothing could block those.
		// See https://github.com/gotenberg/gotenberg/issues/376.
		switches["proxy-server"] = proxyServer
	}

	if arguments.usesPinningProxy() && (arguments.restrictsNetwork() || arguments.hostMapping() != nil) {
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
		// response instead of a loading failure. Host-resolver mappings
		// also need it, since a rule may send "localhost" elsewhere and
		// only the proxy applies the rules.
		switches["proxy-bypass-list"] = "<-loopback>"
	}

	args := make([]string, 0, len(switches)+1)
	for name, value := range switches {
		switch value := value.(type) {
		case string:
			args = append(args, fmt.Sprintf("--%s=%s", name, value))
		case bool:
			if value {
				args = append(args, "--"+name)
			}
		}
	}
	slices.Sort(args)

	// The first tab opens on a blank page, which --no-first-run alone does
	// not ensure.
	return append(args, "about:blank")
}

// closeFiles closes files, ignoring errors: callers close copies another
// process holds, or ends they no longer read.
func closeFiles(files ...*os.File) {
	for _, f := range files {
		_ = f.Close()
	}
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
	err := b.stopPinningProxy()
	if err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("stop pinning proxy: %s", err))
	}

	return nil
}

// stopPinningProxy stops the pinning proxy, giving its in-flight handlers a
// few seconds to complete.
func (b *chromiumBrowser) stopPinningProxy() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return b.pinningProxy.Stop(ctx)
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
		pauseWorkersActionFunc(logger),
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
		waitForWindowStatusBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitForWindowStatus),
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
		pauseWorkersActionFunc(logger),
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
		waitForWindowStatusBeforePrintActionFunc(logger, b.arguments.disableJavaScript, options.WaitForWindowStatus),
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

func (b *chromiumBrowser) do(ctx context.Context, logger *slog.Logger, url string, options Options, aggregate *networkAggregate, tasks chromedp.Tasks) (err error) {
	if !b.isStarted.Load() {
		return errors.New("browser not started, cannot handle tasks")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return errors.New("context has no deadline")
	}

	// We validate the "main" URL against our allowed / deny lists, and
	// against the IP-based outbound URL guard. See [gotenberg.FilterOutboundURL].
	// The page the HTML and Markdown routes wrote is Gotenberg's own: the lists
	// govern what it loads, not the page.
	if !isLocalPage(url, options.LocalPage) {
		err := gotenberg.FilterOutboundURL(ctx, url, b.arguments.allowList, b.arguments.denyList, deadline,
			gotenberg.WithDenyPrivateIPs(b.arguments.denyPrivateIPs),
			gotenberg.WithDenyPublicIPs(b.arguments.denyPublicIPs),
			gotenberg.WithHostMapping(b.arguments.hostMapping()),
		)
		if err != nil {
			return fmt.Errorf("filter URL: %w", err)
		}
	}

	b.ctxMu.RLock()
	defer b.ctxMu.RUnlock()

	defer b.watchMemory(logger)()

	// Killed for its memory, Chromium fails the conversion wherever it was,
	// creating the tab included.
	defer func() {
		if used := b.memoryExceeded.Load(); err != nil && used > 0 {
			err = fmt.Errorf("handle tasks: %w: %s held, %s allowed: %w", ErrChromiumMemoryLimitExceeded, gotenberg.FormatMemory(used), gotenberg.FormatMemory(b.arguments.maxMemory), err)
		}
	}()

	// The tab derives from the browser, not from the request, but must not
	// outlive it: the supervisor holds the request's slot until do returns.
	// It ends with the request's own cause, so that a wait tells a deadline
	// from a client that went away.
	requestCtx, requestCancel := context.WithCancelCause(b.ctx)
	defer requestCancel(nil)
	stopCancelOnRequest := context.AfterFunc(ctx, func() { requestCancel(context.Cause(ctx)) })
	defer stopCancelOnRequest()
	timeoutCtx, timeoutCancel := context.WithDeadline(requestCtx, deadline)
	defer timeoutCancel()

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
	b.listen(ctx, taskCtx, taskCancel, logger, options, aggregate, failures)

	runErr := chromedp.Run(taskCtx, tasks...)

	runErr = requestError(ctx, runErr)

	return failures.err(runErr, b.ctx.Err() != nil, options.FailOnResourceLoadingFailed)
}

// watchMemory counts a conversion in flight, and watches the memory of
// Chromium while one is: a page that never stops allocating would get the
// container OOM-killed, Gotenberg with it, since on cgroup v2 Kubernetes kills
// every process of the container together. Killing Chromium first fails the
// conversions in flight alone, and the supervisor restarts it before the next
// one. An idle Chromium is not watched, so that a limit under its idle memory
// fails conversions rather than restarting it forever. The caller holds ctxMu
// for reading, and calls the returned function once the conversion ends.
func (b *chromiumBrowser) watchMemory(logger *slog.Logger) func() {
	if b.arguments.maxMemory == 0 || b.pid == 0 {
		return func() {}
	}

	b.watchMu.Lock()
	defer b.watchMu.Unlock()

	b.inFlight++
	if b.inFlight == 1 {
		ctx, cancel := context.WithCancel(context.Background())
		b.stopWatching = cancel
		pid := b.pid

		go gotenberg.WatchMemory(ctx, logger, pid, b.arguments.maxMemory, func(used uint64) {
			b.memoryExceeded.Store(used)
			logger.WarnContext(ctx, fmt.Sprintf("Chromium holds %s of memory, over the %s of --chromium-max-memory (env CHROMIUM_MAX_MEMORY): kill it", gotenberg.FormatMemory(used), gotenberg.FormatMemory(b.arguments.maxMemory)))

			// The renderers exit with the browser, and Stop kills any that
			// linger when the supervisor restarts it.
			err := syscall.Kill(pid, syscall.SIGKILL)
			if err != nil && !errors.Is(err, syscall.ESRCH) {
				logger.ErrorContext(ctx, fmt.Sprintf("kill Chromium: %s", err))
			}
		})
	}

	return func() {
		b.watchMu.Lock()
		defer b.watchMu.Unlock()

		b.inFlight--
		if b.inFlight == 0 {
			b.stopWatching()
			b.stopWatching = nil
		}
	}
}

// requestError returns the error of a run that ctx, the request's context,
// may have ended. The tab is canceled when the request ends, so the run
// reports context.Canceled even when the request ran out of time: the
// request's own error then comes first, so that a deadline still maps to a 503
// and a client that went away to a 499. A wait that ran out of time already
// says so, and names its form field.
func requestError(ctx context.Context, runErr error) error {
	if runErr == nil || ctx.Err() == nil || errors.Is(runErr, ErrWaitTimeout) {
		return runErr
	}

	return fmt.Errorf("%w: %w", ctx.Err(), runErr)
}

// listen registers the event listeners of one conversion on taskCtx. They
// record what fails the conversion in failures, and some cancel taskCtx so
// that the conversion stops at once. Their log records about the page carry
// requestCtx, the client request's, for its trace.
func (b *chromiumBrowser) listen(requestCtx, taskCtx context.Context, taskCancel context.CancelFunc, logger *slog.Logger, options Options, aggregate *networkAggregate, failures *conversionFailures) {
	// Accumulate per-conversion network activity for telemetry.
	listenForNetworkActivity(taskCtx, aggregate)

	// We validate all other requests against our allowed / deny lists.
	// If a request does not pass the validation, we make it fail. It also set
	// the extra HTTP headers, if any.
	// See https://github.com/gotenberg/gotenberg/issues/1011.
	listenForEventRequestPaused(taskCtx, logger, eventRequestPausedOptions{
		requestCtx:          requestCtx,
		localPage:           options.LocalPage,
		allowList:           b.arguments.allowList,
		denyList:            b.arguments.denyList,
		denyPrivateIPs:      b.arguments.denyPrivateIPs,
		denyPublicIPs:       b.arguments.denyPublicIPs,
		hostMapping:         b.arguments.hostMapping(),
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
		hostMapping:    b.arguments.hostMapping(),
		failures:       failures,
		cancel:         taskCancel,
	}
	if b.arguments.usesPinningProxy() {
		webSocketOptions.decideTunnel = b.pinningProxy.DecideTunnel
	}

	listenForEventWebSocketCreated(taskCtx, logger, webSocketOptions)

	// Workers load what they want out of reach of the listeners above. They
	// start paused, see pauseWorkersActionFunc, and stay locked out of
	// file:// URLs.
	listenForWorkers(taskCtx, logger)

	// See:
	// https://github.com/gotenberg/gotenberg/issues/613.
	// https://github.com/gotenberg/gotenberg/issues/1021.
	if len(options.FailOnHttpStatusCodes) != 0 || len(options.FailOnResourceHttpStatusCodes) != 0 {
		listenForEventResponseReceived(taskCtx, logger, eventResponseReceivedOptions{
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
