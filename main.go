/*
Package main handles the application startup and shutdown.

Gotenberg is a stateless API for generating PDF from many sources
(".html", ".doc", ".docx").

For more information, go to https://github.com/gulien/gotenberg.
*/
package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/gulien/gotenberg/app"
	"github.com/gulien/gotenberg/app/handlers/converter/process"
	"github.com/gulien/gotenberg/app/logger"

	"github.com/sirupsen/logrus"
)

// version will be set on build time.
var version = "master"

// main initializes the application and handles
// graceful shutdown.
func main() {
	a, err := app.NewApp(version)
	if err != nil {
		resetState()
		logger.Fatal(err)
		os.Exit(1)
	}

	// runs our server in a goroutine so that it doesn't block.
	go func() {
		if err = a.Run(); err != nil {
			resetState()
			logger.Panic(err)
			os.Exit(1)
		}
	}()

	c := make(chan os.Signal, 1)
	// we'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// blocks until we receive our signal.
	<-c

	// creates a deadline to wait for.
	var wait time.Duration
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	a.Server.Shutdown(ctx)
	resetState()

	logger.Info("Bye!")
	os.Exit(0)
}

func resetState() {
	logger.SetLevel(logrus.InfoLevel)
	process.Reset()
}
