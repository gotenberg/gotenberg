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
type Chrome struct{}

// Launch starts Chrome headless with PM2.
func (c *Chrome) Launch() error {
	return launch(c)
}

// Shutdown stops Chrome headless and
// removes it from the list of PM2
// processes.
func (c *Chrome) Shutdown() error {
	return shutdown(c)
}

func (c *Chrome) getArgs() []string {
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

func (c *Chrome) getName() string {
	return "google-chrome-stable"
}

func (c *Chrome) getFullname() string {
	return "Chrome headless"
}

func (c *Chrome) isViable() bool {
	// check if Chrome is correctly running.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := devtool.New("http://localhost:9222").Version(ctx)
	return err == nil
}

func (c *Chrome) warmup() {
	notify.Println(fmt.Sprintf("warming-up %s", c.getFullname()))
	time.Sleep(5 * time.Second)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(Chrome))
)
