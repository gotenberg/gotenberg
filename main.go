/*
Package main handles the application startup and shutdown.

Gotenberg is a stateless API for converting Markdown files, HTML files and Office documents to PDF.

For more information, go to https://github.com/thecodingmachine/gotenberg.
*/
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/thecodingmachine/gotenberg/app"
	"github.com/thecodingmachine/gotenberg/app/config"
	"github.com/thecodingmachine/gotenberg/app/logger"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// version will be set on build time.
var version = "master"

// defaultConfigurationFilePath is our default configuration file to parse.
const defaultConfigurationFilePath = "gotenberg.yml"

// main initializes the application, starts it, and handles
// graceful shutdown.
func main() {
	err := config.ParseFile(defaultConfigurationFilePath)
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
		logger.Fatal(err)
		os.Exit(1)
	}

	// defines our application logging.
	logger.SetLevel(config.GetLogsLevel())
	logger.SetFormatter(config.GetLogsFormatter())

	// defines our application router.
	r := mux.NewRouter()
	r.Handle("/", app.GetHandlersChain()).Methods(http.MethodPost)

	// defines our server.
	s := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.GetPort()),
		Handler: r,
	}

	logger.Debugf("configuration loaded from file %s", defaultConfigurationFilePath)
	logger.Infof("starting Gotenberg version %s", version)
	logger.Infof("listening on port %s", config.GetPort())

	// runs our server in a goroutine so that it doesn't block.
	go func() {
		if err = s.ListenAndServe(); err != nil {
			logger.SetLevel(logrus.InfoLevel)
			logger.Panic(err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	// we'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(quit, os.Interrupt)

	// blocks until we receive our signal.
	<-quit

	// creates a deadline to wait for.
	var wait time.Duration
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	s.Shutdown(ctx)

	logger.SetLevel(logrus.InfoLevel)
	logger.Info("bye!")
	os.Exit(0)
}
