package gotenbergcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// telemetryShutdownTimeout bounds the final flush of spans, metrics, and logs.
const telemetryShutdownTimeout = 5 * time.Second

// Version is the... version of the Gotenberg application. We set it at the
// build stage of the Docker image.
var Version = "snapshot"

// Run starts the Gotenberg application. Call this in the main of your program.
func Run() {
	startedAt := time.Now()
	gotenberg.Version = Version

	// Create the root FlagSet and adds the modules flags to it.
	fs := flag.NewFlagSet("gotenberg", flag.ExitOnError)
	fs.Bool("gotenberg-hide-banner", false, "Hide the ASCII art banner above the startup summary")
	fs.Duration("gotenberg-graceful-shutdown-duration", time.Duration(30)*time.Second, "Set the graceful shutdown duration")
	fs.Bool("gotenberg-build-debug-data", true, "Set if build data is needed")

	// Logging & telemetry flags.
	fs.String("log-level", gotenberg.InfoLoggingLevel, "Set the log level")
	fs.String("log-fields-prefix", "", "Prepend a specified prefix to each log field key")
	fs.String("log-std-format", gotenberg.AutoLoggingFormat, "Set the log format for standard output")
	fs.Bool("log-std-enable-gcp-fields", false, "Use GCP-compatible field names in log output")
	fs.String("log-std-level-case", gotenberg.LowerLevelCase, "Set the case of the level field in the standard output, either lower or upper")

	descriptors := gotenberg.GetModuleDescriptors()
	for _, desc := range descriptors {
		fs.AddFlagSet(desc.FlagSet)
	}

	// Parse the flags.
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Override their values if the corresponding environment variables are
	// set.
	err = gotenberg.ApplyEnvOverrides(fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] %v\n", err)
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
		fmt.Fprintf(os.Stderr, "[FATAL] invalid telemetry config: %s\n", err)
		os.Exit(1)
	}

	shutdownTelemetry, err := gotenberg.StartTelemetry(telemetryCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] start telemetry: %s\n", err)
		os.Exit(1)
	}

	logger := gotenberg.ProcessLogger()
	bgCtx := context.Background()

	fatal := func(msg string, err error) {
		logger.ErrorContext(bgCtx, msg, slog.Any("error", err))
		os.Exit(1)
	}

	ctx := gotenberg.NewContext(parsedFlags, descriptors)

	// Start application modules. The API waits for the others to be ready
	// before it listens, so they start together.
	apps, err := ctx.Modules(new(gotenberg.App))
	if err != nil {
		fatal("load modules", err)
	}

	eg := new(errgroup.Group)
	for _, a := range apps {
		eg.Go(func() error {
			err := a.(gotenberg.App).Start()
			if err != nil {
				return fmt.Errorf("start %s: %w", a.(gotenberg.Module).Descriptor().ID, err)
			}

			return nil
		})
	}

	err = eg.Wait()
	if err != nil {
		fatal("start modules", err)
	}

	startup, err := gotenberg.NewStartup(ctx, Version, time.Since(startedAt))
	if err != nil {
		fatal("gather startup summary", err)
	}

	if telemetryCfg.Interactive() {
		// See https://no-color.org.
		err = startup.Write(os.Stdout, !hideBanner, os.Getenv("NO_COLOR") == "")
		if err != nil {
			fatal("print startup summary", err)
		}
	} else {
		startup.Log(bgCtx, logger)
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

	logger.InfoContext(bgCtx, "graceful shutdown", slog.String("duration", gracefulShutdownDuration.String()))

	stoppable := make([]gotenberg.App, len(apps))
	for i, a := range apps {
		stoppable[i] = a.(gotenberg.App)
	}

	err = gotenberg.StopApps(gracefulShutdownCtx, stoppable, func(app gotenberg.App) {
		logger.InfoContext(bgCtx, "module stopped", slog.String("module", app.(gotenberg.Module).Descriptor().ID))
	})
	if err != nil {
		cancel()
		fatal("stop modules", err)
	}

	// Shutdown telemetry (flush spans, metrics, logs). The graceful shutdown's
	// context may already be done, by its deadline or a forced quit, so the
	// flush gets its own.
	telemetryCtx, telemetryCancel := context.WithTimeout(context.Background(), telemetryShutdownTimeout)
	defer telemetryCancel()

	err = shutdownTelemetry(telemetryCtx)
	if err != nil {
		cancel()
		// The telemetry, logs included, is down: write to stderr.
		fmt.Fprintf(os.Stderr, "[FATAL] shut down telemetry: %v\n", err)
		os.Exit(1) //nolint:gocritic // defers are already called explicitly above
	}

	os.Exit(0)
}
