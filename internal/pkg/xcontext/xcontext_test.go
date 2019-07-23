package xcontext

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/xerrortest"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/xlogtest"
)

func TestMustHandleError(t *testing.T) {
	previousErr := errors.New("previous error")
	logger := xlogtest.DebugLogger()
	// context should not have an error.
	ctx, cancel := WithTimeout(logger, 5)
	defer cancel()
	err := MustHandleError(ctx, previousErr)
	assert.Equal(t, previousErr, err)
	// should panic.
	ctx, cancel = WithTimeout(logger, 5)
	defer cancel()
	assert.Panics(t, func() {
		MustHandleError(ctx, nil)
	})
	// context should timed out.
	ctx, cancel = WithTimeout(logger, 0.5)
	defer cancel()
	time.Sleep(xtime.Duration(1))
	err = MustHandleError(ctx, previousErr)
	xerr := xerrortest.AssertError(t, err)
	assert.Equal(t, xerror.TimeoutCode, xerror.Code(xerr))
	// context should have an error different
	// than context.DeadlineExceeded.
	ctx, cancel = WithTimeout(logger, 5)
	cancel()
	err = MustHandleError(ctx, previousErr)
	xerr = xerrortest.AssertError(t, err)
	assert.Equal(t, xerror.InternalCode, xerror.Code(xerr))
}
