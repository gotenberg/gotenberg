/*
Package main handles the application startup and shutdown.

Gotenberg is a stateless API for generating PDF from many sources
(".html", ".doc", ".docx").

For more information, go to https://github.com/gulien/gotenberg.
*/
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gulien/gotenberg/config"
	"github.com/gulien/gotenberg/logger"
	"github.com/gulien/gotenberg/middlewares"

	"github.com/gorilla/mux"
)

// version will be set on build time.
var version = "master"

// init sets up the application configuration.
func init() {
	err := config.MakeConfig()
	if err != nil {
		logger.Log.Fatal(err)
		os.Exit(1)
	}

	logger.SetLevel(config.AppConfig.LogLevel)
	logger.Log.Infof("Gotenberg %s", version)
}

// main initializes the application and handles
// graceful shutdown.
func main() {
	r := mux.NewRouter()

	// defines our entry point.
	r.Handle("/", middlewares.GetMiddlewaresChain()).Methods("POST")

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", config.AppConfig.Port),
		// good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	// runs our server in a goroutine so that it doesn't block.
	go func() {
		logger.Log.Infof("Listening to port %s", config.AppConfig.Port)
		if err := srv.ListenAndServe(); err != nil {
			logger.Log.Panicf("Unrecoverable error: %s", err)
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
	srv.Shutdown(ctx)

	logger.Log.Info("Bye!")
	os.Exit(0)
}
