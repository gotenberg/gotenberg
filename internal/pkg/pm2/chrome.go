package pm2

import (
	"context"
	"time"

	"github.com/mafredri/cdp/devtool"
)

const (
	warmupTime = 10 * time.Second
)

type chrome struct {
	manager *processManager
}

// NewChrome retruns a Google Chrome
// headless process.
func NewChrome(debug bool) Process {
	return &chrome{
		manager: &processManager{verbose: debug},
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
	p.manager.notifyf(`%v: checking Chrome liveness via debug version endpoint 
'http://localhost:9222/json/version'`, p.name())
	v, err := devtool.New("http://localhost:9222").Version(ctx)
	if err != nil {
		p.manager.notifyf("%v: Chrome version endpoint returned error: %v", p.name(), err)
		return false
	}
	p.manager.notifyf("%v: Chrome returned version info: %+v", p.name(), *v)
	return true
}

func (p *chrome) warmup() {
	p.manager.notifyf("%v: allowing Chrome %v to startup", p.name(), warmupTime)
	time.Sleep(warmupTime)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(chrome))
)
