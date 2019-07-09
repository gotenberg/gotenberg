package timeout

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	ctx, cancel := Context(1.5)
	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)
}

func TestDuration(t *testing.T) {
	expected := time.Duration(1500) * time.Millisecond
	result := Duration(1.5)
	assert.Equal(t, expected.String(), result.String())
}
