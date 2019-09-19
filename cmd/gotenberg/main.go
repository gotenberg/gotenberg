package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/thecodingmachine/gotenberg/internal/app/xhttp"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/prinery"
	"github.com/thecodingmachine/gotenberg/internal/pkg/process"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xcontext"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

// version will be set on build time.
// nolint: gochecknoglobals
var version = "snapshot"

func main() {
	const op string = "main"
	config, err := conf.FromEnv()
	systemLogger := xlog.New(config.LogLevel(), "system")
	if err != nil {
		systemLogger.FatalOp(op, err)
	}
	systemLogger.InfofOp(op, "Gotenberg %s", version)
	systemLogger.DebugfOp(op, "configuration: %+v", config)
	// create PM2 manager and start processes.
	manager := process.NewPM2Manager(systemLogger, config)
	if err := manager.Start(); err != nil {
		systemLogger.FatalOp(op, err)
	}
	// create prineries.
	var chromePrinery *prinery.Prinery
	if !config.DisableGoogleChrome() {
		chromePrinery, err = prinery.New(systemLogger, manager, process.ChromeKey)
		if err != nil {
			systemLogger.FatalOp(op, err)
		}
		go chromePrinery.Start()
	}
	var sofficePrinery *prinery.Prinery
	if !config.DisableUnoconv() {
		sofficePrinery, err = prinery.New(systemLogger, manager, process.SofficeKey)
		if err != nil {
			systemLogger.FatalOp(op, err)
		}
		go sofficePrinery.Start()
	}
	// create our API.
	srv := xhttp.New(config, manager, chromePrinery, sofficePrinery)
	// run our API in a goroutine so that it doesn't block.
	go func() {
		systemLogger.InfofOp(op, "http server started on port '%d'", config.DefaultListenPort())
		if err := srv.Start(fmt.Sprintf(":%d", config.DefaultListenPort())); err != nil {
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
	ctx, cancel := xcontext.WithTimeout(systemLogger, 120)
	defer cancel()
	// doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	systemLogger.InfoOp(op, "shutting down http server...")
	if err := srv.Shutdown(ctx); err != nil {
		systemLogger.FatalOp(op, err)
	}
	systemLogger.InfoOp(op, "bye!")
	os.Exit(0)
}
