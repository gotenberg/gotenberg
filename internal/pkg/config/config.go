package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

const (
	defaultWaitTimeoutEnvVar  = "DEFAULT_WAIT_TIMEOUT"
	defaultListenPortEnvVar   = "DEFAULT_LISTEN_PORT"
	disableGoogleChromeEnvVar = "DISABLE_GOOGLE_CHROME"
	disableUnoconvEnvVar      = "DISABLE_UNOCONV"
	logLevelEnvVar            = "LOG_LEVEL"
)

// Config contains the application
// configuration.
type Config struct {
	defaultWaitTimeout     float64
	defaultListenPort      string
	enableChromeEndpoints  bool
	enableUnoconvEndpoints bool
	logLevel               logrus.Level
}

func defaultConfig() *Config {
	return &Config{
		defaultWaitTimeout:     10,
		defaultListenPort:      "3000",
		enableChromeEndpoints:  true,
		enableUnoconvEndpoints: true,
		logLevel:               logrus.InfoLevel,
	}
}

// FromEnv fetches configuration
// from environment variables.
func FromEnv() (*Config, error) {
	const op = "config.FromEnv"
	c := defaultConfig()
	defaultWaitTimeout, err := defaultWaitTimeoutFromEnv(defaultWaitTimeoutEnvVar, c.DefaultWaitTimeout())
	c.defaultWaitTimeout = defaultWaitTimeout
	if err != nil {
		return c, &standarderror.Error{Op: op, Err: err}
	}
	defaultListenPort, err := defaultListenPortFromEnv(defaultListenPortEnvVar, c.DefaultListenPort())
	c.defaultListenPort = defaultListenPort
	if err != nil {
		return c, &standarderror.Error{Op: op, Err: err}
	}
	disableChromeEndpoints, err := boolFromEnv(disableGoogleChromeEnvVar, !c.EnableChromeEndpoints())
	c.enableChromeEndpoints = !disableChromeEndpoints
	if err != nil {
		return c, &standarderror.Error{Op: op, Err: err}
	}
	disableUnoconvEndpoints, err := boolFromEnv(disableUnoconvEnvVar, !c.EnableUnoconvEndpoints())
	c.enableUnoconvEndpoints = !disableUnoconvEndpoints
	if err != nil {
		return c, &standarderror.Error{Op: op, Err: err}
	}
	logLevel, err := logLevelFromEnv(logLevelEnvVar, c.LogLevel())
	c.logLevel = logLevel
	if err != nil {
		return c, &standarderror.Error{Op: op, Err: err}
	}
	return c, nil
}

// DefaultWaitTimeout returns the default
// wait timeout from the configuration.
func (c *Config) DefaultWaitTimeout() float64 {
	return c.defaultWaitTimeout
}

// DefaultListenPort returns the default
// listen port from the configuration.
func (c *Config) DefaultListenPort() string {
	return c.defaultListenPort
}

// EnableChromeEndpoints returns true if
// Chrome endpoints are enabled in the
// configuration.
func (c *Config) EnableChromeEndpoints() bool {
	return c.enableChromeEndpoints
}

// EnableUnoconvEndpoints returns true if
// Unoconv endpoints are enabled in the
// configuration.
func (c *Config) EnableUnoconvEndpoints() bool {
	return c.enableUnoconvEndpoints
}

// LogLevel returns the logrus.Level from
// the configuration.
func (c *Config) LogLevel() logrus.Level {
	return c.logLevel
}

func defaultWaitTimeoutFromEnv(envVar string, defaultValue float64) (float64, error) {
	const op = "config.defaultWaitTimeoutFromEnv"
	if v, ok := os.LookupEnv(envVar); ok {
		waitTimeout, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return defaultValue, &standarderror.Error{
				Code:    standarderror.Invalid,
				Message: fmt.Sprintf("'%s' is not a float, got '%s'", envVar, v),
				Op:      op,
			}
		}
		return waitTimeout, nil
	}
	return defaultValue, nil
}

func defaultListenPortFromEnv(envVar string, defaultValue string) (string, error) {
	const op = "config.defaultListenPortFromEnv"
	if v, ok := os.LookupEnv(envVar); ok {
		portAsUint, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return defaultValue, &standarderror.Error{
				Code:    standarderror.Invalid,
				Message: fmt.Sprintf("'%s' is not a uint, got '%s'", envVar, v),
				Op:      op,
			}
		}
		if portAsUint > 65535 {
			return defaultValue, &standarderror.Error{
				Code:    standarderror.Invalid,
				Message: fmt.Sprintf("'%s' is not a uint < 65535, got '%d'", envVar, portAsUint),
				Op:      op,
			}
		}
		return v, nil
	}
	return defaultValue, nil
}

func boolFromEnv(envVar string, defaultValue bool) (bool, error) {
	const op = "config.boolFromEnv"
	if v, ok := os.LookupEnv(envVar); ok {
		if v != "1" && v != "0" {
			return defaultValue, &standarderror.Error{
				Code:    standarderror.Invalid,
				Message: fmt.Sprintf("'%s' is not '0' or '1', got %s", envVar, v),
				Op:      op,
			}
		}
		return v == "1", nil
	}
	return defaultValue, nil
}

func logLevelFromEnv(envVar string, defaultValue logrus.Level) (logrus.Level, error) {
	const op = "config.logLevelFromEnv"
	if v, ok := os.LookupEnv(envVar); ok {
		switch v {
		case "DEBUG":
			return logrus.DebugLevel, nil
		case "INFO":
			return logrus.InfoLevel, nil
		case "ERROR":
			return logrus.ErrorLevel, nil
		default:
			return defaultValue, &standarderror.Error{
				Code:    standarderror.Invalid,
				Message: fmt.Sprintf("'%s' is not 'DEBUG', 'INFO' or 'ERROR', got '%s'", envVar, v),
				Op:      op,
			}
		}
	}
	return defaultValue, nil
}
