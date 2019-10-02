package xassert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestFloat64NotInferiorTo(t *testing.T) {
	rule := Float64NotInferiorTo(0.0)
	// should be OK.
	rule.with("FOO", 10.0)
	err := rule.validate()
	assert.Nil(t, err)
	// should not be OK.
	rule.with("FOO", -10.0)
	err = rule.validate()
	test.AssertError(t, err)
}

func TestFloat64NotSuperiorTo(t *testing.T) {
	rule := Float64NotSuperiorTo(0.0)
	// should be OK.
	rule.with("FOO", -10.0)
	err := rule.validate()
	assert.Nil(t, err)
	// should not be OK.
	rule.with("FOO", 10.0)
	err = rule.validate()
	test.AssertError(t, err)
}
