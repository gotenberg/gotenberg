package timeout

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration(t *testing.T) {
	expected := time.Duration(1500) * time.Millisecond
	result := Duration(1.5)
	assert.Equal(t, expected.String(), result.String())
}
