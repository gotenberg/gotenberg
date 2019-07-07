package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/app/api"
	"github.com/thecodingmachine/gotenberg/internal/pkg/config"
	"github.com/thecodingmachine/gotenberg/internal/pkg/logger"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
)

// version will be set on build time.
// nolint: gochecknoglobals
var version = "snapshot"

func main() {
	const op = "main"
	config, err := config.FromEnv()
	systemLogger := logger.New(config.LogLevel(), "system")
	if err != nil {
		systemLogger.FatalOp(op, err)
	}
	systemLogger.InfofOp(op, "Gotenberg %s", version)
	// start PM2 processes.
	var processes []pm2.Process
	if config.EnableChromeEndpoints() {
		processes = append(processes, pm2.NewChrome(systemLogger))
	}
	if config.EnableUnoconvEndpoints() {
		processes = append(processes, pm2.NewUnoconv(systemLogger))
	}
	for _, p := range processes {
		systemLogger.InfofOp(op, "starting %s with PM2...", p.Fullname())
		if err := p.Start(); err != nil {
			systemLogger.FatalOp(op, err)
		}
	}
	// run our API in a goroutine so that it doesn't block.
	// create our API.
	srv := api.New(config)
	go func() {
		systemLogger.InfofOp(op, "http server started on port %s", config.DefaultListenPort())
		if err := srv.Start(fmt.Sprintf(":%s", config.DefaultListenPort())); err != nil {
			if err != http.ErrServerClosed {
				systemLogger.FatalOp(op, err)
			}
		}
	}()
	quit := make(chan os.Signal, 1)
	// we'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(quit, os.Interrupt)
	// block until we receive our signal.
	<-quit
	// create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	// doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	systemLogger.InfofOp(op, "shutting down http server...")
	if err := srv.Shutdown(ctx); err != nil {
		systemLogger.FatalOp(op, err)
	}
	// shutdown PM2 processes.
	for _, p := range processes {
		systemLogger.InfofOp(op, "shutting down %s with PM2...", p.Fullname())
		if err := p.Shutdown(); err != nil {
			systemLogger.FatalOp(op, err)
		}
	}
	systemLogger.InfofOp(op, "bye!")
	os.Exit(0)
}
