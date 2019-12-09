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
	os.Setenv(MaximumWaitTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.maximumWaitTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWaitTimeoutEnvVar)
	// MAXIMUM_WAIT_TIMEOUT wrongly set.
	os.Setenv(MaximumWaitTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWaitTimeoutEnvVar)
	// MAXIMUM_WAIT_TIMEOUT < 0.
	os.Setenv(MaximumWaitTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWaitTimeoutEnvVar)
}

func TestMaximumWaitDelayFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// MAXIMUM_WAIT_DELAY correctly set.
	os.Setenv(MaximumWaitDelayEnvVar, "10.0")
	expected = DefaultConfig()
	expected.maximumWaitDelay = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWaitDelayEnvVar)
	// MAXIMUM_WAIT_DELAY wrongly set.
	os.Setenv(MaximumWaitDelayEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWaitDelayEnvVar)
	// MAXIMUM_WAIT_DELAY < 0.
	os.Setenv(MaximumWaitDelayEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWaitDelayEnvVar)
}

func TestMaximumWebhookURLTimeoutFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// MAXIMUM_WEBHOOK_URL_TIMEOUT correctly set.
	os.Setenv(MaximumWebhookURLTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.maximumWebhookURLTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWebhookURLTimeoutEnvVar)
	// MAXIMUM_WEBHOOK_URL_TIMEOUT wrongly set.
	os.Setenv(MaximumWebhookURLTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWebhookURLTimeoutEnvVar)
	// MAXIMUM_WEBHOOK_URL_TIMEOUT < 0.
	os.Setenv(MaximumWebhookURLTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(MaximumWebhookURLTimeoutEnvVar)
}

func TestDefaultWaitTimeoutFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DEFAULT_WAIT_TIMEOUT correctly set.
	os.Setenv(DefaultWaitTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.defaultWaitTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWaitTimeoutEnvVar)
	// DEFAULT_WAIT_TIMEOUT wrongly set.
	os.Setenv(DefaultWaitTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWaitTimeoutEnvVar)
	// DEFAULT_WAIT_TIMEOUT < 0.
	os.Setenv(DefaultWaitTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWaitTimeoutEnvVar)
	// DEFAULT_WAIT_TIMEOUT > MAXIMUM_WAIT_TIMEOUT.
	os.Setenv(DefaultWaitTimeoutEnvVar, "40.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWaitTimeoutEnvVar)
}

func TestDefaultWebhookURLTimeoutFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DEFAULT_WEBHOOK_URL_TIMEOUT correctly set.
	os.Setenv(DefaultWebhookURLTimeoutEnvVar, "10.0")
	expected = DefaultConfig()
	expected.defaultWebhookURLTimeout = 10.0
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWebhookURLTimeoutEnvVar)
	// DEFAULT_WEBHOOK_URL_TIMEOUT wrongly set.
	os.Setenv(DefaultWebhookURLTimeoutEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWebhookURLTimeoutEnvVar)
	// DEFAULT_WEBHOOK_URL_TIMEOUT < 0.
	os.Setenv(DefaultWebhookURLTimeoutEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWebhookURLTimeoutEnvVar)
	// DEFAULT_WEBHOOK_URL_TIMEOUT > MAXIMUM_WEBHOOK_URL_TIMEOUT.
	os.Setenv(DefaultWebhookURLTimeoutEnvVar, "40.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultWebhookURLTimeoutEnvVar)
}

func TestDefaultListenPortFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DEFAULT_LISTEN_PORT correctly set.
	os.Setenv(DefaultListenPortEnvVar, "80")
	expected = DefaultConfig()
	expected.defaultListenPort = 80
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultListenPortEnvVar)
	// DEFAULT_LISTEN_PORT wrongly set.
	os.Setenv(DefaultListenPortEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultListenPortEnvVar)
	// DEFAULT_LISTEN_PORT < 0.
	os.Setenv(DefaultListenPortEnvVar, "-1.0")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultListenPortEnvVar)
	// DEFAULT_LISTEN_PORT > 65535.
	os.Setenv(DefaultListenPortEnvVar, "65536")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultListenPortEnvVar)
}

func TestDisableGoogleChromeFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DISABLE_GOOGLE_CHROME correctly set.
	os.Setenv(DisableGoogleChromeEnvVar, "1")
	expected = DefaultConfig()
	expected.disableGoogleChrome = true
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DisableGoogleChromeEnvVar)
	os.Setenv(DisableGoogleChromeEnvVar, "0")
	expected = DefaultConfig()
	expected.disableGoogleChrome = false
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DisableGoogleChromeEnvVar)
	// DISABLE_GOOGLE_CHROME wrongly set.
	os.Setenv(DisableGoogleChromeEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DisableGoogleChromeEnvVar)
}

func TestDisableUnoconvFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DISABLE_UNOCONV correctly set.
	os.Setenv(DisableUnoconvEnvVar, "1")
	expected = DefaultConfig()
	expected.disableUnoconv = true
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DisableUnoconvEnvVar)
	os.Setenv(DisableUnoconvEnvVar, "0")
	expected = DefaultConfig()
	expected.disableUnoconv = false
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DisableUnoconvEnvVar)
	// DISABLE_UNOCONV wrongly set.
	os.Setenv(DisableUnoconvEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DisableUnoconvEnvVar)
}

func TestLogLevelFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// LOG_LEVEL correctly set.
	os.Setenv(LogLevelEnvVar, "DEBUG")
	expected = DefaultConfig()
	expected.logLevel = xlog.DebugLevel
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(LogLevelEnvVar)
	os.Setenv(LogLevelEnvVar, "INFO")
	expected = DefaultConfig()
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(LogLevelEnvVar)
	os.Setenv(LogLevelEnvVar, "ERROR")
	expected = DefaultConfig()
	expected.logLevel = xlog.ErrorLevel
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(LogLevelEnvVar)
	// LOG_LEVEL wrongly set.
	os.Setenv(LogLevelEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(LogLevelEnvVar)
}

func TestRootPathFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// ROOT_PATH correctly set.
	os.Setenv(RootPathEnvVar, "/foo/")
	expected = DefaultConfig()
	expected.rootPath = "/foo/"
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(RootPathEnvVar)
	// ROOT_PATH wrongly set.
	os.Setenv(RootPathEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(RootPathEnvVar)
}

func TestDefaultGoogleChromeRpccBufferSizeFromEnv(t *testing.T) {
	var (
		expected Config
		result   Config
		err      error
	)
	// DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE correctly set.
	os.Setenv(DefaultGoogleChromeRpccBufferSizeEnvVar, "100")
	expected = DefaultConfig()
	expected.defaultGoogleChromeRpccBufferSize = 100
	result, err = FromEnv()
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultGoogleChromeRpccBufferSizeEnvVar)
	// DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE wrongly set.
	os.Setenv(DefaultGoogleChromeRpccBufferSizeEnvVar, "foo")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultGoogleChromeRpccBufferSizeEnvVar)
	// DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE < 0.
	os.Setenv(DefaultGoogleChromeRpccBufferSizeEnvVar, "-1")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultGoogleChromeRpccBufferSizeEnvVar)
	// DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE > 100 MB (maximumGoogleChromeRpccBufferSize).
	os.Setenv(DefaultGoogleChromeRpccBufferSizeEnvVar, "104857601")
	expected = DefaultConfig()
	result, err = FromEnv()
	test.AssertError(t, err)
	assert.Equal(t, expected, result)
	os.Unsetenv(DefaultGoogleChromeRpccBufferSizeEnvVar)
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
	assert.Equal(t, result.rootPath, result.RootPath())
	assert.Equal(t, result.maximumGoogleChromeRpccBufferSize, result.MaximumGoogleChromeRpccBufferSize())
	assert.Equal(t, result.defaultGoogleChromeRpccBufferSize, result.DefaultGoogleChromeRpccBufferSize())
}
