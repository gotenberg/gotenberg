package config

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestDefaultWaitTimeout(t *testing.T) {
	// should be OK.
	config, err := FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, 10.0, config.DefaultWaitTimeout())
	os.Setenv(defaultWaitTimeoutEnvVar, "1.5")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, 1.5, config.DefaultWaitTimeout())
	// should failed.
	os.Setenv(defaultWaitTimeoutEnvVar, "foo")
	_, err = FromEnv()
	assert.NotNil(t, err)
	standardized := test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Invalid, standarderror.Code(standardized))
	os.Unsetenv(defaultWaitTimeoutEnvVar)
}

func TestDefaultListenPort(t *testing.T) {
	// should be OK.
	config, err := FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, "3000", config.DefaultListenPort())
	os.Setenv(defaultListenPortEnvVar, "4000")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, "4000", config.DefaultListenPort())
	// should failed.
	os.Setenv(defaultListenPortEnvVar, "foo")
	_, err = FromEnv()
	assert.NotNil(t, err)
	standardized := test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Invalid, standarderror.Code(standardized))
	os.Setenv(defaultListenPortEnvVar, "100000000")
	_, err = FromEnv()
	assert.NotNil(t, err)
	standardized = test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Invalid, standarderror.Code(standardized))
	os.Unsetenv(defaultListenPortEnvVar)
}

func TestEnableChromeEndpoints(t *testing.T) {
	// should be OK.
	config, err := FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, true, config.EnableChromeEndpoints())
	os.Setenv(disableGoogleChromeEnvVar, "1")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, false, config.EnableChromeEndpoints())
	os.Setenv(disableGoogleChromeEnvVar, "0")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, true, config.EnableChromeEndpoints())
	// should failed.
	os.Setenv(disableGoogleChromeEnvVar, "true")
	_, err = FromEnv()
	assert.NotNil(t, err)
	standardized := test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Invalid, standarderror.Code(standardized))
	os.Unsetenv(disableGoogleChromeEnvVar)
}

func TestEnableUnoconvEndpoints(t *testing.T) {
	// should be OK.
	config, err := FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, true, config.EnableUnoconvEndpoints())
	os.Setenv(disableUnoconvEnvVar, "1")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, false, config.EnableUnoconvEndpoints())
	os.Setenv(disableUnoconvEnvVar, "0")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, true, config.EnableUnoconvEndpoints())
	// should failed.
	os.Setenv(disableUnoconvEnvVar, "true")
	_, err = FromEnv()
	assert.NotNil(t, err)
	standardized := test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Invalid, standarderror.Code(standardized))
	os.Unsetenv(disableUnoconvEnvVar)
}

func TestLogLevel(t *testing.T) {
	// should be OK.
	config, err := FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, logrus.InfoLevel, config.LogLevel())
	os.Setenv(logLevelEnvVar, "DEBUG")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, logrus.DebugLevel, config.LogLevel())
	os.Setenv(logLevelEnvVar, "INFO")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, logrus.InfoLevel, config.LogLevel())
	os.Setenv(logLevelEnvVar, "ERROR")
	config, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, logrus.ErrorLevel, config.LogLevel())
	// should failed.
	os.Setenv(logLevelEnvVar, "foo")
	config, err = FromEnv()
	assert.Equal(t, logrus.InfoLevel, config.LogLevel())
	assert.NotNil(t, err)
	standardized := test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Invalid, standarderror.Code(standardized))
	os.Unsetenv(logLevelEnvVar)
}
