package main

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/test"
)

func main() {
	logger := test.DebugLogger()
	process := pm2.NewChromeProcess(logger)
	if err := process.Start(); err != nil {
		panic(err)
	}
	process = pm2.NewUnoconvProcess(logger)
	if err := process.Start(); err != nil {
		panic(err)
	}
}
