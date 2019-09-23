package prinery

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/network"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/protocol/target"
	"github.com/mafredri/cdp/rpcc"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerrgroup"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
)

type chromeProcess struct {
	devtPort uint
}

func newChromeProcesses(nInstances int64) []process {
	processes := make([]process, nInstances)
	var i int64
	var currentPort uint = 9222
	for i = 0; i < nInstances; i++ {
		processes[i] = chromeProcess{
			devtPort: currentPort,
		}
		currentPort++
	}
	return processes
}

func (p chromeProcess) id() string {
	return fmt.Sprintf("%s-%d", p.binary(), p.port())
}

func (p chromeProcess) host() string {
	return "127.0.0.1"
}

func (p chromeProcess) port() uint {
	return p.devtPort
}

func (p chromeProcess) spec() processSpec {
	return p
}

func (p chromeProcess) binary() string {
	return "google-chrome-stable"
}

func (p chromeProcess) args() []string {
	return []string{
		"--no-sandbox",
		"--headless",
		// see https://github.com/GoogleChrome/puppeteer/issues/2410.
		"--font-render-hinting=medium",
		fmt.Sprintf("--remote-debugging-port=%d", p.port()),
		"--disable-gpu",
		"--disable-translate",
		"--disable-extensions",
		"--disable-background-networking",
		"--safebrowsing-disable-auto-update",
		"--disable-sync",
		"--disable-default-apps",
		"--hide-scrollbars",
		"--metrics-recording-only",
		"--mute-audio",
		"--no-first-run",
	}
}

func (p chromeProcess) warmupTime() time.Duration {
	return 10 * time.Second
}

func (p chromeProcess) viabilityFunc() func(logger xlog.Logger) bool {
	const op string = "prinery.chromeProcess.viabilityFunc"
	return func(logger xlog.Logger) bool {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		endpoint := fmt.Sprintf("http://%s:%d" /*p.host()*/, "localhost", p.port())
		logger.DebugfOp(
			op,
			"checking '%s' viability via endpoint '%s/json/version'",
			p.id(),
			endpoint,
		)
		v, err := devtool.New(endpoint).Version(ctx)
		if err != nil {
			logger.ErrorfOp(
				op,
				"'%s' is not viable as endpoint returned '%v'",
				p.id(),
				err,
			)
			return false
		}
		logger.DebugfOp(
			op,
			"'%s' is viable as endpoint returned '%v'",
			p.id(),
			v,
		)
		return true
	}
}

type chromePrinter struct {
	logger xlog.Logger
	url    string
	opts   ChromePrintOptions
}

func (p chromePrinter) print(ctx context.Context, spec processSpec, dest string) error {
	const op string = "prinery.chromePrinter.print"
	resolver := func() error {
		devtEndpoint := fmt.Sprintf("http://%s:%d", spec.host(), spec.port())
		devt, err := devtool.New(devtEndpoint).Version(ctx)
		if err != nil {
			return err
		}
		// connect to WebSocket URL (page) that speaks the Chrome DevTools Protocol.
		devtConn, err := rpcc.DialContext(ctx, devt.WebSocketDebuggerURL)
		if err != nil {
			return err
		}
		defer devtConn.Close() // nolint: errcheck
		// create a new CDP Client that uses conn.
		devtClient := cdp.NewClient(devtConn)
		newContextTarget, err := devtClient.Target.CreateBrowserContext(ctx)
		if err != nil {
			return err
		}
		/*
			close the browser context when done.
			we're not using the "default" context
			as it may timeout before actually closing
			the browser context.
			see: https://github.com/mafredri/cdp/issues/101#issuecomment-524533670
		*/
		disposeBrowserContextArgs := target.NewDisposeBrowserContextArgs(newContextTarget.BrowserContextID)
		defer devtClient.Target.DisposeBrowserContext(context.Background(), disposeBrowserContextArgs) // nolint: errcheck
		// create a new blank target with the new browser context.
		createTargetArgs := target.
			NewCreateTargetArgs("about:blank").
			SetBrowserContextID(newContextTarget.BrowserContextID)
		newTarget, err := devtClient.Target.CreateTarget(ctx, createTargetArgs)
		if err != nil {
			return err
		}
		// connect the client to the new target.
		newTargetWsURL := fmt.Sprintf("ws://%s:%d/devtools/page/%s", spec.host(), spec.port(), newTarget.TargetID)
		newContextConn, err := rpcc.DialContext(ctx, newTargetWsURL)
		if err != nil {
			return err
		}
		defer newContextConn.Close() // nolint: errcheck
		// create a new CDP Client that uses newContextConn.
		targetClient := cdp.NewClient(newContextConn)
		/*
			close the target when done.
			we're not using the "default" context
			as it may timeout before actually closing
			the target.
			see: https://github.com/mafredri/cdp/issues/101#issuecomment-524533670
		*/
		closeTargetArgs := target.NewCloseTargetArgs(newTarget.TargetID)
		defer targetClient.Target.CloseTarget(context.Background(), closeTargetArgs) // nolint: errcheck
		// enable all events.
		if err := p.enableEvents(ctx, targetClient); err != nil {
			return err
		}
		// listen for all events.
		if err := p.listenEvents(ctx, targetClient); err != nil {
			return err
		}
		// apply a wait delay (if any).
		if p.opts.WaitDelay > 0.0 {
			// wait for a given amount of time (useful for javascript delay).
			p.logger.DebugfOp(op, "applying a wait delay of '%.2fs'...", p.opts.WaitDelay)
			time.Sleep(xtime.Duration(p.opts.WaitDelay))
		} else {
			p.logger.DebugOp(op, "no wait delay to apply, moving on...")
		}
		// print the page to PDF.
		print, err := targetClient.Page.PrintToPDF(
			ctx,
			page.NewPrintToPDFArgs().
				SetPaperWidth(p.opts.PaperWidth).
				SetPaperHeight(p.opts.PaperHeight).
				SetMarginTop(p.opts.MarginTop).
				SetMarginBottom(p.opts.MarginBottom).
				SetMarginLeft(p.opts.MarginLeft).
				SetMarginRight(p.opts.MarginRight).
				SetLandscape(p.opts.Landscape).
				SetDisplayHeaderFooter(true).
				SetHeaderTemplate(p.opts.HeaderHTML).
				SetFooterTemplate(p.opts.FooterHTML).
				SetPrintBackground(true),
		)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(dest, print.Data, 0644); err != nil {
			return err
		}
		return nil
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p chromePrinter) enableEvents(ctx context.Context, client *cdp.Client) error {
	const op string = "prinery.chromePrinter.enableEvents"
	// enable all the domain events that we're interested in.
	if err := xerrgroup.Run(
		func() error { return client.DOM.Enable(ctx) },
		func() error { return client.Network.Enable(ctx, network.NewEnableArgs()) },
		func() error { return client.Page.Enable(ctx) },
		func() error {
			return client.Page.SetLifecycleEventsEnabled(ctx, page.NewSetLifecycleEventsEnabledArgs(true))
		},
		func() error { return client.Runtime.Enable(ctx) },
	); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p chromePrinter) listenEvents(ctx context.Context, client *cdp.Client) error {
	const op string = "prinery.chromePrinter.listenEvents"
	resolver := func() error {
		// make sure Page events are enabled.
		if err := client.Page.Enable(ctx); err != nil {
			return err
		}
		// make sure Network events are enabled.
		if err := client.Network.Enable(ctx, nil); err != nil {
			return err
		}
		// create all clients for events.
		domContentEventFired, err := client.Page.DOMContentEventFired(ctx)
		if err != nil {
			return err
		}
		defer domContentEventFired.Close() // nolint: errcheck
		loadEventFired, err := client.Page.LoadEventFired(ctx)
		if err != nil {
			return err
		}
		defer loadEventFired.Close() // nolint: errcheck
		lifecycleEvent, err := client.Page.LifecycleEvent(ctx)
		if err != nil {
			return err
		}
		defer lifecycleEvent.Close() // nolint: errcheck
		loadingFinished, err := client.Network.LoadingFinished(ctx)
		if err != nil {
			return err
		}
		defer loadingFinished.Close() // nolint: errcheck
		if _, err := client.Page.Navigate(ctx, page.NewNavigateArgs(p.url)); err != nil {
			return err
		}
		// wait for all events.
		return xerrgroup.Run(
			func() error {
				_, err := domContentEventFired.Recv()
				if err != nil {
					return err
				}
				p.logger.DebugOp(op, "event 'domContentEventFired' received")
				return nil
			},
			func() error {
				_, err := loadEventFired.Recv()
				if err != nil {
					return err
				}
				p.logger.DebugOp(op, "event 'loadEventFired' received")
				return nil
			},
			func() error {
				const networkIdleEventName string = "networkIdle"
				for {
					ev, err := lifecycleEvent.Recv()
					if err != nil {
						return err
					}
					p.logger.DebugfOp(op, "event '%s' received", ev.Name)
					if ev.Name == networkIdleEventName {
						break
					}
				}
				return nil
			},
			func() error {
				_, err := loadingFinished.Recv()
				if err != nil {
					return err
				}
				p.logger.DebugOp(op, "event 'loadingFinished' received")
				return nil
			},
		)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = processSpec(new(chromeProcess))
	_ = process(new(chromeProcess))
	_ = printer(new(chromePrinter))
)
