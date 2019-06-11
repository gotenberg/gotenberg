package pm2

import (
	"context"
	"time"

	"github.com/mafredri/cdp/devtool"
)

type chrome struct {
	manager *processManager
}

// NewChrome retruns a Google Chrome
// headless process.
func NewChrome() Process {
	return &chrome{
		manager: &processManager{},
	}
}

func (p *chrome) Fullname() string {
	return "Google Chrome headless"
}

func (p *chrome) Start() error {
	return p.manager.start(p)
}

func (p *chrome) Shutdown() error {
	return p.manager.shutdown(p)
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
	// check if Google Chrome is correctly running.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := devtool.New("http://localhost:9222").Version(ctx)
	return err == nil
}

func (p *chrome) warmup() {
	time.Sleep(5 * time.Second)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(chrome))
)
