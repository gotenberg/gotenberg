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
	dsn                   string
	sendDefaultPii bool
	environment    string

	logger        *zap.Logger
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
			return fs
		}(),
		New: func() gotenberg.Module { return new(Sentry) },
	}
}

// Provision configures the Sentry module by reading flags and initializing the Sentry SDK.
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
	s.DSN = flags.MustString("sentry-dsn")
	s.SendDefaultPII = flags.MustBool("sentry-send-default-pii")
	s.Environment = flags.MustString("sentry-environment")
	s.isInitialized = false

	// If no DSN is provided, Sentry integration is disabled.
	if s.DSN == "" {
		return nil
	}

	// Prepare Sentry client options.
	initOpts := sentry.ClientOptions{
		Dsn:            s.DSN,
		SendDefaultPII: s.SendDefaultPII,
	}

	// Set environment if defined.
	if s.Environment != "" {
		initOpts.Environment = s.Environment
	}

	// Enable Sentry SDK debug logging if the application's log level is debug.
	if s.logger.Core().Enabled(zapcore.DebugLevel) {
		initOpts.Debug = true
	}

	// Initialize the Sentry SDK.
	if err = sentry.Init(initOpts); err != nil {
		s.initError = err // Store error for Validate()
		// Do not return the error here to allow the application to start even if Sentry fails.
		// Validate() will report this as a configuration error if DSN was set.
		return nil
	}

	s.isInitialized = true

	return nil
}

// Validate checks if the Sentry module was configured correctly.
func (s *Sentry) Validate() error {
	if s.initError != nil {
		// This error is reported if a DSN was provided but Sentry failed to initialize.
		return fmt.Errorf("Sentry configuration error: %w", s.initError)
	}
	// If DSN was empty, isInitialized is false and initError is nil (valid: Sentry disabled).
	// If DSN was present and init was successful, isInitialized is true and initError is nil (valid).
	return nil
}

// Start does nothing
func (s *Sentry) Start() error {
	err = sentry.Init(initOpts)
	if err != nil {
	// etc.
	}
}

// StartupMessage returns a custom startup message.
func (s *Sentry) StartupMessage() string {
	if s.isInitialized {
		return "initialized successfully"
	}
	return "integration is disabled - no DSN provided"
}

// Stop flushes any remaining events.
func (s *Sentry) Stop(ctx context.Context) error {
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)

	return nil
}

// Interface guards to ensure the module implements Gotenberg interfaces.
var (
	_ gotenberg.Module      = (*Sentry)(nil)
	_ gotenberg.Provisioner = (*Sentry)(nil)
	_ gotenberg.Validator   = (*Sentry)(nil)
	_ gotenberg.App         = (*Sentry)(nil)
)
