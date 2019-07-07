package config

import (
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const (
	defaultWaitTimeoutEnvVar  = "DEFAULT_WAIT_TIMEOUT"
	defaultListenPortEnvVar   = "DEFAULT_LISTEN_PORT"
	disableGoogleChromeEnvVar = "DISABLE_GOOGLE_CHROME"
	disableUnoconvEnvVar      = "DISABLE_UNOCONV"
	logLevelEnvVar            = "LOG_LEVEL"
)

type Config struct {
	defaultWaitTimeout     float64
	defaultListenPort      string
	enableChromeEndpoints  bool
	enableUnoconvEndpoints bool
	logLevel               log.Level
}

func defaultConfig() *Config {
	return &Config{
		defaultWaitTimeout:     10,
		defaultListenPort:      "3000",
		enableChromeEndpoints:  true,
		enableUnoconvEndpoints: true,
		logLevel:               log.InfoLevel,
	}
}

func FromEnv() (*Config, error) {
	c := defaultConfig()
	defaultWaitTimeout, err := defaultWaitTimeoutFromEnv(defaultWaitTimeoutEnvVar, c.DefaultWaitTimeout())
	c.defaultWaitTimeout = defaultWaitTimeout
	if err != nil {
		return c, err
	}
	defaultListenPort, err := defaultListenPortFromEnv(defaultListenPortEnvVar, c.DefaultListenPort())
	c.defaultListenPort = defaultListenPort
	if err != nil {
		return c, err
	}
	disableChromeEndpoints, err := boolFromEnv(disableGoogleChromeEnvVar, c.EnableChromeEndpoints())
	c.enableChromeEndpoints = !disableChromeEndpoints
	if err != nil {
		return c, err
	}
	disableUnoconvEndpoints, err := boolFromEnv(disableUnoconvEnvVar, c.EnableUnoconvEndpoints())
	c.enableUnoconvEndpoints = !disableUnoconvEndpoints
	if err != nil {
		return c, err
	}
	logLevel, err := logLevelFromEnv(logLevelEnvVar, c.LogLevel())
	c.logLevel = logLevel
	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Config) DefaultWaitTimeout() float64  { return c.defaultWaitTimeout }
func (c *Config) DefaultListenPort() string    { return c.defaultListenPort }
func (c *Config) EnableChromeEndpoints() bool  { return c.enableChromeEndpoints }
func (c *Config) EnableUnoconvEndpoints() bool { return c.enableUnoconvEndpoints }
func (c *Config) LogLevel() log.Level          { return c.logLevel }

func defaultWaitTimeoutFromEnv(envVar string, defaultValue float64) (float64, error) {
	if v, ok := os.LookupEnv(envVar); ok {
		waitTimeout, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return defaultValue, fmt.Errorf("%s: wrong value: want float got %v", envVar, err)
		}
		return waitTimeout, nil
	}
	return defaultValue, nil
}

func defaultListenPortFromEnv(envVar string, defaultValue string) (string, error) {
	if v, ok := os.LookupEnv(envVar); ok {
		portAsUint, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return defaultValue, fmt.Errorf("%s: wrong value: want uint got %v", envVar, err)
		}
		if portAsUint > 65535 {
			return defaultValue, fmt.Errorf("%s: wrong value: want uint < 65535 got %d", envVar, portAsUint)
		}
		return v, nil
	}
	return defaultValue, nil
}

func boolFromEnv(envVar string, defaultValue bool) (bool, error) {
	if v, ok := os.LookupEnv(envVar); ok {
		if v != "1" && v != "0" {
			return defaultValue, fmt.Errorf("%s: wrong value: want \"0\" or \"1\" got %s", envVar, v)
		}
		return v == "1", nil
	}
	return defaultValue, nil
}

func logLevelFromEnv(envVar string, defaultValue log.Level) (log.Level, error) {
	if v, ok := os.LookupEnv(envVar); ok {
		switch v {
		case "DEBUG":
			return log.DebugLevel, nil
		case "INFO":
			return log.InfoLevel, nil
		case "ERROR":
			return log.ErrorLevel, nil
		default:
			return defaultValue, fmt.Errorf("%s: wrong value: want \"DEBUG\",\"INFO\" or \"ERROR\" got %s", envVar, v)
		}
	}
	return defaultValue, nil
}
