package printer

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/network"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/protocol/target"
	"github.com/mafredri/cdp/rpcc"
)

// HTML facilitates HTML to PDF conversion.
type HTML struct {
	Context      context.Context
	URL          string
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

const defaultHeaderFooterHTML string = "<html><head></head><body></body></html>"

// Print converts HTML to PDF.
// Credits: https://medium.com/compass-true-north/go-service-to-convert-web-pages-to-pdf-using-headless-chrome-5fd9ffbae1af
func (html *HTML) Print(destination string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// use the DevTools HTTP/JSON API to manage targets (e.g. pages, webworkers).
	devt, err := devtool.New("http://localhost:9222").Version(ctx)
	if err != nil {
		return fmt.Errorf("creating DevTools target: %v", err)
	}
	// open a new RPC connection to the Chrome Debugging Protocol target.
	conn, err := rpcc.DialContext(html.Context, devt.WebSocketDebuggerURL)
	if err != nil {
		return fmt.Errorf("creating RPC connection: %v", err)
	}
	defer conn.Close()
	// create new browser context.
	baseBrowser := cdp.NewClient(conn)
	newContextTarget, err := baseBrowser.Target.CreateBrowserContext(html.Context)
	if err != nil {
		return fmt.Errorf("creating new browser context: %v", err)
	}
	// create a new blank target with the new browser context.
	newTargetArgs := target.NewCreateTargetArgs("about:blank").
		SetBrowserContextID(newContextTarget.BrowserContextID)
	newTarget, err := baseBrowser.Target.CreateTarget(html.Context, newTargetArgs)
	if err != nil {
		return fmt.Errorf("creating new blank target: %v", err)
	}
	// connect the client to the new target.
	newTargetWsURL := fmt.Sprintf("ws://127.0.0.1:9222/devtools/page/%s", newTarget.TargetID)
	newContextConn, err := rpcc.DialContext(html.Context, newTargetWsURL)
	if err != nil {
		return fmt.Errorf("connecting client to blank target: %v", err)
	}
	defer newContextConn.Close()
	// close the target when done.
	closeTargetArgs := target.NewCloseTargetArgs(newTarget.TargetID)
	defer baseBrowser.Target.CloseTarget(html.Context, closeTargetArgs)
	c := cdp.NewClient(newContextConn)
	// enable the runtime.
	if err := c.Runtime.Enable(html.Context); err != nil {
		return fmt.Errorf("enabling runtime: %v", err)
	}
	// enable the network.
	if err := c.Network.Enable(html.Context, network.NewEnableArgs()); err != nil {
		return fmt.Errorf("enabling network: %v", err)
	}
	// enable events on the page domain.
	if err := c.Page.Enable(html.Context); err != nil {
		return fmt.Errorf("enabling events on page domain: %v", err)
	}
	// create a client to listen for the load event to be fired.
	loadEventFiredClient, err := c.Page.LoadEventFired(html.Context)
	if err != nil {
		return fmt.Errorf("creating client listening for load event: %v", err)
	}
	defer loadEventFiredClient.Close()
	// tell the page to navigate to the URL.
	navArgs := page.NewNavigateArgs(html.URL)
	_, err = c.Page.Navigate(html.Context, navArgs)
	if err != nil {
		return fmt.Errorf("%s: navigating to URL: %v", html.URL, err)
	}
	// wait for the page to finish loading.
	_, err = loadEventFiredClient.Recv()
	if err != nil {
		return fmt.Errorf("waiting for page loading: %v", err)
	}
	// inject a script to make sure web fonts are loaded.
	script := `new Promise((resolve, reject) => {
		document.fonts.ready.then(function () {
			resolve('fonts loaded');
		});
		setTimeout(resolve.bind(resolve, 'timeout'), 500);
	});`
	scriptArg := runtime.NewEvaluateArgs(script).SetAwaitPromise(true)
	returnObj, _ := c.Runtime.Evaluate(html.Context, scriptArg)
	if returnObj.ExceptionDetails != nil {
		return fmt.Errorf("script evaluated with exception: %+v", returnObj.ExceptionDetails)
	}
	loadFontsResult := string(returnObj.Result.Value)
	if strings.Contains(loadFontsResult, "timeout") {
		return errors.New("timed out loading fonts")
	}
	// if no header or footer, use the default template
	// for avoiding displaying default Chrome templates.
	if html.HeaderHTML == "" {
		html.HeaderHTML = defaultHeaderFooterHTML
	}
	if html.FooterHTML == "" {
		html.FooterHTML = defaultHeaderFooterHTML
	}
	print, err := c.Page.PrintToPDF(
		html.Context,
		page.NewPrintToPDFArgs().
			SetPaperWidth(html.PaperWidth).
			SetPaperHeight(html.PaperHeight).
			SetMarginTop(html.MarginTop).
			SetMarginBottom(html.MarginBottom).
			SetMarginLeft(html.MarginLeft).
			SetMarginRight(html.MarginRight).
			SetLandscape(html.Landscape).
			SetDisplayHeaderFooter(true).
			SetHeaderTemplate(html.HeaderHTML).
			SetFooterTemplate(html.FooterHTML).
			SetPrintBackground(true),
	)
	if err != nil {
		return fmt.Errorf("printing page to PDF: %v", err)
	}
	return writeBytesToFile(destination, print.Data)
}

// WithLocalURL sets a local URL from a file path.
func (html *HTML) WithLocalURL(fpath string) {
	html.URL = fmt.Sprintf("file://%s", fpath)
}

// WithHeaderFile sets header content from a file.
func (html *HTML) WithHeaderFile(fpath string) error {
	if fpath == "" {
		return nil
	}
	contentHTML, err := fileContentToString(fpath)
	if err != nil {
		return err
	}
	html.HeaderHTML = contentHTML
	return nil
}

// WithFooterFile sets footer content from a file.
func (html *HTML) WithFooterFile(fpath string) error {
	if fpath == "" {
		return nil
	}
	contentHTML, err := fileContentToString(fpath)
	if err != nil {
		return err
	}
	html.FooterHTML = contentHTML
	return nil
}

func fileContentToString(fpath string) (string, error) {
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", fmt.Errorf("%s: reading file: %v", fpath, err)
	}
	return string(b), nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(HTML))
)
