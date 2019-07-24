package xassert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/xerrortest"
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
	xerrortest.AssertError(t, err)
}
