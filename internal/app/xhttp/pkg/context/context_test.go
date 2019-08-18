package context

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/pm2"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestMustCastFromEchoContext(t *testing.T) {
	// should be OK.
	ctx := New(
		test.DummyEchoContext(),
		test.DebugLogger(),
		conf.DefaultConfig(),
	)
	assert.NotPanics(t, func() {
		result := MustCastFromEchoContext(ctx)
		assert.Equal(t, ctx, result)
	})
	// should not be OK.
	assert.Panics(t, func() {
		MustCastFromEchoContext(test.DummyEchoContext())
	})
}

func TestProcessesHealthcheck(t *testing.T) {
	// process is viable.
	ctx := New(
		test.DummyEchoContext(),
		test.DebugLogger(),
		conf.DefaultConfig(),
		pm2.NewDummyProcess(true),
	)
	err := ctx.ProcessesHealthcheck()
	assert.Nil(t, err)
	// process is not viable.
	ctx = New(
		test.DummyEchoContext(),
		test.DebugLogger(),
		conf.DefaultConfig(),
		pm2.NewDummyProcess(false),
	)
	err = ctx.ProcessesHealthcheck()
	test.AssertError(t, err)
}

func TestLogRequestResult(t *testing.T) {
	ctx := New(
		test.DummyEchoContext(),
		test.DebugLogger(),
		conf.DefaultConfig(),
	)
	// Info log.
	err := ctx.LogRequestResult(nil, false)
	assert.Nil(t, err)
	// Debug log.
	err = ctx.LogRequestResult(nil, true)
	assert.Nil(t, err)
	// Error log.
	err = ctx.LogRequestResult(errors.New("foo"), true)
	assert.NotNil(t, err)
}

func TestGetters(t *testing.T) {
	const resourceDirectoryName string = "foo"
	logger := test.DebugLogger()
	config := conf.DefaultConfig()
	ctx := New(
		test.DummyEchoContext(),
		logger,
		config,
	)
	// Logger.
	assert.Equal(t, logger, ctx.XLogger())
	// Config.
	assert.Equal(t, config, ctx.Config())
	// Context should not have a resource.Resource.
	assert.Equal(t, false, ctx.HasResource())
	assert.Panics(t, func() {
		ctx.MustResource()
	})
	// Context should have a resource.Resource.
	ctx = New(
		test.EchoContextMultipart(t),
		logger,
		config,
	)
	err := ctx.WithResource(resourceDirectoryName)
	assert.Nil(t, err)
	assert.Equal(t, true, ctx.HasResource())
	assert.NotPanics(t, func() {
		r := ctx.MustResource()
		err = r.Close()
		assert.Nil(t, err)
	})
}
