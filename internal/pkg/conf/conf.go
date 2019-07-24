package conf

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/xassert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

const (
	maximumWaitTimeoutEnvVar       string = "MAXIMUM_WAIT_TIMEOUT"
	maximumWaitDelayEnvVar         string = "MAXIMUM_WAIT_DELAY"
	maximumWebhookURLTimeoutEnvVar string = "MAXIMUM_WEBHOOK_URL_TIMEOUT"
	defaultWaitTimeoutEnvVar       string = "DEFAULT_WAIT_TIMEOUT"
	defaultWebhookURLTimeoutEnvVar string = "DEFAULT_WEBHOOK_URL_TIMEOUT"
	defaultListenPortEnvVar        string = "DEFAULT_LISTEN_PORT"
	disableGoogleChromeEnvVar      string = "DISABLE_GOOGLE_CHROME"
	disableUnoconvEnvVar           string = "DISABLE_UNOCONV"
	logLevelEnvVar                 string = "LOG_LEVEL"
)

// Config contains the application
// configuration.
type Config struct {
	maximumWaitTimeout       float64
	maximumWaitDelay         float64
	maximumWebhookURLTimeout float64
	defaultWaitTimeout       float64
	defaultWebhookURLTimeout float64
	defaultListenPort        int64
	disableGoogleChrome      bool
	disableUnoconv           bool
	logLevel                 xlog.Level
}

// DefaultConfig returns the default
// configuration.
func DefaultConfig() Config {
	return Config{
		maximumWaitTimeout:       30.0,
		maximumWaitDelay:         10.0,
		maximumWebhookURLTimeout: 30.0,
		defaultWaitTimeout:       10.0,
		defaultWebhookURLTimeout: 10.0,
		defaultListenPort:        3000,
		disableGoogleChrome:      false,
		disableUnoconv:           false,
		logLevel:                 xlog.InfoLevel,
	}
}

/*
FromEnv returns a Conf according
to environment variables.
*/
func FromEnv() (Config, error) {
	const op string = "conf.FromEnv"
	resolver := func() (Config, error) {
		c := DefaultConfig()
		maximumWaitTimeout, err := xassert.Float64FromEnv(
			maximumWaitTimeoutEnvVar,
			c.maximumWaitTimeout,
			xassert.Float64NotInferiorTo(0.0),
		)
		c.maximumWaitTimeout = maximumWaitTimeout
		if err != nil {
			return c, err
		}
		maximumWaitDelay, err := xassert.Float64FromEnv(
			maximumWaitDelayEnvVar,
			c.maximumWaitDelay,
			xassert.Float64NotInferiorTo(0.0),
		)
		c.maximumWaitDelay = maximumWaitDelay
		if err != nil {
			return c, err
		}
		maximumWebhookURLTimeout, err := xassert.Float64FromEnv(
			maximumWebhookURLTimeoutEnvVar,
			c.maximumWebhookURLTimeout,
			xassert.Float64NotInferiorTo(0.0),
		)
		c.maximumWebhookURLTimeout = maximumWebhookURLTimeout
		if err != nil {
			return c, err
		}
		defaultWaitTimeout, err := xassert.Float64FromEnv(
			defaultWaitTimeoutEnvVar,
			c.defaultWaitTimeout,
			xassert.Float64NotInferiorTo(0.0),
			xassert.Float64NotSuperiorTo(c.maximumWaitTimeout),
		)
		c.defaultWaitTimeout = defaultWaitTimeout
		if err != nil {
			return c, err
		}
		defaultWebhookURLTimeout, err := xassert.Float64FromEnv(
			defaultWebhookURLTimeoutEnvVar,
			c.defaultWebhookURLTimeout,
			xassert.Float64NotInferiorTo(0.0),
			xassert.Float64NotSuperiorTo(c.defaultWebhookURLTimeout),
		)
		c.defaultWebhookURLTimeout = defaultWebhookURLTimeout
		if err != nil {
			return c, err
		}
		defaultListenPort, err := xassert.Int64FromEnv(
			defaultListenPortEnvVar,
			c.defaultListenPort,
			xassert.Int64NotInferiorTo(0),
			xassert.Int64NotSuperiorTo(65535),
		)
		c.defaultListenPort = defaultListenPort
		if err != nil {
			return c, err
		}
		disableGoogleChrome, err := xassert.BoolFromEnv(
			disableGoogleChromeEnvVar,
			c.disableGoogleChrome,
		)
		c.disableGoogleChrome = disableGoogleChrome
		if err != nil {
			return c, err
		}
		disableUnoconv, err := xassert.BoolFromEnv(
			disableUnoconvEnvVar,
			c.disableUnoconv,
		)
		c.disableUnoconv = disableUnoconv
		if err != nil {
			return c, err
		}
		logLevel, err := xassert.StringFromEnv(
			logLevelEnvVar,
			string(c.logLevel),
			xassert.StringOneOf(xlog.Levels()),
		)
		c.logLevel = xlog.MustParseLevel(logLevel)
		if err != nil {
			return c, err
		}
		return c, nil
	}
	result, err := resolver()
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

// MaximumWaitTimeout returns the maximum
// wait timeout from the configuration.
func (c Config) MaximumWaitTimeout() float64 {
	return c.maximumWaitTimeout
}

// MaximumWaitDelay returns the maximum
// wait timeout from the configuration.
func (c Config) MaximumWaitDelay() float64 {
	return c.maximumWaitDelay
}

// MaximumWebhookURLTimeout returns the maximum
// webhook URL wait timeout from the configuration.
func (c Config) MaximumWebhookURLTimeout() float64 {
	return c.maximumWebhookURLTimeout
}

// DefaultWaitTimeout returns the default
// wait timeout from the configuration.
func (c Config) DefaultWaitTimeout() float64 {
	return c.defaultWaitTimeout
}

// DefaultWebhookURLTimeout returns the default
// webhook URL wait timeout from the configuration.
func (c Config) DefaultWebhookURLTimeout() float64 {
	return c.defaultWebhookURLTimeout
}

// DefaultListenPort returns the default
// listen port from the configuration.
func (c Config) DefaultListenPort() int64 {
	return c.defaultListenPort
}

/*
DisableGoogleChrome returns true if
Google Chrome is disabled in the
configuration.
*/
func (c Config) DisableGoogleChrome() bool {
	return c.disableGoogleChrome
}

/*
DisableUnoconv returns true if
Unoconv is disabled in the
configuration.
*/
func (c Config) DisableUnoconv() bool {
	return c.disableUnoconv
}

// LogLevel returns the xlog.Level from
// the configuration.
func (c Config) LogLevel() xlog.Level {
	return c.logLevel
}
