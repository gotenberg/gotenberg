package xassert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestStringOfOne(t *testing.T) {
	rule := StringOneOf([]string{"foo", "bar", "baz"})
	// should be OK.
	rule.with("FOO", "foo")
	err := rule.validate()
	assert.Nil(t, err)
	// should not be OK.
	rule.with("FOO", "qux")
	err = rule.validate()
	test.AssertError(t, err)
}

func TestStringStartWith(t *testing.T) {
	rule := StringStartWith("foo")
	// should be OK.
	rule.with("FOO", "foobarfoo")
	err := rule.validate()
	assert.Nil(t, err)
	// should not be OK.
	rule.with("FOO", "qux")
	err = rule.validate()
	test.AssertError(t, err)
}

func TestStringEndWith(t *testing.T) {
	rule := StringEndWith("foo")
	// should be OK.
	rule.with("FOO", "foobarfoo")
	err := rule.validate()
	assert.Nil(t, err)
	// should not be OK.
	rule.with("FOO", "qux")
	err = rule.validate()
	test.AssertError(t, err)
}
