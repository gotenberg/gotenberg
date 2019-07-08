package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	var rands []string
	// use case: for 1 000 concurrent
	// requests (which is a big Gotenberg instance),
	// none should have the identifier.
	for i := 0; i < 1000; i++ {
		rands = append(rands, Get())
	}
	unique := func() bool {
		for i, rand := range rands {
			for j, current := range rands {
				if i != j && rand == current {
					return false
				}
			}
		}
		return true
	}
	assert.Equal(t, true, unique())
}
