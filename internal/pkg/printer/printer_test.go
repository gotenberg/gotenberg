package printer

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/timeout"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestHandlerErr(t *testing.T) {
	previousErr := errors.New("previous error")
	// should be OK.
	ctx, cancel := timeout.Context(5)
	defer cancel()
	assert.NotNil(t, handleErrContext(ctx, previousErr))
	// should timeout.
	ctx, cancel = timeout.Context(0.5)
	defer cancel()
	time.Sleep(timeout.Duration(1))
	err := handleErrContext(ctx, previousErr)
	assert.NotNil(t, err)
	standardized := test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Timeout, standardized.Code)
	// should failed.
	ctx, cancel = timeout.Context(5)
	cancel()
	err = handleErrContext(ctx, previousErr)
	assert.NotNil(t, err)
	standardized = test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Internal, standarderror.Code(err))
}
