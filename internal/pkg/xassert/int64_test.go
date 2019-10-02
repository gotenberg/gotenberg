package xassert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestInt64NotInferiorTo(t *testing.T) {
	rule := Int64NotInferiorTo(0)
	// should be OK.
	rule.with("FOO", 10)
	err := rule.validate()
	assert.Nil(t, err)
	// should not be OK.
	rule.with("FOO", -10)
	err = rule.validate()
	test.AssertError(t, err)
}

func TestInt64NotSuperiorTo(t *testing.T) {
	rule := Int64NotSuperiorTo(0)
	// should be OK.
	rule.with("FOO", -10)
	err := rule.validate()
	assert.Nil(t, err)
	// should not be OK.
	rule.with("FOO", 10)
	err = rule.validate()
	test.AssertError(t, err)
}
