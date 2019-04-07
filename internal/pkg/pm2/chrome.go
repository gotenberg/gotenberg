package pm2

import (
	"context"
	"fmt"
	"time"

	"github.com/mafredri/cdp/devtool"
	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

// Chrome facilitates starting or shutting down
// Chrome headless with PM2.
type Chrome struct {
	heuristicState int32
}

// Start starts Chrome headless with PM2.
func (c *Chrome) Start() error {
	return startProcess(c)
}

// Shutdown stops Chrome headless and
// removes it from the list of PM2
// processes.
func (c *Chrome) Shutdown() error {
	return shutdownProcess(c)
}

// State returns the current state of
// Chrome headless process.
func (c *Chrome) State() int32 {
	return c.heuristicState
}

func (c *Chrome) state(state int32) {
	c.heuristicState = state
}

func (c *Chrome) args() []string {
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

func (c *Chrome) name() string {
	return "google-chrome-stable"
}

func (c *Chrome) fullname() string {
	return "Chrome headless"
}

func (c *Chrome) viable() bool {
	// check if Chrome is correctly running.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := devtool.New("http://localhost:9222").Version(ctx)
	return err == nil
}

func (c *Chrome) warmup() {
	notify.Println(fmt.Sprintf("warming-up %s", c.fullname()))
	time.Sleep(5 * time.Second)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(Chrome))
)
