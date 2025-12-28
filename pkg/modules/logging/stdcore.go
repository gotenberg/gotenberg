package logging

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

func newStdCore(level zapcore.LevelEnabler, format string, enableGcpFields bool) (zapcore.Core, error) {
	encoder, err := newLogEncoder(format, enableGcpFields)
	if err != nil {
		return nil, fmt.Errorf("get log encoder: %w", err)
	}

	return zapcore.NewCore(encoder, os.Stderr, level), nil
}

func newLogEncoder(format string, gcpFields bool) (zapcore.Encoder, error) {
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	encCfg := zap.NewProductionEncoderConfig()

	// Normalize the log format based on the output device.
	if format == autoLoggingFormat {
		if isTerminal {
			format = textLoggingFormat
		} else {
			format = jsonLoggingFormat
		}
	}

	// Use a human-readable time format if running in a terminal.
	if isTerminal {
		encCfg.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
			encoder.AppendString(ts.Local().Format("2006/01/02 15:04:05.000"))
		}
	}

	// Configure level encoding based on format and GCP settings.
	if format == textLoggingFormat && isTerminal {
		if gcpFields {
			encCfg.EncodeLevel = gcpSeverityColorEncoder
		} else {
			encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	}

	// For non-text (JSON) or when GCP fields are requested outside a terminal text output,
	// adjust the configuration to use GCP-specific field names and encoders.
	if gcpFields && format != textLoggingFormat {
		encCfg.EncodeLevel = gcpSeverityEncoder
		encCfg.TimeKey = "time"
		encCfg.LevelKey = "severity"
		encCfg.MessageKey = "message"
		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encCfg.EncodeDuration = zapcore.MillisDurationEncoder
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
