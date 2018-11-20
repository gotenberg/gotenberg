package process

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

// StartAll starts all processes.
func StartAll() error {
	if err := StartChromeHeadless(); err != nil {
		return err
	}
	return StartOfficeHeadless()
}

// StartChromeHeadless starts Chrome headless
// with PM2.
func StartChromeHeadless() error {
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
	time.Sleep(2 * time.Second)
	notify.Println("Chrome headless started with PM2")
	return nil
}

// StartOfficeHeadless starts soffice headless
// with PM2.
func StartOfficeHeadless() error {
	cmd := exec.Command(
		"pm2",
		"start",
		"soffice",
		"--headless",
		"--invisible",
		"--nocrashreport",
		"--nodefault",
		"--nofirststartwizard",
		"--nologo",
		"--norestore",
		"--accept=socket,host=127.0.0.1,port=2002,tcpNoDelay=1;urp;StarOffice.ComponentContext",
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting soffice headless with PM2: %v", err)
	}
	time.Sleep(2 * time.Second)
	notify.Println("soffice headless started with PM2")
	return nil
}
