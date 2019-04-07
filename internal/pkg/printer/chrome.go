package printer

import (
	"context"
	"fmt"
	"time"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/network"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/rpcc"
	"github.com/thecodingmachine/gotenberg/internal/pkg/file"
	"github.com/thecodingmachine/gotenberg/internal/pkg/hijackable"
	"golang.org/x/sync/errgroup"
)

type chrome struct {
	url  string
	opts *ChromeOptions

	conn                 *rpcc.Conn
	client               *cdp.Client
	exceptionThrown      runtime.ExceptionThrownClient
	loadingFailed        network.LoadingFailedClient
	domContentEventFired page.DOMContentEventFiredClient
	loadEventFired       page.LoadEventFiredClient
}

// ChromeOptions helps customizing the
// Chrome printer behaviour.
type ChromeOptions struct {
	HeaderHTML   string
	FooterHTML   string
	PaperWidth   float64
	PaperHeight  float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
	Landscape    bool
	WaitTimeout  float64
}

const defaultHeaderFooterHTML string = "<html><head></head><body></body></html>"

func newChrome(URL string, opts *ChromeOptions) (Printer, error) {
	if URL == "" {
		return nil, fmt.Errorf("URL should not be empty: got %s", URL)
	}
	// if no header or footer, use the default template
	// for avoiding displaying default Chrome templates.
	if opts.HeaderHTML == "" {
		opts.HeaderHTML = defaultHeaderFooterHTML
	}
	if opts.FooterHTML == "" {
		opts.FooterHTML = defaultHeaderFooterHTML
	}
	// if no custom paper size, set default size to A4.
	if opts.PaperWidth == 0.0 && opts.PaperHeight == 0.0 {
		opts.PaperWidth, opts.PaperHeight = 8.27, 11.7
	}
	// if no custom margins, set default margins to 1 inch.
	if opts.MarginTop == 0.0 && opts.MarginBottom == 0.0 && opts.MarginLeft == 0.0 && opts.MarginRight == 0.0 {
		opts.MarginTop, opts.MarginBottom, opts.MarginLeft, opts.MarginRight = 1, 1, 1, 1
	}
	// if no custom timeout, set default timeout to 30 seconds.
	if opts.WaitTimeout == 0.0 {
		opts.WaitTimeout = 30.0
	}
	return &chrome{
		url:  URL,
		opts: opts,
	}, nil
}

func (c *chrome) Print(destination string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	devt := devtool.New("http://localhost:9222")
	pt, err := devt.Get(ctx, devtool.Page)
	if err != nil {
		return err
	}
	// connect to WebSocket URL (page) that speaks the Chrome DevTools Protocol.
	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	if err != nil {
		return err
	}
	c.conn = conn
	// create a new CDP Client that uses conn.
	c.client = cdp.NewClient(conn)
	// give enough capacity to avoid blocking any event listeners.
	abort := make(chan error, 2)
	// watch the abort channel.
	go func() {
		select {
		case <-ctx.Done():
		case <-abort:
			cancel()
		}
	}()
	// setup event handlers early because domain events can be sent as
	// soon as Enable is called on the domain.
	if err := c.abortOnErrors(ctx, abort); err != nil {
		return hijackable.HijackOnError(c, err)
	}
	if err := chromeEnableEvents(
		// enable all the domain events that we're interested in.
		func() error { return c.client.DOM.Enable(ctx) },
		func() error { return c.client.Network.Enable(ctx, nil) },
		func() error { return c.client.Page.Enable(ctx) },
		func() error { return c.client.Runtime.Enable(ctx) },
	); err != nil {
		return hijackable.HijackOnError(c, err)
	}
	if err := c.navigate(ctx); err != nil {
		return hijackable.HijackOnError(c, err)
	}
	print, err := c.client.Page.PrintToPDF(
		ctx,
		page.NewPrintToPDFArgs().
			SetPaperWidth(c.opts.PaperWidth).
			SetPaperHeight(c.opts.PaperHeight).
			SetMarginTop(c.opts.MarginTop).
			SetMarginBottom(c.opts.MarginBottom).
			SetMarginLeft(c.opts.MarginLeft).
			SetMarginRight(c.opts.MarginRight).
			SetLandscape(c.opts.Landscape).
			SetDisplayHeaderFooter(true).
			SetHeaderTemplate(c.opts.HeaderHTML).
			SetFooterTemplate(c.opts.FooterHTML).
			SetPrintBackground(true),
	)
	if err != nil {
		return hijackable.HijackOnError(c, fmt.Errorf("printing page to PDF: %v", err))
	}
	if err := file.WriteBytesToFile(destination, print.Data); err != nil {
		return hijackable.HijackOnError(c, err)
	}
	return c.Hijack()
}

func (c *chrome) Hijack() error {
	if c.loadEventFired != nil {
		if err := c.loadEventFired.Close(); err != nil {
			return err
		}
	}
	if c.domContentEventFired != nil {
		if err := c.domContentEventFired.Close(); err != nil {
			return err
		}
	}
	if c.loadingFailed != nil {
		if err := c.loadingFailed.Close(); err != nil {
			return err
		}
	}
	if c.exceptionThrown != nil {
		if err := c.exceptionThrown.Close(); err != nil {
			return err
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (c *chrome) abortOnErrors(ctx context.Context, abort chan<- error) error {
	exceptionThrown, err := c.client.Runtime.ExceptionThrown(ctx)
	if err != nil {
		return err
	}
	c.exceptionThrown = exceptionThrown
	loadingFailed, err := c.client.Network.LoadingFailed(ctx)
	if err != nil {
		return err
	}
	c.loadingFailed = loadingFailed
	go func() {
		for {
			select {
			// check for exceptions so we can abort as soon
			// as one is encountered.
			case <-c.exceptionThrown.Ready():
				ev, err := c.exceptionThrown.Recv()
				if err != nil {
					// this could be any one of: stream closed,
					// connection closed, context deadline or
					// unmarshal failed.
					abort <- err
					return
				}
				// ruh-roh! Let the caller know something went wrong.
				abort <- ev.ExceptionDetails
			// check for non-canceled resources that failed
			// to load.
			case <-c.loadingFailed.Ready():
				ev, err := c.loadingFailed.Recv()
				if err != nil {
					abort <- err
					return
				}
				// for now, most optional fields are pointers
				// and must be checked for nil.
				canceled := ev.Canceled != nil && *ev.Canceled
				if !canceled {
					abort <- fmt.Errorf("request %s failed: %s", ev.RequestID, ev.ErrorText)
				}
			}
		}
	}()
	return nil
}

func (c *chrome) navigate(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.opts.WaitTimeout)*time.Second)
	defer cancel()
	// make sure Page events are enabled.
	if err := c.client.Page.Enable(ctx); err != nil {
		return err
	}
	// open client for DOMContentEventFired to block until DOM has fully loaded.
	domContentEventFired, err := c.client.Page.DOMContentEventFired(ctx)
	if err != nil {
		return err
	}
	c.domContentEventFired = domContentEventFired
	// open client for LoadEventFired to block until navigation is finished.
	loadEventFired, err := c.client.Page.LoadEventFired(ctx)
	if err != nil {
		return err
	}
	c.loadEventFired = loadEventFired
	// TODO check c.client.Network.LoadingFinished
	_, err = c.client.Page.Navigate(ctx, page.NewNavigateArgs(c.url))
	if err != nil {
		return err
	}
	if _, err := c.domContentEventFired.Recv(); err != nil {
		return err
	}
	if _, err := c.loadEventFired.Recv(); err != nil {
		return err
	}
	return nil
}

type chomeEnableEventsFunc func() error

func chromeEnableEvents(fn ...chomeEnableEventsFunc) error {
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
	_ = Printer(new(chrome))
	_ = hijackable.Hijackable(new(chrome))
)
