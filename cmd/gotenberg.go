package gotenbergcmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// See https://patorjk.com/software/taag/#p=display&f=Small%20Slant&t=Gotenberg.
// Credits: https://github.com/labstack/echo/blob/v4.3.0/echo.go#L240.
const banner = `
  _____     __           __               
 / ___/__  / /____ ___  / /  ___ _______ _
/ (_ / _ \/ __/ -_) _ \/ _ \/ -_) __/ _ '/
\___/\___/\__/\__/_//_/_.__/\__/_/  \_, / 
                                   /___/

A Docker-based API for converting documents to PDF.
Version: %s
-------------------------------------------------------
`

// telemetryShutdownTimeout bounds the final flush of spans, metrics, and logs.
const telemetryShutdownTimeout = 5 * time.Second

// Version is the... version of the Gotenberg application. We set it at the
// build stage of the Docker image.
var Version = "snapshot"

// Run starts the Gotenberg application. Call this in the main of your program.
func Run() {
	gotenberg.Version = Version

	// Create the root FlagSet and adds the modules flags to it.
	fs := flag.NewFlagSet("gotenberg", flag.ExitOnError)
	fs.Bool("gotenberg-hide-banner", false, "Hide the banner")
	fs.Duration("gotenberg-graceful-shutdown-duration", time.Duration(30)*time.Second, "Set the graceful shutdown duration")
	fs.Bool("gotenberg-build-debug-data", true, "Set if build data is needed")

	// Logging & telemetry flags.
	fs.String("log-level", gotenberg.InfoLoggingLevel, "Set the log level")
	fs.String("log-fields-prefix", "", "Prepend a specified prefix to each log field key")
	fs.String("log-std-format", gotenberg.AutoLoggingFormat, "Set the log format for standard output")
	fs.Bool("log-std-enable-gcp-fields", false, "Use GCP-compatible field names in log output")
	fs.String("log-std-level-case", gotenberg.LowerLevelCase, "Set the case of the level field in the standard output, either lower or upper")

	descriptors := gotenberg.GetModuleDescriptors()
	var modsInfo strings.Builder
	for _, desc := range descriptors {
		fs.AddFlagSet(desc.FlagSet)
		modsInfo.WriteString(desc.ID + " ")
	}

	// Parse the flags.
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Override their values if the corresponding environment variables are
	// set.
	err = gotenberg.ApplyEnvOverrides(fs)
	if err != nil {
		fmt.Printf("[FATAL] %v\n", err)
		os.Exit(1)
	}

	// Create a wrapper around our flags.
	parsedFlags := gotenberg.ParsedFlags{FlagSet: fs}
	hideBanner := parsedFlags.MustBool("gotenberg-hide-banner")
	gracefulShutdownDuration := parsedFlags.MustDuration("gotenberg-graceful-shutdown-duration")

	// Initialize telemetry (logging + OTEL).
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "gotenberg"
	}

	telemetryCfg := gotenberg.TelemetryConfig{
		ServiceName:           serviceName,
		ServiceVersion:        Version,
		LogLevel:              parsedFlags.MustString("log-level"),
		LogFieldsPrefix:       parsedFlags.MustString("log-fields-prefix"),
		LogStdFormat:          parsedFlags.MustString("log-std-format"),
		LogStdEnableGcpFields: parsedFlags.MustBool("log-std-enable-gcp-fields"),
		LogStdLevelCase:       parsedFlags.MustString("log-std-level-case"),
	}

	err = telemetryCfg.Validate()
	if err != nil {
		fmt.Printf("[FATAL] invalid telemetry config: %s\n", err)
		os.Exit(1)
	}

	shutdownTelemetry, err := gotenberg.StartTelemetry(telemetryCfg)
	if err != nil {
		fmt.Printf("[FATAL] start telemetry: %s\n", err)
		os.Exit(1)
	}

	if !hideBanner {
		fmt.Printf(banner, Version)
	}
	fmt.Printf("[SYSTEM] modules: %s\n", modsInfo.String())

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
			err := app.Start()
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

	if parsedFlags.MustBool("gotenberg-build-debug-data") {
		// Build the debug data.
		gotenberg.BuildDebug(ctx)
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

	stoppable := make([]gotenberg.App, len(apps))
	for i, a := range apps {
		stoppable[i] = a.(gotenberg.App)
	}

	err = gotenberg.StopApps(gracefulShutdownCtx, stoppable, func(app gotenberg.App) {
		fmt.Printf("[SYSTEM] %s: application stopped\n", app.(gotenberg.Module).Descriptor().ID)
	})
	if err != nil {
		cancel()
		fmt.Printf("[FATAL] %v\n", err)
		os.Exit(1) //nolint:gocritic // defers are already called explicitly above
	}

	// Shutdown telemetry (flush spans, metrics, logs). The graceful shutdown's
	// context may already be done, by its deadline or a forced quit, so the
	// flush gets its own.
	telemetryCtx, telemetryCancel := context.WithTimeout(context.Background(), telemetryShutdownTimeout)
	defer telemetryCancel()

	err = shutdownTelemetry(telemetryCtx)
	if err != nil {
		cancel()
		fmt.Printf("[FATAL] %v\n", err)
		os.Exit(1) //nolint:gocritic // defers are already called explicitly above
	}

	os.Exit(0)
}
