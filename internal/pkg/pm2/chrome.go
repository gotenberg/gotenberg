package pm2

import (
	"context"
	"time"

	"github.com/mafredri/cdp/devtool"
	"github.com/thecodingmachine/gotenberg/internal/pkg/logger"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

const chromeWarmupTime = 10 * time.Second

type chrome struct {
	manager *processManager
}

// NewChrome returns a Google Chrome
// headless process.
func NewChrome(logger *logger.Logger) Process {
	return &chrome{
		manager: &processManager{logger: logger},
	}
}

func (p *chrome) Fullname() string {
	return "Google Chrome headless"
}

func (p *chrome) Start() error {
	const op = "pm2.chrome.Start"
	if err := p.manager.start(p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

func (p *chrome) Shutdown() error {
	const op = "pm2.chrome.Shutdown"
	if err := p.manager.shutdown(p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

func (p *chrome) args() []string {
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

func (p *chrome) name() string {
	return "google-chrome-stable"
}

func (p *chrome) viable() bool {
	const op = "pm2.chrome.viable"
	// check if Google Chrome is correctly running.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.manager.logger.DebugfOp(
		op,
		"checking liveness via debug version endpoint http://localhost:9222/json/version",
	)
	v, err := devtool.New("http://localhost:9222").Version(ctx)
	if err != nil {
		p.manager.logger.DebugfOp(
			op,
			"debug version endpoint returned error: %v",
			err,
		)
		return false
	}
	p.manager.logger.DebugfOp(
		op,
		"debug version endpoint returned version info: %+v",
		*v,
	)
	return true
}

func (p *chrome) warmup() {
	const op = "pm2.chrome.warmup"
	p.manager.logger.DebugfOp(
		op,
		"allowing %v to startup",
		chromeWarmupTime,
	)
	time.Sleep(chromeWarmupTime)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(chrome))
)
