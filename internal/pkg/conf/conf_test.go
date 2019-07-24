package conf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestEmptyFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// no environment variables set,
	// values should be equal to default config.
	expected = DefaultConfig()
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}

func TestMaximumWaitTimeoutFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// MAXIMUM_WAIT_TIMEOUT correctly set.
	os.Setenv(maximumWaitTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.maximumWaitTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWaitTimeoutEnvVar)
	// MAXIMUM_WAIT_TIMEOUT wrongly set.
	os.Setenv(maximumWaitTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWaitTimeoutEnvVar)
	// MAXIMUM_WAIT_TIMEOUT < 0.
	os.Setenv(maximumWaitTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWaitTimeoutEnvVar)
}

func TestMaximumWaitDelayFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// MAXIMUM_WAIT_DELAY correctly set.
	os.Setenv(maximumWaitDelayEnvVar, "10.0")
	expected = DefaultConfig()
	expected.maximumWaitDelay = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWaitDelayEnvVar)
	// MAXIMUM_WAIT_DELAY wrongly set.
	os.Setenv(maximumWaitDelayEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWaitDelayEnvVar)
	// MAXIMUM_WAIT_DELAY < 0.
	os.Setenv(maximumWaitDelayEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWaitDelayEnvVar)
}

func TestMaximumWebhookURLTimeoutFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// MAXIMUM_WEBHOOK_URL_TIMEOUT correctly set.
	os.Setenv(maximumWebhookURLTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.maximumWebhookURLTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWebhookURLTimeoutEnvVar)
	// MAXIMUM_WEBHOOK_URL_TIMEOUT wrongly set.
	os.Setenv(maximumWebhookURLTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWebhookURLTimeoutEnvVar)
	// MAXIMUM_WEBHOOK_URL_TIMEOUT < 0.
	os.Setenv(maximumWebhookURLTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(maximumWebhookURLTimeoutEnvVar)
}

func TestDefaultWaitTimeoutFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DEFAULT_WAIT_TIMEOUT correctly set.
	os.Setenv(defaultWaitTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.defaultWaitTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWaitTimeoutEnvVar)
	// DEFAULT_WAIT_TIMEOUT wrongly set.
	os.Setenv(defaultWaitTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWaitTimeoutEnvVar)
	// DEFAULT_WAIT_TIMEOUT < 0.
	os.Setenv(defaultWaitTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWaitTimeoutEnvVar)
	// DEFAULT_WAIT_TIMEOUT > MAXIMUM_WAIT_TIMEOUT.
	os.Setenv(defaultWaitTimeoutEnvVar, "40.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWaitTimeoutEnvVar)
}

func TestDefaultWebhookURLTimeoutFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DEFAULT_WEBHOOK_URL_TIMEOUT correctly set.
	os.Setenv(defaultWebhookURLTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.defaultWebhookURLTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWebhookURLTimeoutEnvVar)
	// DEFAULT_WEBHOOK_URL_TIMEOUT wrongly set.
	os.Setenv(defaultWebhookURLTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWebhookURLTimeoutEnvVar)
	// DEFAULT_WEBHOOK_URL_TIMEOUT < 0.
	os.Setenv(defaultWebhookURLTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWebhookURLTimeoutEnvVar)
	// DEFAULT_WEBHOOK_URL_TIMEOUT > MAXIMUM_WEBHOOK_URL_TIMEOUT.
	os.Setenv(defaultWebhookURLTimeoutEnvVar, "40.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultWebhookURLTimeoutEnvVar)
}

func TestDefaultListenPortFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DEFAULT_LISTEN_PORT correctly set.
	os.Setenv(defaultListenPortEnvVar, "80")
	expected = DefaultConfig()
	expected.defaultListenPort = 80
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultListenPortEnvVar)
	// DEFAULT_LISTEN_PORT wrongly set.
	os.Setenv(defaultListenPortEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultListenPortEnvVar)
	// DEFAULT_LISTEN_PORT < 0.
	os.Setenv(defaultListenPortEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultListenPortEnvVar)
	// DEFAULT_LISTEN_PORT > 65535.
	os.Setenv(defaultListenPortEnvVar, "65536")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(defaultListenPortEnvVar)
}

func TestDisableGoogleChromeFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DISABLE_GOOGLE_CHROME correctly set.
	os.Setenv(disableGoogleChromeEnvVar, "1")
	expected = DefaultConfig()
	expected.disableGoogleChrome = true
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(disableGoogleChromeEnvVar)
	os.Setenv(disableGoogleChromeEnvVar, "0")
	expected = DefaultConfig()
	expected.disableGoogleChrome = false
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(disableGoogleChromeEnvVar)
	// DISABLE_GOOGLE_CHROME wrongly set.
	os.Setenv(disableGoogleChromeEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(disableGoogleChromeEnvVar)
}

func TestDisableUnoconvFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DISABLE_UNOCONV correctly set.
	os.Setenv(disableUnoconvEnvVar, "1")
	expected = DefaultConfig()
	expected.disableUnoconv = true
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(disableUnoconvEnvVar)
	os.Setenv(disableUnoconvEnvVar, "0")
	expected = DefaultConfig()
	expected.disableUnoconv = false
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(disableUnoconvEnvVar)
	// DISABLE_UNOCONV wrongly set.
	os.Setenv(disableUnoconvEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(disableUnoconvEnvVar)
}

func TestLogLevelFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// LOG_LEVEL correctly set.
	os.Setenv(logLevelEnvVar, "DEBUG")
	expected = DefaultConfig()
	expected.logLevel = xlog.DebugLevel
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(logLevelEnvVar)
	os.Setenv(logLevelEnvVar, "INFO")
	expected = DefaultConfig()
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(logLevelEnvVar)
	os.Setenv(logLevelEnvVar, "ERROR")
	expected = DefaultConfig()
	expected.logLevel = xlog.ErrorLevel
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(logLevelEnvVar)
	// LOG_LEVEL wrongly set.
	os.Setenv(logLevelEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(logLevelEnvVar)
}

func TestGetters(t *testing.T) {
	result := DefaultConfig()
	assert.Equal(t, result.maximumWaitTimeout, result.MaximumWaitTimeout())
	assert.Equal(t, result.maximumWaitDelay, result.MaximumWaitDelay())
	assert.Equal(t, result.maximumWebhookURLTimeout, result.MaximumWebhookURLTimeout())
	assert.Equal(t, result.defaultWaitTimeout, result.DefaultWaitTimeout())
	assert.Equal(t, result.defaultWebhookURLTimeout, result.DefaultWebhookURLTimeout())
	assert.Equal(t, result.defaultListenPort, result.DefaultListenPort())
	assert.Equal(t, result.disableGoogleChrome, result.DisableGoogleChrome())
	assert.Equal(t, result.disableUnoconv, result.DisableUnoconv())
	assert.Equal(t, result.logLevel, result.LogLevel())
}
