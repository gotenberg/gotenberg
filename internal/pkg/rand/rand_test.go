package rand

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	rand1, err := Get()
	require.Nil(t, err)
	rand2, err := Get()
	require.Nil(t, err)
	assert.NotEqual(t, rand1, rand2)
}
