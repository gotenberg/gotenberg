package sentry

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(Sentry))
}

// Sentry is a module for Sentry.io integration.
type Sentry struct {
	dsn            string
	sendDefaultPii bool
	environment    string
	flushTimeout   time.Duration

	sentryClientOptions sentry.ClientOptions
	sentryInitialized   bool
	logger              *zap.Logger
}

// Descriptor returns a [Sentry]'s module descriptor.
func (s *Sentry) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "sentry",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("sentry", flag.ExitOnError)
			fs.String("sentry-dsn", "", "Sentry DSN. If empty, Sentry is disabled")
			fs.Bool("sentry-send-default-pii", false, "Enable sending of default PII to Sentry")
			fs.String("sentry-environment", "", "Sentry environment. If empty, the environment is not used")
			fs.Duration("sentry-flush-timeout", time.Duration(2)*time.Second, "Set the time limit (seconds) to wait for the underlying Sentry Transport to send any buffered events to Sentry server during Gotenberg stop")
			return fs
		}(),
		New: func() gotenberg.Module { return new(Sentry) },
	}
}

// Provision configures the Sentry module by reading flags.
func (s *Sentry) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()

	// Logger.
	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}

	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(s)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}

	s.logger = logger

	// Populate Sentry configuration from flags.
	s.dsn = flags.MustString("sentry-dsn")
	s.sendDefaultPii = flags.MustBool("sentry-send-default-pii")
	s.environment = flags.MustString("sentry-environment")
	s.flushTimeout = flags.MustDuration("sentry-flush-timeout")

	// If no dsn is provided, Sentry integration is disabled.
	if s.dsn == "" {
		return nil
	}

	// Prepare Sentry client options.
	s.sentryClientOptions = sentry.ClientOptions{
		Dsn:            s.dsn,
		SendDefaultPII: s.sendDefaultPii,
	}

	// Set environment if defined.
	if s.environment != "" {
		s.sentryClientOptions.Environment = s.environment
	}

	// Enable Sentry SDK debug logging if the application's log level is debug.
	if s.logger.Core().Enabled(zapcore.DebugLevel) {
		s.sentryClientOptions.Debug = true
	}

	// Used to track if Sentry client was initialized in Start() which allows Stop() to correctly flush any buffered events.
	s.sentryInitialized = false

	return nil
}

// Start initializing the Sentry SDK.
func (s *Sentry) Start() error {
	if s.sentryClientOptions.Dsn != "" {
		err := sentry.Init(s.sentryClientOptions)
		if err != nil {
			// This error is reported if a dsn was provided but Sentry failed to initialize.
			return fmt.Errorf("Sentry configuration error: %w", err)
		}

		s.sentryInitialized = true
	}

	return nil
}

// StartupMessage returns a custom startup message.
func (s *Sentry) StartupMessage() string {
	if s.sentryInitialized {
		return "initialized successfully"
	}
	return "integration is disabled - no DSN provided"
}

// Stop flushes any remaining events.
func (s *Sentry) Stop(ctx context.Context) error {
	// Flush buffered events before the program terminates, but only when Sentry client is initialized.
	// Set the timeout to the maximum duration the program can afford to wait.
	if s.sentryInitialized {
		defer sentry.Flush(s.flushTimeout)
	}

	return nil
}

// Interface guards to ensure the module implements Gotenberg interfaces.
var (
	_ gotenberg.Module      = (*Sentry)(nil)
	_ gotenberg.Provisioner = (*Sentry)(nil)
	_ gotenberg.App         = (*Sentry)(nil)
)
