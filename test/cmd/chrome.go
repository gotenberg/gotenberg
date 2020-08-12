package main

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/chrome"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

func main() {
	const op string = "main"
	config, err := conf.FromEnv()
	systemLogger := xlog.New(config.LogLevel(), "system")
	if err != nil {
		systemLogger.FatalOp(op, err)
	}
	// start Google Chrome headless.
	if err := chrome.Start(systemLogger, config.GoogleChromeIgnoreCertificateErrors()); err != nil {
		systemLogger.FatalOp(op, err)
	}
}
