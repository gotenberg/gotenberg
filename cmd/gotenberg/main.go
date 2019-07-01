package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/app/api"
	conf "github.com/thecodingmachine/gotenberg/internal/pkg/config"
	log "github.com/thecodingmachine/gotenberg/internal/pkg/logger"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
)

// version will be set on build time.
// nolint: gochecknoglobals
var version = "snapshot"

func main() {
	config, err := conf.FromEnv()
	systemLogger := log.New(config.LogLevel(), "system")
	if err != nil {
		systemLogger.Fatal(err)
	}
	systemLogger.Infof("Gotenberg %s", version)
	// start PM2 processes.
	var processes []pm2.Process
	if config.EnableChromeEndpoints() {
		processes = append(processes, pm2.NewChrome(systemLogger))
	}
	if config.EnableUnoconvEndpoints() {
		processes = append(processes, pm2.NewUnoconv(systemLogger))
	}
	for _, p := range processes {
		systemLogger.Infof("starting %s with PM2...", p.Fullname())
		if err := p.Start(); err != nil {
			systemLogger.Fatal(err)
		}
	}
	// run our API in a goroutine so that it doesn't block.
	// create our API.
	srv := api.New(config)
	go func() {
		systemLogger.Infof("http server started on port %s", config.DefaultListenPort())
		if err := srv.Start(fmt.Sprintf(":%s", config.DefaultListenPort())); err != nil {
			if err != http.ErrServerClosed {
				systemLogger.Fatal(err)
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
	systemLogger.Info("shutting down http server...")
	if err := srv.Shutdown(ctx); err != nil {
		systemLogger.Fatal(err)
	}
	// shutdown PM2 processes.
	for _, p := range processes {
		systemLogger.Infof("shutting down %s with PM2...", p.Fullname())
		if err := p.Shutdown(); err != nil {
			systemLogger.Fatal(err)
		}
	}
	systemLogger.Info("bye!")
	os.Exit(0)
}
