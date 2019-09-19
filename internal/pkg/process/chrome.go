package process

import (
	"context"
	"fmt"
	"time"

	"github.com/mafredri/cdp/devtool"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

const ChromeKey Key = "chrome"

type chromeProcess struct {
	id   string
	host string
	port int
}

// NewChromeProcess returns a Google Chrome
// headless process.
func NewChromeProcess(id, host string, port int) Process {
	return chromeProcess{
		id:   id,
		host: host,
		port: port,
	}
}

func (p chromeProcess) ID() string {
	return p.id
}

func (p chromeProcess) Host() string {
	return p.host
}

func (p chromeProcess) Port() int {
	return p.port
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
		fmt.Sprintf("--remote-debugging-port=%d", p.port),
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
	const op string = "process.chromeProcess.viabilityFunc"
	return func(logger xlog.Logger) bool {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		endpoint := fmt.Sprintf("http://%s:%d" /*p.host*/, "localhost", p.port)
		logger.DebugfOp(
			op,
			"checking '%s' viability via endpoint '%s/json/version'",
			p.ID(),
			endpoint,
		)
		v, err := devtool.New(endpoint).Version(ctx)
		if err != nil {
			logger.ErrorfOp(
				op,
				"'%s' is not viable as endpoint returned '%v'",
				p.ID(),
				err,
			)
			return false
		}
		logger.DebugfOp(
			op,
			"'%s' is viable as endpoint returned '%v'",
			p.ID(),
			v,
		)
		return true
	}
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(chromeProcess))
)
