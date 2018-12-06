package rand

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	rand1, err1 := Get()
	rand2, err2 := Get()
	assert.Nil(t, err1)
	assert.Nil(t, err2)
	assert.NotEqual(t, rand1, rand2)
}
