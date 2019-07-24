package printer

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
	"github.com/thecodingmachine/gotenberg/internal/pkg/xcontext"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
	"golang.org/x/sync/errgroup"
)

type chromePrinter struct {
	logger xlog.Logger
	url    string
	opts   ChromePrinterOptions
}

// ChromePrinterOptions helps customizing the
// Google Chrome printer behaviour.
type ChromePrinterOptions struct {
	WaitTimeout  float64
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

func (p chromePrinter) Print(destination string) error {
	const op string = "printer.chromePrinter.Print"
	logOptions(p.logger, p.opts)
	ctx, cancel := xcontext.WithTimeout(p.logger, p.opts.WaitTimeout+p.opts.WaitDelay)
	defer cancel()
	resolver := func() error {
		devt, err := devtool.New("http://localhost:9222").Version(ctx)
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
		// create a new blank target with the new browser context.
		createTargetArgs := target.
			NewCreateTargetArgs("about:blank").
			SetBrowserContextID(newContextTarget.BrowserContextID)
		newTarget, err := devtClient.Target.CreateTarget(ctx, createTargetArgs)
		if err != nil {
			return err
		}
		// connect the client to the new target.
		newTargetWsURL := fmt.Sprintf("ws://127.0.0.1:9222/devtools/page/%s", newTarget.TargetID)
		newContextConn, err := rpcc.DialContext(ctx, newTargetWsURL)
		if err != nil {
			return err
		}
		defer newContextConn.Close() // nolint: errcheck
		// create a new CDP Client that uses newContextConn.
		targetClient := cdp.NewClient(newContextConn)
		closeTargetArgs := target.NewCloseTargetArgs(newTarget.TargetID)
		// close the target when done.
		defer targetClient.Target.CloseTarget(ctx, closeTargetArgs) // nolint: errcheck
		if err := runBatch(
			// enable all the domain events that we're interested in.
			func() error { return targetClient.DOM.Enable(ctx) },
			func() error { return targetClient.Network.Enable(ctx, network.NewEnableArgs()) },
			func() error { return targetClient.Page.Enable(ctx) },
			func() error { return targetClient.Runtime.Enable(ctx) },
		); err != nil {
			return err
		}
		if err := p.navigate(ctx, targetClient); err != nil {
			return err
		}
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
		if err := ioutil.WriteFile(destination, print.Data, 0644); err != nil {
			return err
		}
		return nil
	}
	if err := resolver(); err != nil {
		return xcontext.MustHandleError(
			ctx,
			xerror.New(op, err),
		)
	}
	return nil
}

func (p chromePrinter) navigate(ctx context.Context, client *cdp.Client) error {
	const op string = "printer.chromePrinter.navigate"
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
		loadingFinished, err := client.Network.LoadingFinished(ctx)
		if err != nil {
			return err
		}
		defer loadingFinished.Close() // nolint: errcheck
		if _, err := client.Page.Navigate(ctx, page.NewNavigateArgs(p.url)); err != nil {
			return err
		}
		if err := runBatch(
			// wait for all events.
			func() error { _, err := domContentEventFired.Recv(); return err },
			func() error { _, err := loadEventFired.Recv(); return err },
			func() error { _, err := loadingFinished.Recv(); return err },
		); err != nil {
			return err
		}
		// wait for a given amount of time (useful for javascript delay).
		time.Sleep(xtime.Duration(p.opts.WaitDelay))
		return nil
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func runBatch(fn ...func() error) error {
	// run all functions simultaneously and wait until
	// execution has completed or an error is encountered.
	eg := errgroup.Group{}
	for _, f := range fn {
		eg.Go(f)
	}
	return eg.Wait()
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(chromePrinter))
)
