package pm2

import (
	"context"
	"time"

	"github.com/mafredri/cdp/devtool"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type chromeProcess struct {
	logger xlog.Logger
}

// NewChromeProcess returns a Google Chrome
// headless process.
func NewChromeProcess(logger xlog.Logger) Process {
	return chromeProcess{
		logger: logger,
	}
}

func (p chromeProcess) Fullname() string {
	return "Google Chrome headless"
}

func (p chromeProcess) Start() error {
	const op string = "pm2.chromeProcess.Start"
	if err := start(p.logger, p); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p chromeProcess) IsViable() bool {
	const op string = "pm2.chromeProcess.IsViable"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.logger.DebugfOp(
		op,
		"checking '%s' viability via endpoint '%s'",
		p.Fullname(),
		"http://localhost:9222/json/version",
	)
	v, err := devtool.New("http://localhost:9222").Version(ctx)
	if err != nil {
		p.logger.ErrorfOp(
			op,
			"'%s' is not viable as endpoint returned '%v'",
			p.Fullname(),
			err,
		)
		return false
	}
	p.logger.DebugfOp(
		op,
		"'%s' is viable as endpoint returned '%v'",
		p.Fullname(),
		v,
	)
	return true
}

func (p chromeProcess) Stop() error {
	const op string = "pm2.chromeProcess.Stop"
	if err := stop(p.logger, p); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p chromeProcess) args() []string {
	return []string{
		"--no-sandbox",
		"--headless",
		"--remote-debugging-port=9222",
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

func (p chromeProcess) binary() string {
	return "google-chrome-stable"
}

func (p chromeProcess) warmup() {
	const (
		op         string        = "pm2.chromeProcess.warmup"
		warmupTime time.Duration = 10 * time.Second
	)
	p.logger.DebugfOp(
		op,
		"waiting '%v' for allowing '%s' to warmup",
		warmupTime,
		p.Fullname(),
	)
	time.Sleep(warmupTime)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(chromeProcess))
)
