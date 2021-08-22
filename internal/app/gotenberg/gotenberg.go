package gotenberg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	flag "github.com/spf13/pflag"
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

var version = "snapshot"

func Run() {
	fmt.Printf(banner, version)

	// Creates the roo` FlagSet and adds the modules flags to it.
	fs := flag.NewFlagSet("gotenberg", flag.ExitOnError)
	fs.Duration("gotenberg-graceful-shutdown-duration", time.Duration(30)*time.Second, "Set the graceful shutdown duration")

	descriptors := gotenberg.GetModuleDescriptors()
	var modsInfo string
	for _, desc := range descriptors {
		fs.AddFlagSet(desc.FlagSet)
		modsInfo += desc.ID + " "
	}

	fmt.Printf("[SYSTEM] modules: %s\n", modsInfo)

	// Parses the flags...
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// ...and creates a wrapper around those.
	parsedFlags := gotenberg.ParsedFlags{FlagSet: fs}

	// Get the graceful shutdown duration.
	gracefulShutdownDuration := parsedFlags.MustDuration("gotenberg-graceful-shutdown-duration")

	ctx := gotenberg.NewContext(parsedFlags, descriptors)

	// Starts application modules.
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

	quit := make(chan os.Signal, 1)

	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C).
	signal.Notify(quit, os.Interrupt)

	// Block until we receive our signal.
	<-quit

	gracefulShutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownDuration)
	defer cancel()

	fmt.Printf("[SYSTEM] graceful shutdown of %s\n", gracefulShutdownDuration)

	for _, a := range apps {
		id := a.(gotenberg.Module).Descriptor().ID
		app := a.(gotenberg.App)

		err = app.Stop(gracefulShutdownCtx)
		if err != nil {
			fmt.Printf("[ERROR] stopping %s: %s\n", id, err)
		}

		fmt.Printf("[SYSTEM] %s: application stopped\n", id)
	}

	os.Exit(0)
}
