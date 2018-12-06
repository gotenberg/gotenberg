package process

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

// Start starts Chrome headless and
// unoconv listener with PM2.
func Start() error {
	if err := startChromeHeadless(); err != nil {
		return err
	}
	return startUnoconvListener()
}

func startChromeHeadless() error {
	cmd := exec.Command(
		"pm2",
		"start",
		"google-chrome-stable",
		"--interpreter none",
		"--",
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
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting Chrome headless with PM2: %v", err)
	}
	time.Sleep(4 * time.Second)
	notify.Println("Chrome headless started with PM2")
	return nil
}

func startUnoconvListener() error {
	cmd := exec.Command(
		"pm2",
		"start",
		"unoconv",
		"--interpreter none",
		"--",
		"--listener",
		"--verbose",
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting unoconv listener with PM2: %v", err)
	}
	time.Sleep(4 * time.Second)
	notify.Println("unoconv listener started with PM2")
	return nil
}
