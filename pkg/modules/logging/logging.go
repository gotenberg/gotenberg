package logging

import (
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"

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

// Logging is a module which implements the [gotenberg.LoggerProvider]
// interface.
type Logging struct {
	level        string
	format       string
	fieldsPrefix string
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
		lvl, err := newLogLevel(log.level)
		if err != nil {
			return nil, fmt.Errorf("get log level: %w", err)
		}

		encoder, err := newLogEncoder(log.format)
		if err != nil {
			return nil, fmt.Errorf("get log encoder: %w", err)
		}

		logger = zap.New(customCore{
			Core:         zapcore.NewCore(encoder, os.Stderr, lvl),
			fieldsPrefix: log.fieldsPrefix,
		})
	}

	return logger.Named(mod.Descriptor().ID), nil
}

// See https://github.com/gotenberg/gotenberg/issues/659.
type customCore struct {
	zapcore.Core
	fieldsPrefix string
}

func (c customCore) With(fields []zapcore.Field) zapcore.Core {
	if c.fieldsPrefix != "" {
		for i := range fields {
			fields[i].Key = c.fieldsPrefix + "_" + fields[i].Key
		}
	}

	return customCore{
		Core:         c.Core.With(fields),
		fieldsPrefix: c.fieldsPrefix,
	}
}

func (c customCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// This is a copy from the zapcore.ioCore implementation. Indeed, by doing
	// so, we are able to prefix the fields given to the logger methods like
	// Debug, Info, Warn, Error, etc.
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}

func (c customCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if c.fieldsPrefix != "" {
		for i := range fields {
			fields[i].Key = c.fieldsPrefix + "_" + fields[i].Key
		}
	}

	return c.Core.Write(entry, fields)
}

func newLogLevel(level string) (zapcore.Level, error) {
	lvl := zapcore.InvalidLevel

	err := lvl.UnmarshalText([]byte(level))
	if err != nil {
		return lvl, fmt.Errorf("%q is not a recognized log level: %w", level, err)
	}

	return lvl, nil
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

// Singleton so that we instantiate our logger only once.
var logger *zap.Logger = nil

// Interface guards.
var (
	_ gotenberg.Module         = (*Logging)(nil)
	_ gotenberg.Provisioner    = (*Logging)(nil)
	_ gotenberg.Validator      = (*Logging)(nil)
	_ gotenberg.LoggerProvider = (*Logging)(nil)
	_ zapcore.Core             = (*customCore)(nil)
)
