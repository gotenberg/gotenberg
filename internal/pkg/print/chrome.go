package print

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
	"github.com/thecodingmachine/gotenberg/internal/pkg/process"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerrgroup"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
)

type chromePrint struct {
	logger xlog.Logger
	url    string
	opts   ChromePrintOptions
}

// ChromePrintOptions helps customizing the
// Google Chrome Print result.
type ChromePrintOptions struct {
	WaitDelay    float64
	HeaderHTML   string
	FooterHTML   string
	PaperWidth   float64
	PaperHeight  float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
	Landscape    bool
}

// DefaultChromePrintOptions returns the default
// Google Chrome Print options.
func DefaultChromePrintOptions() ChromePrintOptions {
	const defaultHeaderFooterHTML string = "<html><head></head><body></body></html>"
	return ChromePrintOptions{
		WaitDelay:    0.0,
		HeaderHTML:   defaultHeaderFooterHTML,
		FooterHTML:   defaultHeaderFooterHTML,
		PaperWidth:   8.27,
		PaperHeight:  11.7,
		MarginTop:    1.0,
		MarginBottom: 1.0,
		MarginLeft:   1.0,
		MarginRight:  1.0,
		Landscape:    false,
	}
}

func (p chromePrint) Print(ctx context.Context, dest string, proc process.Process) error {
	const op string = "print.chromePrint.Print"
	resolver := func() error {
		devtEndpoint := fmt.Sprintf("http://%s:%d", proc.Host(), proc.Port())
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
		newTargetWsURL := fmt.Sprintf("ws://%s:%d/devtools/page/%s", proc.Host(), proc.Port(), newTarget.TargetID)
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

func (p chromePrint) enableEvents(ctx context.Context, client *cdp.Client) error {
	const op string = "print.chromePrint.enableEvents"
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

func (p chromePrint) listenEvents(ctx context.Context, client *cdp.Client) error {
	const op string = "print.chromePrint.listenEvents"
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
	_ = Print(new(chromePrint))
)
