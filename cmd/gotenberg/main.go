package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/app/api"
	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
)

const (
	// version will be set on build time.
	version                   = "snapshort"
	waitTimeoutEnvVar         = "WAIT_TIMEOUT"
	disableGoogleChromeEnvVar = "DISABLE_GOOGLE_CHROME"
	disableUnoconvEnvVar      = "DISABLE_UNOCONV"
)

func mustStartProcess(p pm2.Process) {
	notify.Printf("starting %s with PM2...", p.Fullname())
	if err := p.Start(); err != nil {
		notify.ErrPrint(err)
		os.Exit(1)
	}
}

func startAPI(srv *api.API) {
	notify.Print("http server started on port 3000")
	if err := srv.Start(":3000"); err != nil {
		if err == http.ErrServerClosed {
			notify.WarnPrint(err)
		}
	}
}

func mustShutdownProcess(p pm2.Process) {
	notify.Printf("shutting down %s with PM2... (Ctrl+C to force)", p.Fullname())
	if err := p.Shutdown(); err != nil {
		notify.ErrPrint(err)
		os.Exit(1)
	}
}

func mustShutdownAPI(srv *api.API) {
	// create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	// doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	notify.Print("shutting down http server... (Ctrl+C to force)")
	if err := srv.Shutdown(ctx); err != nil {
		notify.ErrPrint(err)
		os.Exit(1)
	}
}

func main() {
	notify.Printf("Gotenberg %s", version)
	// TODO env var
	chrome := pm2.NewChrome()
	unoconv := pm2.NewUnoconv()
	srv := api.New(&api.Options{
		EnableChromeEndpoints:  true,
		EnableUnoconvEndpoints: true,
	})
	mustStartProcess(chrome)
	mustStartProcess(unoconv)
	// run our API in a goroutine so that it doesn't block.s
	go func() {
		startAPI(srv)
	}()
	quit := make(chan os.Signal, 1)
	// we'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(quit, os.Interrupt)
	// block until we receive our signal.
	<-quit
	mustShutdownAPI(srv)
	mustShutdownProcess(chrome)
	mustShutdownProcess(unoconv)
	notify.Print("bye!")
	os.Exit(0)
}
