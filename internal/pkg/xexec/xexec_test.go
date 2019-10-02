package xexec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xcontext"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestCommand(t *testing.T) {
	logger := test.DebugLogger()
	// should pipe command output as
	// xlog.Logger has a xlog.DebugLevel.
	cmd, err := Command(logger, "echo", "Hello", "World")
	assert.Nil(t, err)
	LogBeforeExecute(logger, cmd)
	// should not pipe command output as
	// xlog.Logger has a xlog.InfoLevel.
	logger = test.InfoLogger()
	cmd, err = Command(logger, "echo", "Hello", "World")
	LogBeforeExecute(logger, cmd)
	assert.Nil(t, err)
}

func TestCommandContext(t *testing.T) {
	logger := test.DebugLogger()
	// should pipe command output as
	// xlog.Logger has a xlog.DebugLevel.
	cmd, err := CommandContext(context.Background(), logger, "echo", "Hello", "World")
	assert.Nil(t, err)
	LogBeforeExecute(logger, cmd)
	// should not pipe command output as
	// xlog.Logger has a xlog.InfoLevel.
	logger = test.InfoLogger()
	cmd, err = CommandContext(context.Background(), logger, "echo", "Hello", "World")
	LogBeforeExecute(logger, cmd)
	assert.Nil(t, err)
}

func TestRun(t *testing.T) {
	logger := test.DebugLogger()
	// should run without issue.
	err := Run(context.Background(), logger, "echo", "Hello", "World")
	assert.Nil(t, err)
	// should not be OK as context.Context
	// should timeout.
	ctx, cancel := xcontext.WithTimeout(logger, 0)
	defer cancel()
	err = Run(ctx, logger, "echo", "Hello", "World")
	assert.NotNil(t, err)
}
