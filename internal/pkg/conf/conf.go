package conf

import (
	"github.com/thecodingmachine/gotenberg/internal/pkg/xassert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

const (
	// MaximumWaitTimeoutEnvVar contains the name
	// of the environment variable "MAXIMUM_WAIT_TIMEOUT".
	MaximumWaitTimeoutEnvVar string = "MAXIMUM_WAIT_TIMEOUT"
	// MaximumWaitDelayEnvVar contains the name
	// of the environment variable "MAXIMUM_WAIT_DELAY".
	MaximumWaitDelayEnvVar string = "MAXIMUM_WAIT_DELAY"
	// MaximumWebhookURLTimeoutEnvVar contains the name
	// of the environment variable "MAXIMUM_WEBHOOK_URL_TIMEOUT".
	MaximumWebhookURLTimeoutEnvVar string = "MAXIMUM_WEBHOOK_URL_TIMEOUT"
	// DefaultWaitTimeoutEnvVar contains the name
	// of the environment variable "DEFAULT_WAIT_TIMEOUT".
	DefaultWaitTimeoutEnvVar string = "DEFAULT_WAIT_TIMEOUT"
	// DefaultWebhookURLTimeoutEnvVar contains the name
	// of the environment variable "DEFAULT_WEBHOOK_URL_TIMEOUT".
	DefaultWebhookURLTimeoutEnvVar string = "DEFAULT_WEBHOOK_URL_TIMEOUT"
	// DefaultListenPortEnvVar contains the name
	// of the environment variable "DEFAULT_LISTEN_PORT".
	DefaultListenPortEnvVar string = "PORT"
	// DisableGoogleChromeEnvVar contains the name
	// of the environment variable "DISABLE_GOOGLE_CHROME".
	DisableGoogleChromeEnvVar string = "DISABLE_GOOGLE_CHROME"
	// DisableUnoconvEnvVar contains the name
	// of the environment variable "DISABLE_UNOCONV".
	DisableUnoconvEnvVar string = "DISABLE_UNOCONV"
	// LogLevelEnvVar contains the name
	// of the environment variable "LOG_LEVEL".
	LogLevelEnvVar string = "LOG_LEVEL"
	// RootPathEnvVar contains the name
	// of the environment variable "ROOT_PATH".
	RootPathEnvVar string = "ROOT_PATH"
	// DefaultGoogleChromeRpccBufferSizeEnvVar contains the name
	// of the environment variable "DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE".
	DefaultGoogleChromeRpccBufferSizeEnvVar string = "DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE"
	// GoogleChromeIgnoreCertificateErrorsEnvVar contains the name
	// of the environment variable "GOOGLE_CHROME_IGNORE_CERTIFICATE_ERRORS".
	GoogleChromeIgnoreCertificateErrorsEnvVar string = "GOOGLE_CHROME_IGNORE_CERTIFICATE_ERRORS"
)

// Config contains the application
// configuration.
type Config struct {
	maximumWaitTimeout                  float64
	maximumWaitDelay                    float64
	maximumWebhookURLTimeout            float64
	defaultWaitTimeout                  float64
	defaultWebhookURLTimeout            float64
	defaultListenPort                   int64
	disableGoogleChrome                 bool
	disableUnoconv                      bool
	googleChromeIgnoreCertificateErrors bool
	logLevel                            xlog.Level
	rootPath                            string
	maximumGoogleChromeRpccBufferSize   int64
	defaultGoogleChromeRpccBufferSize   int64
}

// DefaultConfig returns the default
// configuration.
func DefaultConfig() Config {
	return Config{
		maximumWaitTimeout:                  30.0,
		maximumWaitDelay:                    10.0,
		maximumWebhookURLTimeout:            30.0,
		defaultWaitTimeout:                  10.0,
		defaultWebhookURLTimeout:            10.0,
		defaultListenPort:                   3000,
		disableGoogleChrome:                 false,
		disableUnoconv:                      false,
		logLevel:                            xlog.InfoLevel,
		rootPath:                            "/",
		maximumGoogleChromeRpccBufferSize:   104857600, // ~100 MB
		defaultGoogleChromeRpccBufferSize:   1048576,   // 1 MB
		googleChromeIgnoreCertificateErrors: false,
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
			MaximumWaitTimeoutEnvVar,
			c.maximumWaitTimeout,
			xassert.Float64NotInferiorTo(0.0),
		)
		c.maximumWaitTimeout = maximumWaitTimeout
		if err != nil {
			return c, err
		}
		maximumWaitDelay, err := xassert.Float64FromEnv(
			MaximumWaitDelayEnvVar,
			c.maximumWaitDelay,
			xassert.Float64NotInferiorTo(0.0),
		)
		c.maximumWaitDelay = maximumWaitDelay
		if err != nil {
			return c, err
		}
		maximumWebhookURLTimeout, err := xassert.Float64FromEnv(
			MaximumWebhookURLTimeoutEnvVar,
			c.maximumWebhookURLTimeout,
			xassert.Float64NotInferiorTo(0.0),
		)
		c.maximumWebhookURLTimeout = maximumWebhookURLTimeout
		if err != nil {
			return c, err
		}
		defaultWaitTimeout, err := xassert.Float64FromEnv(
			DefaultWaitTimeoutEnvVar,
			c.defaultWaitTimeout,
			xassert.Float64NotInferiorTo(0.0),
			xassert.Float64NotSuperiorTo(c.maximumWaitTimeout),
		)
		c.defaultWaitTimeout = defaultWaitTimeout
		if err != nil {
			return c, err
		}
		defaultWebhookURLTimeout, err := xassert.Float64FromEnv(
			DefaultWebhookURLTimeoutEnvVar,
			c.defaultWebhookURLTimeout,
			xassert.Float64NotInferiorTo(0.0),
			xassert.Float64NotSuperiorTo(c.defaultWebhookURLTimeout),
		)
		c.defaultWebhookURLTimeout = defaultWebhookURLTimeout
		if err != nil {
			return c, err
		}
		defaultListenPort, err := xassert.Int64FromEnv(
			DefaultListenPortEnvVar,
			c.defaultListenPort,
			xassert.Int64NotInferiorTo(0),
			xassert.Int64NotSuperiorTo(65535),
		)
		c.defaultListenPort = defaultListenPort
		if err != nil {
			return c, err
		}
		disableGoogleChrome, err := xassert.BoolFromEnv(
			DisableGoogleChromeEnvVar,
			c.disableGoogleChrome,
		)
		c.disableGoogleChrome = disableGoogleChrome
		if err != nil {
			return c, err
		}
		disableUnoconv, err := xassert.BoolFromEnv(
			DisableUnoconvEnvVar,
			c.disableUnoconv,
		)
		c.disableUnoconv = disableUnoconv
		if err != nil {
			return c, err
		}
		logLevel, err := xassert.StringFromEnv(
			LogLevelEnvVar,
			string(c.logLevel),
			xassert.StringOneOf(xlog.Levels()),
		)
		c.logLevel = xlog.MustParseLevel(logLevel)
		if err != nil {
			return c, err
		}
		rootPath, err := xassert.StringFromEnv(
			RootPathEnvVar,
			c.rootPath,
			xassert.StringStartWith("/"),
			xassert.StringEndWith("/"),
		)
		c.rootPath = rootPath
		if err != nil {
			return c, err
		}
		defaultGoogleChromeRpccBufferSize, err := xassert.Int64FromEnv(
			DefaultGoogleChromeRpccBufferSizeEnvVar,
			c.defaultGoogleChromeRpccBufferSize,
			xassert.Int64NotInferiorTo(0),
			xassert.Int64NotSuperiorTo(c.MaximumGoogleChromeRpccBufferSize()),
		)
		c.defaultGoogleChromeRpccBufferSize = defaultGoogleChromeRpccBufferSize
		if err != nil {
			return c, err
		}
		googleChromeIgnoreCertificateErrors, err := xassert.BoolFromEnv(
			GoogleChromeIgnoreCertificateErrorsEnvVar,
			c.googleChromeIgnoreCertificateErrors,
		)
		c.googleChromeIgnoreCertificateErrors = googleChromeIgnoreCertificateErrors
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

// RootPath returns the rooth path from
// the configuration.
func (c Config) RootPath() string {
	return c.rootPath
}

// MaximumGoogleChromeRpccBufferSize returns the maximum
// Google Chrome rpcc buffer size from the configuration.
func (c Config) MaximumGoogleChromeRpccBufferSize() int64 {
	return c.maximumGoogleChromeRpccBufferSize
}

// DefaultGoogleChromeRpccBufferSize returns the default
//  Google Chrome rpcc buffer size from the configuration.
func (c Config) DefaultGoogleChromeRpccBufferSize() int64 {
	return c.defaultGoogleChromeRpccBufferSize
}

func (c Config) GoogleChromeIgnoreCertificateErrors() bool {
	return c.googleChromeIgnoreCertificateErrors
}
