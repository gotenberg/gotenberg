package xassert

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/test/internalpkg/xerrortest"
)

func TestString(t *testing.T) {
	const (
		defaultValue string = "FOO"
	)
	var expected string
	rule := StringOneOf([]string{"FOO", "BAR"})
	// empty value, result should be equal
	// to the default value.
	v, err := String("foo", "", defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to given value
	// as it is one of "FOO" and "BAR".
	expected = "FOO"
	v, err = String("foo", expected, defaultValue, rule)
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// should not be OK as given value is not
	// one of "FOO" and "BAR".
	v, err = String("foo", "BAZ", defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
}

func TestStringFromEnv(t *testing.T) {
	const (
		envVar       string = "FOO"
		defaultValue string = "FOO"
	)
	var expected string
	rule := StringOneOf([]string{"FOO", "BAR"})
	// no environment variable set,
	// value should be equal to default value.
	v, err := StringFromEnv(envVar, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to environment variable
	// value as it is one of "FOO" and "BAR".
	expected = "BAR"
	os.Setenv(envVar, expected)
	v, err = StringFromEnv(envVar, defaultValue, rule)
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	os.Unsetenv(envVar)
	// should not be OK as environment variable
	// value is not one of "FOO" and "BAR".
	os.Setenv(envVar, "BAZ")
	v, err = StringFromEnv(envVar, defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	os.Unsetenv(envVar)
}

func TestInt64(t *testing.T) {
	const (
		defaultValue int64 = 10
	)
	var expected int64
	rule := Int64NotInferiorTo(6)
	// empty value, result should be equal
	// to the default value.
	v, err := Int64("foo", "", defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to given value
	// but as integer.
	v, err = Int64("foo", "5", defaultValue)
	expected = 5
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// should not be OK as given value is not
	// a string representation of an integer.
	v, err = Int64("foo", "foo", defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	// should not be OK as given value does not
	// validate the rule x >= 6.
	v, err = Int64("foo", "5", defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
}

func TestInt64FromEnv(t *testing.T) {
	const (
		envVar       string = "FOO"
		defaultValue int64  = 10
	)
	var expected int64
	rule := Int64NotInferiorTo(6)
	// no environment variable set,
	// value should be equal to default value.
	v, err := Int64FromEnv(envVar, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to environment variable
	// value but as integer.
	os.Setenv(envVar, "5")
	v, err = Int64FromEnv(envVar, defaultValue)
	expected = 5
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	os.Unsetenv(envVar)
	// should not be OK as environment variable
	// value is not a string representation of an integer.
	os.Setenv(envVar, "foo")
	v, err = Int64FromEnv(envVar, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	os.Unsetenv(envVar)
	// should not be OK as environment variable
	// value does not validate the rule x >= 6.
	os.Setenv(envVar, "5")
	v, err = Int64FromEnv(envVar, defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	os.Unsetenv(envVar)
}

func TestFloat64(t *testing.T) {
	const defaultValue float64 = 10.0
	var expected float64
	rule := Float64NotInferiorTo(6.0)
	// empty value, result should be equal
	// to the default value.
	v, err := Float64("foo", "", defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to given value
	// but as float.
	v, err = Float64("foo", "5.5", defaultValue)
	expected = 5.5
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// should not be OK as given value is not
	// a string representation of a float.
	v, err = Float64("foo", "foo", defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	// should not be OK as given value does not
	// validate the rule x >= 6.
	v, err = Float64("foo", "5", defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
}

func TestFloat64FromEnv(t *testing.T) {
	const (
		envVar       string  = "FOO"
		defaultValue float64 = 10.0
	)
	var expected float64
	rule := Float64NotInferiorTo(6.0)
	// no environment variable set,
	// value should be equal to default value.
	v, err := Float64FromEnv(envVar, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to environment variable
	// value but as float.
	os.Setenv(envVar, "5.5")
	v, err = Float64FromEnv(envVar, defaultValue)
	expected = 5.5
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	os.Unsetenv(envVar)
	// should not be OK as environment variable
	// value is not a string representation of a float.
	os.Setenv(envVar, "foo")
	v, err = Float64FromEnv(envVar, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	os.Unsetenv(envVar)
	// should not be OK as environment variable
	// value does not validate the rule x >= 6.
	os.Setenv(envVar, "5")
	v, err = Float64FromEnv(envVar, defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	os.Unsetenv(envVar)
}

func TestBool(t *testing.T) {
	const defaultValue bool = true
	var expected bool
	// empty value, result should be equal
	// to the default value.
	v, err := Bool("foo", "", defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to given value
	// but as boolean.
	v, err = Bool("foo", "1", defaultValue)
	expected = true
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	v, err = Bool("foo", "true", defaultValue)
	expected = true
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	v, err = Bool("foo", "0", defaultValue)
	expected = false
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	v, err = Bool("foo", "false", defaultValue)
	expected = false
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// should not be OK as given value is not
	// a string representation of a boolean.
	v, err = Bool("foo", "foo", defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
}

func TestBoolFromEnv(t *testing.T) {
	const (
		envVar       string = "FOO"
		defaultValue bool   = true
	)
	var expected bool
	// no environment variable set,
	// value should be equal to default value.
	v, err := BoolFromEnv(envVar, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to environment variable
	// value but as boolean.
	os.Setenv(envVar, "1")
	v, err = BoolFromEnv(envVar, defaultValue)
	expected = true
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	os.Unsetenv(envVar)
	os.Setenv(envVar, "true")
	v, err = BoolFromEnv(envVar, defaultValue)
	expected = true
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	os.Unsetenv(envVar)
	os.Setenv(envVar, "0")
	v, err = BoolFromEnv(envVar, defaultValue)
	expected = false
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	os.Unsetenv(envVar)
	os.Setenv(envVar, "false")
	v, err = BoolFromEnv(envVar, defaultValue)
	expected = false
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	os.Unsetenv(envVar)
	// should not be OK as environment variable
	// value is not a string representation of a boolean.
	os.Setenv(envVar, "foo")
	v, err = BoolFromEnv(envVar, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	xerrortest.AssertError(t, err)
	os.Unsetenv(envVar)
}
