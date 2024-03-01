package gotenbergcmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// See https://patorjk.com/software/taag/#p=display&f=Small%20Slant&t=Gotenberg.
// Credits: https://github.com/labstack/echo/blob/v4.3.0/echo.go#L240.
const banner = `
  _____     __           __               
 / ___/__  / /____ ___  / /  ___ _______ _
/ (_ / _ \/ __/ -_) _ \/ _ \/ -_) __/ _ '/
\___/\___/\__/\__/_//_/_.__/\__/_/  \_, / 
                                   /___/

A Docker-powered stateless API for PDF files.
Version: %s
-------------------------------------------------------
`

// Version is the... version of the Gotenberg application. We set it at the
// build stage of the Docker image.
var Version = "snapshot"

// Run starts the Gotenberg application. Call this in the main of your program.
func Run() {
	fmt.Printf(banner, Version)

	// Create the root FlagSet and adds the modules flags to it.
	fs := flag.NewFlagSet("gotenberg", flag.ExitOnError)
	fs.Duration("gotenberg-graceful-shutdown-duration", time.Duration(30)*time.Second, "Set the graceful shutdown duration")

	descriptors := gotenberg.GetModuleDescriptors()
	var modsInfo string
	for _, desc := range descriptors {
		fs.AddFlagSet(desc.FlagSet)
		modsInfo += desc.ID + " "
	}

	fmt.Printf("[SYSTEM] modules: %s\n", modsInfo)

	// Parse the flags...
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// ...and create a wrapper around those.
	parsedFlags := gotenberg.ParsedFlags{FlagSet: fs}

	// Get the graceful shutdown duration.
	gracefulShutdownDuration := parsedFlags.MustDuration("gotenberg-graceful-shutdown-duration")

	ctx := gotenberg.NewContext(parsedFlags, descriptors)

	// Start application modules.
	apps, err := ctx.Modules(new(gotenberg.App))
	if err != nil {
		fmt.Printf("[FATAL] %s\n", err)
		os.Exit(1)
	}

	for _, a := range apps {
		go func(app gotenberg.App) {
			id := app.(gotenberg.Module).Descriptor().ID
			err = app.Start()
			if err != nil {
				fmt.Printf("[FATAL] starting %s: %s\n", id, err)
				os.Exit(1)
			}

			startupMessage := app.StartupMessage()
			if startupMessage == "" {
				fmt.Printf("[SYSTEM] %s: application started\n", id)
				return
			}

			fmt.Printf("[SYSTEM] %s: %s\n", id, startupMessage)
		}(a.(gotenberg.App))
	}

	// Get modules that want to print system messages.
	sysLoggers, err := ctx.Modules(new(gotenberg.SystemLogger))
	if err != nil {
		fmt.Printf("[FATAL] %s\n", err)
		os.Exit(1)
	}

	for _, l := range sysLoggers {
		go func(logger gotenberg.SystemLogger) {
			id := logger.(gotenberg.Module).Descriptor().ID

			for _, message := range logger.SystemMessages() {
				fmt.Printf("[SYSTEM] %s: %s\n", id, message)
			}
		}(l.(gotenberg.SystemLogger))
	}

	quit := make(chan os.Signal, 1)

	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Kubernetes).
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-quit

	gracefulShutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownDuration)
	defer cancel()

	forceQuit := make(chan os.Signal, 1)
	signal.Notify(forceQuit, syscall.SIGINT)

	go func() {
		// In case of force quit, cancel the context.
		<-forceQuit
		cancel()
	}()

	fmt.Printf("[SYSTEM] graceful shutdown of %s\n", gracefulShutdownDuration)

	eg, _ := errgroup.WithContext(gracefulShutdownCtx)

	for _, a := range apps {
		eg.Go(func(app gotenberg.App) func() error {
			return func() error {
				id := app.(gotenberg.Module).Descriptor().ID

				err = app.Stop(gracefulShutdownCtx)
				if err != nil {
					return fmt.Errorf("stopping %s: %w", id, err)
				}

				fmt.Printf("[SYSTEM] %s: application stopped\n", id)
				return nil
			}
		}(a.(gotenberg.App)))
	}

	err = eg.Wait()
	if err != nil {
		fmt.Printf("[FATAL] %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
