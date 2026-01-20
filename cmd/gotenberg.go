package gotenbergcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
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

A containerized API for seamless PDF conversion.
Version: %s
-------------------------------------------------------
`

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
	fs.String("telemetry-service-name", "gotenberg", "Set the telemetry service name")
	fs.StringSlice("telemetry-trace-exporter-protocols", []string{}, fmt.Sprintf("Set the telemetry trace exporter protocols - leave empty to disable this feature. Option include only %s for now", gotenberg.GrpcTelemetryExporterProtocol))
	fs.StringSlice("telemetry-metric-exporter-protocols", []string{gotenberg.PrometheusTelemetryMetricExporterProtocol}, fmt.Sprintf("Set the telemetry metric exporter protocols - leave empty to disable this feature. Options include %s and %s", gotenberg.PrometheusTelemetryMetricExporterProtocol, gotenberg.GrpcTelemetryExporterProtocol))
	fs.StringSlice("telemetry-log-exporter-protocols", []string{}, fmt.Sprintf("Set the telemetry log exporter protocols - leave empty to disable this feature. Option include only %s for now", gotenberg.GrpcTelemetryExporterProtocol))
	fs.String("log-level", gotenberg.InfoLoggingLevel, fmt.Sprintf("Choose the level of logging detail. Options include %s, %s, %s, or %s", gotenberg.ErrorLoggingLevel, gotenberg.WarnLoggingLevel, gotenberg.InfoLoggingLevel, gotenberg.DebugLoggingLevel))
	fs.String("log-fields-prefix", "", "Prepend a specified prefix to each field in the logs")
	fs.String("log-std-format", gotenberg.AutoLoggingFormat, fmt.Sprintf("Specify the format of standard logging. Options include %s, %s, or %s", gotenberg.AutoLoggingFormat, gotenberg.JsonLoggingFormat, gotenberg.TextLoggingFormat))
	fs.Bool("log-std-enable-gcp-fields", false, "Enable Google Cloud Platform fields for standard logging - namely: time, message, severity")

	// Deprecated flags.
	fs.String("log-format", gotenberg.AutoLoggingFormat, fmt.Sprintf("Specify the format of logging. Options include %s, %s, or %s", gotenberg.AutoLoggingFormat, gotenberg.JsonLoggingFormat, gotenberg.TextLoggingFormat))
	fs.Bool("log-enable-gcp-fields", false, "Enable Google Cloud Platform fields - namely: time, message, severity")
	fs.Bool("log-enable-gcp-severity", false, "Enable Google Cloud Platform severity mapping")
	err := errors.Join(
		fs.MarkDeprecated("log-format", "use log-std-format"),
		fs.MarkDeprecated("log-enable-gcp-fields", "use log-std-enable-gcp-fields"),
		fs.MarkDeprecated("log-enable-gcp-severity", "use log-std-enable-gcp-fields"),
	)
	if err != nil {
		panic(err)
	}

	descriptors := gotenberg.GetModuleDescriptors()
	var modsInfo string
	for _, desc := range descriptors {
		fs.AddFlagSet(desc.FlagSet)
		modsInfo += desc.ID + " "
	}

	// Parse the flags.
	err = fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Override their values if the corresponding environment variables are
	// set.
	fs.VisitAll(func(f *flag.Flag) {
		envName := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		val, ok := os.LookupEnv(envName)
		if !ok {
			return
		}

		sliceVal, ok := f.Value.(flag.SliceValue)
		if ok {
			// We don't want to append the values (default pflag behavior).
			items := strings.Split(val, ",")
			err = sliceVal.Replace(items)
			if err != nil {
				fmt.Printf("[FATAL] invalid overriding value '%s' from %s: %v\n", val, envName, err)
				os.Exit(1)
			}
			return
		}

		err = f.Value.Set(val)
		if err != nil {
			fmt.Printf("[FATAL] invalid overriding value '%s' from %s: %v\n", val, envName, err)
			os.Exit(1)
		}
	})

	// Create a wrapper around our flags.
	parsedFlags := gotenberg.ParsedFlags{FlagSet: fs}
	hideBanner := parsedFlags.MustBool("gotenberg-hide-banner")
	gracefulShutdownDuration := parsedFlags.MustDuration("gotenberg-graceful-shutdown-duration")
	telemetryConfig := gotenberg.TelemetryConfig{
		// OpenTelemetry.
		ServiceName:             parsedFlags.MustString("telemetry-service-name"),
		ServiceVersion:          Version,
		TraceExporterProtocols:  parsedFlags.MustStringSlice("telemetry-trace-exporter-protocols"),
		MetricExporterProtocols: parsedFlags.MustStringSlice("telemetry-metric-exporter-protocols"),
		LogExporterProtocols:    parsedFlags.MustStringSlice("telemetry-log-exporter-protocols"),
		// Logging.
		LogLevel:              parsedFlags.MustString("log-level"),
		LogFieldsPrefix:       parsedFlags.MustString("log-fields-prefix"),
		LogStdFormat:          parsedFlags.MustDeprecatedString("log-format", "log-std-format"),
		LogStdEnableGcpFields: parsedFlags.MustDeprecatedBool("log-enable-gcp-fields", "log-std-enable-gcp-fields"),
	}

	if !hideBanner {
		fmt.Printf(banner, Version)
	}

	err = telemetryConfig.Validate()
	if err != nil {
		fmt.Printf("[FATAL] telemetry: %s\n", err)
		os.Exit(1)
	}

	// Telemetry.
	shutdownTelemetry, err := gotenberg.StartTelemetry(telemetryConfig)
	if err != nil {
		fmt.Printf("[FATAL] telemetry: %s\n", err)
		os.Exit(1)
	}
	printExporters := func(protocols []string) string {
		if len(protocols) == 0 {
			return "none"
		}
		return strings.Join(protocols, " ")
	}
	fmt.Printf("[SYSTEM] telemetry: trace exporter(s): %s\n", printExporters(telemetryConfig.TraceExporterProtocols))
	fmt.Printf("[SYSTEM] telemetry: metric exporter(s): %s\n", printExporters(telemetryConfig.MetricExporterProtocols))
	fmt.Printf("[SYSTEM] telemetry: log exporter(s): %s\n", printExporters(telemetryConfig.LogExporterProtocols))

	// Modules.
	fmt.Printf("[SYSTEM] modules: %s\n", modsInfo)
	ctx := gotenberg.NewContext(parsedFlags, descriptors)

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

	eg, _ := errgroup.WithContext(gracefulShutdownCtx)

	for _, a := range apps {
		eg.Go(func(app gotenberg.App) func() error {
			return func() error {
				id := app.(gotenberg.Module).Descriptor().ID

				err = app.Stop(gracefulShutdownCtx)
				if errors.Is(err, gotenberg.ErrCancelGracefulShutdownContext) {
					cancel()
				} else if err != nil {
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

	err = shutdownTelemetry(gracefulShutdownCtx)
	if err != nil {
		fmt.Printf("[FATAL] telemetry: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[SYSTEM] telemetry: shutdown success\n")

	os.Exit(0)
}
