package logging

import (
	"fmt"
	"sync"

	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func init() {
	gotenberg.MustRegisterModule(new(Logging))
}

const (
	errorLoggingLevel = "error"
	warnLoggingLevel  = "warn"
	infoLoggingLevel  = "info"
	debugLoggingLevel = "debug"
)

const (
	autoLoggingFormat = "auto"
	jsonLoggingFormat = "json"
	textLoggingFormat = "text"
)

// Logging is a module that implements the [gotenberg.LoggerProvider]
// interface.
type Logging struct {
	level           string
	format          string
	fieldsPrefix    string
	enableGcpFields bool

	bridgeCore   *bridgeCore
	bridgeCoreMu sync.Mutex
}

// Descriptor returns a [Logging]'s module descriptor.
func (log *Logging) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "logging",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("logging", flag.ExitOnError)
			fs.String("log-level", infoLoggingLevel, fmt.Sprintf("Choose the level of logging detail. Options include %s, %s, %s, or %s", errorLoggingLevel, warnLoggingLevel, infoLoggingLevel, debugLoggingLevel))
			fs.String("log-format", autoLoggingFormat, fmt.Sprintf("Specify the format of logging. Options include %s, %s, or %s", autoLoggingFormat, jsonLoggingFormat, textLoggingFormat))
			fs.String("log-fields-prefix", "", "Prepend a specified prefix to each field in the logs")
			fs.Bool("log-enable-gcp-fields", false, "Enable Google Cloud Platform fields - namely: time, message, severity")

			// Deprecated flags.
			fs.Bool("log-enable-gcp-severity", false, "Enable Google Cloud Platform severity mapping")
			err := fs.MarkDeprecated("log-enable-gcp-severity", "use log-enable-gcp-fields instead")
			if err != nil {
				panic(err)
			}

			return fs
		}(),
		New: func() gotenberg.Module { return new(Logging) },
	}
}

// Provision sets the log level and format.
func (log *Logging) Provision(ctx *gotenberg.Context) error {
	flags := ctx.ParsedFlags()
	log.level = flags.MustString("log-level")
	log.format = flags.MustString("log-format")
	log.fieldsPrefix = flags.MustString("log-fields-prefix")
	log.enableGcpFields = flags.MustDeprecatedBool("log-enable-gcp-severity", "log-enable-gcp-fields")

	return nil
}

// Validate validates the log level and format.
func (log *Logging) Validate() error {
	var err error

	switch log.level {
	case errorLoggingLevel, warnLoggingLevel, infoLoggingLevel, debugLoggingLevel:
		break
	default:
		err = multierr.Append(
			err,
			fmt.Errorf("log level must be either %s, %s, %s or %s", errorLoggingLevel, warnLoggingLevel, infoLoggingLevel, debugLoggingLevel),
		)
	}

	switch log.format {
	case autoLoggingFormat, jsonLoggingFormat, textLoggingFormat:
		break
	default:
		err = multierr.Append(
			err,
			fmt.Errorf("log format must be either %s, %s or %s", autoLoggingFormat, jsonLoggingFormat, textLoggingFormat),
		)
	}

	return err
}

// Logger returns a [zap.Logger].
func (log *Logging) Logger(mod gotenberg.Module) (*zap.Logger, error) {
	if logger == nil {
		level, err := newLogLevel(log.level)
		if err != nil {
			return nil, fmt.Errorf("get log level: %w", err)
		}

		stdCore, err := newStdCore(level, log.format, log.enableGcpFields)
		if err != nil {
			return nil, fmt.Errorf("create std core: %w", err)
		}

		log.bridgeCoreMu.Lock()
		log.bridgeCore = newBridgeCore(level)
		log.bridgeCoreMu.Unlock()

		teeCore := zapcore.NewTee(
			rootCore{
				Core: stdCore,
				// See https://github.com/gotenberg/gotenberg/issues/659.
				fieldsPrefix: log.fieldsPrefix,
			},
			rootCore{
				Core: log.bridgeCore,
				// See https://github.com/gotenberg/gotenberg/issues/659.
				fieldsPrefix: log.fieldsPrefix,
			},
		)

		logger = zap.New(teeCore)
	}

	return logger.Named(mod.Descriptor().ID), nil
}

// RegisterCore implements [gotenberg.LogExporterHook].
func (log *Logging) RegisterCore(core zapcore.Core) error {
	log.bridgeCoreMu.Lock()
	defer log.bridgeCoreMu.Unlock()

	if log.bridgeCore != nil {
		log.bridgeCore.SetTarget(core)
	}

	return nil
}

func newLogLevel(level string) (zapcore.Level, error) {
	lvl := zapcore.InvalidLevel

	err := lvl.UnmarshalText([]byte(level))
	if err != nil {
		return lvl, fmt.Errorf("%q is not a recognized log level: %w", level, err)
	}

	return lvl, nil
}

// Singleton so that we instantiate our logger only once.
var logger *zap.Logger = nil

// Interface guards.
var (
	_ gotenberg.Module          = (*Logging)(nil)
	_ gotenberg.Provisioner     = (*Logging)(nil)
	_ gotenberg.Validator       = (*Logging)(nil)
	_ gotenberg.LoggerProvider  = (*Logging)(nil)
	_ gotenberg.LogExporterHook = (*Logging)(nil)
)
