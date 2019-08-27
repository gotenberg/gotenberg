package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/thecodingmachine/gotenberg/internal/app/xhttp"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
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
	// start PM2 processes.
	var processes []pm2.Process
	if !config.DisableGoogleChrome() {
		processes = append(processes, pm2.NewChromeProcess(systemLogger))
	}
	/*if !config.DisableUnoconv() {
		processes = append(processes, pm2.NewUnoconvProcess(systemLogger))
	}*/
	for _, p := range processes {
		systemLogger.InfofOp(op, "starting '%s' with PM2...", p.Fullname())
		if err := p.Start(); err != nil {
			systemLogger.FatalOp(op, err)
		}
	}
	// create our API.
	srv := xhttp.New(config, processes...)
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
	// shutdown PM2 processes.
	for _, p := range processes {
		systemLogger.InfofOp(op, "shutting down '%s' with PM2...", p.Fullname())
		if err := p.Stop(); err != nil {
			systemLogger.FatalOp(op, err)
		}
	}
	systemLogger.InfoOp(op, "bye!")
	os.Exit(0)
}
