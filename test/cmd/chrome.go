package main

import (
	"context"

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
	// start Google Chrome.
	_, err = chrome.Start(context.Background(), systemLogger)
	if err != nil {
		systemLogger.FatalOp(op, err)
	}
}
