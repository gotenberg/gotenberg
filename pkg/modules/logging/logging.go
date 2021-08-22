package logging

import (
	"fmt"
	"os"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

func init() {
	gotenberg.MustRegisterModule(Logging{})
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

// Logging is a module which implements the gotenberg.LoggerProvider interface.
type Logging struct {
	level  string
	format string
}

// Descriptor returns a Logging's module descriptor.
func (Logging) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID: "logging",
		FlagSet: func() *flag.FlagSet {
			fs := flag.NewFlagSet("logging", flag.ExitOnError)
			fs.String("log-level", infoLoggingLevel, fmt.Sprintf("Set the log level - %s, %s, %s, or %s", errorLoggingLevel, warnLoggingLevel, infoLoggingLevel, debugLoggingLevel))
			fs.String("log-format", autoLoggingFormat, fmt.Sprintf("Set log format - %s, %s, or %s", autoLoggingFormat, jsonLoggingFormat, textLoggingFormat))

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

	return nil
}

// Validate validates the log level and format.
func (log Logging) Validate() error {
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

// Logger returns a zap.Logger.
func (log Logging) Logger(mod gotenberg.Module) (*zap.Logger, error) {
	if logger == nil {
		lvl, err := newLogLevel(log.level)
		if err != nil {
			return nil, fmt.Errorf("get log level: %w", err)
		}

		encoder, err := newLogEncoder(log.format)
		if err != nil {
			return nil, fmt.Errorf("get log encoder: %w", err)
		}

		core := zapcore.NewCore(encoder, os.Stderr, lvl)
		logger = zap.New(core)

		// nolint
		defer logger.Sync()
	}

	return logger.Named(mod.Descriptor().ID), nil
}

func newLogLevel(level string) (zapcore.Level, error) {
	switch level {
	case errorLoggingLevel:
		return zap.ErrorLevel, nil
	case warnLoggingLevel:
		return zap.WarnLevel, nil
	case infoLoggingLevel:
		return zap.InfoLevel, nil
	case debugLoggingLevel:
		return zap.DebugLevel, nil
	default:
		return -2, fmt.Errorf("%s is not a recognized log level", level)
	}
}

func newLogEncoder(format string) (zapcore.Encoder, error) {
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	encCfg := zap.NewProductionEncoderConfig()

	if isTerminal {
		// If interactive terminal, make output more human-readable by default.
		// Credits: https://github.com/caddyserver/caddy/blob/v2.1.1/logging.go#L671.
		encCfg.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
			encoder.AppendString(ts.UTC().Format("2006/01/02 15:04:05.000"))
		}

		if format == textLoggingFormat || format == autoLoggingFormat {
			encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	}

	if format == autoLoggingFormat && isTerminal {
		format = textLoggingFormat
	} else if format == autoLoggingFormat {
		format = jsonLoggingFormat
	}

	switch format {
	case textLoggingFormat:
		return zapcore.NewConsoleEncoder(encCfg), nil
	case jsonLoggingFormat:
		return zapcore.NewJSONEncoder(encCfg), nil
	default:
		return nil, fmt.Errorf("%s is not a recognized log format", format)
	}
}

var logger *zap.Logger = nil

// Interface guards.
var (
	_ gotenberg.Module         = (*Logging)(nil)
	_ gotenberg.Provisioner    = (*Logging)(nil)
	_ gotenberg.Validator      = (*Logging)(nil)
	_ gotenberg.LoggerProvider = (*Logging)(nil)
)
