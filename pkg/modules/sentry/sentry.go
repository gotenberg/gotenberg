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
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
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

// Debug returns additional debug data.
func (s *Sentry) Debug() map[string]interface{} {
	debug := make(map[string]interface{})

	debug["version"] = fmt.Sprintf("Sentry Go SDK Version: %s", sentry.SDKVersion)

	return debug
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
	if s.sentryInitialized {
		// Defer the flush operation. The timeout will be determined based on the context
		// when the deferred function executes.
		defer func(stopCtx context.Context) {
			// Start with the configured Sentry flush timeout as the default/upper bound.
			timeoutForFlush := s.flushTimeout

			// If the Stop context has a deadline, the flush must not exceed
			// the time remaining for that deadline.
			if deadline, ok := stopCtx.Deadline(); ok {
				remainingTimeForContext := time.Until(deadline)
				if remainingTimeForContext < timeoutForFlush {
					// Context offers less time than s.flushTimeout, so use the context's remaining time.
					timeoutForFlush = remainingTimeForContext
				}
			}

			// If the context is already marked as "done" (e.g., canceled, or deadline already passed),
			// or if the calculated timeout ended up being non-positive (deadline passed during Stop execution),
			// then use a very small positive timeout for a best-effort, non-blocking flush.
			if err := stopCtx.Err(); err != nil {
				s.logger.Debug("Sentry Stop: context is already done, using minimal flush timeout.", zap.Error(err))
				timeoutForFlush = 1 * time.Second // Minimal positive timeout
			} else if timeoutForFlush <= 0 {
				s.logger.Debug("Sentry Stop: calculated flush timeout is non-positive (deadline likely passed), using minimal flush timeout.")
				timeoutForFlush = 1 * time.Second // Minimal positive timeout
			}

			s.logger.Debug("Sentry: flushing events", zap.Duration("timeout", timeoutForFlush))
			sentry.Flush(timeoutForFlush)
		}(ctx) // Pass the context to the deferred function's closure
	}

	return nil
}

// Middlewares returns the Sentry middlewares.
func (s *Sentry) Middlewares() ([]api.Middleware, error) {
	// Check if middleware should be returned
	// Do not use s.sentryInitialized as that occurrs later in the bootstrapping process
	if s.sentryClientOptions.Dsn == "" {
		return nil, nil
	}
	return []api.Middleware{
		sentryPanicMiddleware(),        // Sets up Sentry hub, captures panics.
		sentryErrorCaptureMiddleware(), // Captures specific non-panic errors.
	}, nil
}

// Interface guards to ensure the module implements Gotenberg interfaces.
var (
	_ gotenberg.Module       = (*Sentry)(nil)
	_ gotenberg.Provisioner  = (*Sentry)(nil)
	_ gotenberg.App          = (*Sentry)(nil)
	_ api.MiddlewareProvider = (*Sentry)(nil)
	_ gotenberg.Debuggable   = (*Sentry)(nil)
)
