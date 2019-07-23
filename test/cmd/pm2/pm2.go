package main

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/xlogtest"
)

func main() {
	logger := xlogtest.DebugLogger()
	process := pm2.NewChromeProcess(logger)
	if err := process.Start(); err != nil {
		panic(err)
	}
	process = pm2.NewUnoconvProcess(logger)
	if err := process.Start(); err != nil {
		panic(err)
	}
}
