package pm2

import (
	"context"

	"github.com/mafredri/cdp/devtool"
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
func (c *Chrome) Shutdown(delete bool) error {
	return shutdown(c, delete)
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
	devt := devtool.New("http://127.0.0.1:9222")
	_, err := devt.Create(context.TODO())
	return err == nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(Chrome))
)
