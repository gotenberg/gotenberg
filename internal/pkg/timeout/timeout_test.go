package timeout

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestDuration(t *testing.T) {
	expected := time.Duration(1500) * time.Millisecond
	result := Duration(1.5)
	assert.Equal(t, expected.String(), result.String())
}

func TestErr(t *testing.T) {
	previousErr := errors.New("previous error")
	// should be OK.
	ctx, cancel := Context(5)
	defer cancel()
	assert.NotNil(t, Err(ctx, previousErr))
	// should timeout.
	ctx, cancel = Context(0.5)
	defer cancel()
	time.Sleep(Duration(1))
	err := Err(ctx, previousErr)
	assert.NotNil(t, err)
	standardized := test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Timeout, standardized.Code)
	// should failed.
	ctx, cancel = Context(5)
	cancel()
	err = Err(ctx, previousErr)
	assert.NotNil(t, err)
	standardized = test.RequireStandardError(t, err)
	assert.Equal(t, standarderror.Internal, standarderror.Code(err))
}
