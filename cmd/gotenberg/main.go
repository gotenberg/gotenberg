package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/app/api"
	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

// version will be set on build time.
var version = "snapshot"

func main() {
	// TODO options
	srv := api.New(nil)
	// run our API in a goroutine so that it doesn't block.
	go func() {
		notify.Println(fmt.Sprintf("Gotenberg %s", version))
		notify.Println("http server started on port 3000")
		if err := srv.Start(":3000"); err != nil {
			notify.ErrPrintln(err)
			os.Exit(1)
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
	notify.Println("shutting down http server... (Ctrl+C to force)")
	if err := srv.Shutdown(ctx); err != nil {
		notify.ErrPrintln(err)
		os.Exit(1)
	}
	os.Exit(0)
}
