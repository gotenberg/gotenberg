package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestArgKeys(t *testing.T) {
	expected := []ArgKey{
		ResultFilenameArgKey,
		WaitTimeoutArgKey,
		WebhookURLArgKey,
		WebhookURLTimeoutArgKey,
		RemoteURLArgKey,
		WaitDelayArgKey,
		PaperWidthArgKey,
		PaperHeightArgKey,
		MarginTopArgKey,
		MarginBottomArgKey,
		MarginLeftArgKey,
		MarginRightArgKey,
		LandscapeArgKey,
		GoogleChromeRpccBufferSizeArgKey,
	}
	assert.Equal(t, expected, ArgKeys())
}

func TestWaitTimeoutArg(t *testing.T) {
	const resourceDirectoryName string = "foo"
	var expected float64
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// argument does not exist.
	expected = config.DefaultWaitTimeout()
	v, err := WaitTimeoutArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// argument exist.
	expected = 5.0
	r.WithArg(WaitTimeoutArgKey, "5.0")
	v, err = WaitTimeoutArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is < 0.
	expected = config.DefaultWaitTimeout()
	r.WithArg(WaitTimeoutArgKey, "-1.0")
	v, err = WaitTimeoutArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is > config.MaximumWaitTimeout().
	expected = config.DefaultWaitTimeout()
	r.WithArg(WaitTimeoutArgKey, "31.0")
	v, err = WaitTimeoutArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as
	// argument value is invalid.
	expected = config.DefaultWaitTimeout()
	r.WithArg(WaitTimeoutArgKey, "foo")
	v, err = WaitTimeoutArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestWaitDelayArg(t *testing.T) {
	const (
		resourceDirectoryName string  = "foo"
		defaultValue          float64 = 0.0
	)
	var expected float64
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// argument does not exist.
	expected = defaultValue
	v, err := WaitDelayArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// argument exist.
	expected = 5.0
	r.WithArg(WaitDelayArgKey, "5.0")
	v, err = WaitDelayArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is < 0.
	expected = defaultValue
	r.WithArg(WaitDelayArgKey, "-1.0")
	v, err = WaitDelayArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is > config.MaximumWaitDelay().
	expected = defaultValue
	r.WithArg(WaitDelayArgKey, "31.0")
	v, err = WaitDelayArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as
	// argument value is invalid.
	expected = defaultValue
	r.WithArg(WaitDelayArgKey, "foo")
	v, err = WaitDelayArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestWebhookURLTimeoutArg(t *testing.T) {
	const resourceDirectoryName string = "foo"
	var expected float64
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// argument does not exist.
	expected = config.DefaultWebhookURLTimeout()
	v, err := WebhookURLTimeoutArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// argument exist.
	expected = 5.0
	r.WithArg(WebhookURLTimeoutArgKey, "5.0")
	v, err = WebhookURLTimeoutArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is < 0.
	expected = config.DefaultWebhookURLTimeout()
	r.WithArg(WebhookURLTimeoutArgKey, "-1.0")
	v, err = WebhookURLTimeoutArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is > config.MaximumWebhookURLTimeout().
	expected = config.DefaultWebhookURLTimeout()
	r.WithArg(WebhookURLTimeoutArgKey, "31.0")
	v, err = WebhookURLTimeoutArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as
	// argument value is invalid.
	expected = config.DefaultWebhookURLTimeout()
	r.WithArg(WebhookURLTimeoutArgKey, "foo")
	v, err = WebhookURLTimeoutArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestPaperSizeArgs(t *testing.T) {
	const resourceDirectoryName string = "foo"
	var expected float64
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	opts := printer.DefaultChromePrinterOptions(config)
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// arguments do not exist.
	width, height, err := PaperSizeArgs(r, config)
	assert.Nil(t, err)
	assert.Equal(t, opts.PaperWidth, width)
	assert.Equal(t, opts.PaperHeight, height)
	// arguments exist.
	expected = 5.0
	r.WithArg(PaperWidthArgKey, "5.0")
	r.WithArg(PaperHeightArgKey, "5.0")
	width, height, err = PaperSizeArgs(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, width)
	assert.Equal(t, expected, height)
	// should not be OK as arguments
	// value are < 0.
	expected = opts.PaperWidth
	r.WithArg(PaperWidthArgKey, "-1.0")
	width, _, err = PaperSizeArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, width)
	r.WithArg(PaperWidthArgKey, "5.0")
	expected = opts.PaperHeight
	r.WithArg(PaperHeightArgKey, "-1.0")
	_, height, err = PaperSizeArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, height)
	r.WithArg(PaperHeightArgKey, "5.0")
	// should not be OK as
	// arguments value are invalids.
	expected = opts.PaperWidth
	r.WithArg(PaperWidthArgKey, "foo")
	width, _, err = PaperSizeArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, width)
	r.WithArg(PaperWidthArgKey, "5.0")
	expected = opts.PaperHeight
	r.WithArg(PaperHeightArgKey, "foo")
	_, height, err = PaperSizeArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, height)
	r.WithArg(PaperHeightArgKey, "5.0")
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestMarginArgs(t *testing.T) {
	const resourceDirectoryName string = "foo"
	var expected float64
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	opts := printer.DefaultChromePrinterOptions(config)
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// arguments do not exist.
	top, bottom, left, right, err := MarginArgs(r, config)
	assert.Nil(t, err)
	assert.Equal(t, opts.MarginTop, top)
	assert.Equal(t, opts.MarginBottom, bottom)
	assert.Equal(t, opts.MarginLeft, left)
	assert.Equal(t, opts.MarginRight, right)
	// arguments exist.
	expected = 5.0
	r.WithArg(MarginTopArgKey, "5.0")
	r.WithArg(MarginBottomArgKey, "5.0")
	r.WithArg(MarginLeftArgKey, "5.0")
	r.WithArg(MarginRightArgKey, "5.0")
	top, bottom, left, right, err = MarginArgs(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, top)
	assert.Equal(t, expected, bottom)
	assert.Equal(t, expected, left)
	assert.Equal(t, expected, right)
	// should not be OK as arguments
	// value are < 0.
	expected = opts.MarginTop
	r.WithArg(MarginTopArgKey, "-1.0")
	top, _, _, _, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, top)
	r.WithArg(MarginTopArgKey, "5.0")
	expected = opts.MarginBottom
	r.WithArg(MarginBottomArgKey, "-1.0")
	_, bottom, _, _, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, bottom)
	r.WithArg(MarginBottomArgKey, "5.0")
	expected = opts.MarginLeft
	r.WithArg(MarginLeftArgKey, "-1.0")
	_, _, left, _, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, left)
	r.WithArg(MarginLeftArgKey, "5.0")
	expected = opts.MarginRight
	r.WithArg(MarginRightArgKey, "-1.0")
	_, _, _, right, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, right)
	r.WithArg(MarginRightArgKey, "5.0")
	// should not be OK as
	// arguments value are invalids.
	expected = opts.MarginTop
	r.WithArg(MarginTopArgKey, "foo")
	top, _, _, _, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, top)
	r.WithArg(MarginTopArgKey, "5.0")
	expected = opts.MarginBottom
	r.WithArg(MarginBottomArgKey, "foo")
	_, bottom, _, _, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, bottom)
	r.WithArg(MarginBottomArgKey, "5.0")
	expected = opts.MarginLeft
	r.WithArg(MarginLeftArgKey, "foo")
	_, _, left, _, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, left)
	r.WithArg(MarginLeftArgKey, "5.0")
	expected = opts.MarginRight
	r.WithArg(MarginRightArgKey, "foo")
	_, _, _, right, err = MarginArgs(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, right)
	r.WithArg(MarginRightArgKey, "5.0")
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestGoogleChromeRpccBufferSizeArg(t *testing.T) {
	const resourceDirectoryName string = "foo"
	var expected int64
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// argument does not exist.
	expected = config.DefaultGoogleChromeRpccBufferSize()
	v, err := GoogleChromeRpccBufferSizeArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// argument exist.
	expected = 10
	r.WithArg(GoogleChromeRpccBufferSizeArgKey, "10")
	v, err = GoogleChromeRpccBufferSizeArg(r, config)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is < 0.
	expected = config.DefaultGoogleChromeRpccBufferSize()
	r.WithArg(GoogleChromeRpccBufferSizeArgKey, "-1")
	v, err = GoogleChromeRpccBufferSizeArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as argument
	// value is > config.MaximumGoogleChromeRpccBufferSize().
	expected = config.DefaultGoogleChromeRpccBufferSize()
	r.WithArg(GoogleChromeRpccBufferSizeArgKey, "104857601")
	v, err = GoogleChromeRpccBufferSizeArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as
	// argument value is invalid.
	expected = config.DefaultGoogleChromeRpccBufferSize()
	r.WithArg(GoogleChromeRpccBufferSizeArgKey, "foo")
	v, err = GoogleChromeRpccBufferSizeArg(r, config)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}
