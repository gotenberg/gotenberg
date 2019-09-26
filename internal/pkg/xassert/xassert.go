package xassert

import (
	"fmt"
	"os"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

/*
String applies validation on a string.

If string is empty or validation fails,
returns the default value.

The key is used to identify the value.
*/
func String(key, value, defaultValue string, rules ...RuleString) (string, error) {
	const op string = "xassert.String"
	result := defaultValue
	if value != "" {
		result = value
	}
	for _, rule := range rules {
		rule.with(key, result)
		if err := rule.validate(); err != nil {
			return defaultValue, xerror.New(op, err)
		}
	}
	return result, nil
}

/*
StringFromEnv returns the value of given environment
variable or the default value if not found or
validation fails.
*/
func StringFromEnv(envVar, defaultValue string, rules ...RuleString) (string, error) {
	const op string = "xassert.StringFromEnv"
	value := os.Getenv(envVar)
	result, err := String(envVar, value, defaultValue, rules...)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
Int64 tries to convert a string to an int64.

If string is empty, conversion or validation fails,
returns the default value.

The key is used to identify the value.
*/
func Int64(key, value string, defaultValue int64, rules ...RuleInt64) (int64, error) {
	const op string = "xassert.Int64"
	result := defaultValue
	if value != "" {
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return defaultValue, xerror.Invalid(
				op,
				fmt.Sprintf("'%s' is not an integer, got '%s'", key, value),
				err,
			)
		}
		result = parsedValue
	}
	for _, rule := range rules {
		rule.with(key, result)
		if err := rule.validate(); err != nil {
			return defaultValue, xerror.New(op, err)
		}
	}
	return result, nil
}

/*
Int64FromEnv returns the int64 representation of the
value of given environment variable.

If not found, empty, conversion or validation fails,
returns the default value.
*/
func Int64FromEnv(envVar string, defaultValue int64, rules ...RuleInt64) (int64, error) {
	const op string = "xassert.Int64FromEnv"
	value := os.Getenv(envVar)
	result, err := Int64(envVar, value, defaultValue, rules...)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
Float64 tries to convert a string to a float64.

If string is empty, conversion or validation fails,
returns the default value.

The key is used to identify the value.
*/
func Float64(key, value string, defaultValue float64, rules ...RuleFloat64) (float64, error) {
	const op string = "xassert.Float64"
	result := defaultValue
	if value != "" {
		parsedValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return defaultValue, xerror.Invalid(
				op,
				fmt.Sprintf("'%s' is not a float, got '%s'", key, value),
				err,
			)
		}
		result = parsedValue
	}
	for _, rule := range rules {
		rule.with(key, result)
		if err := rule.validate(); err != nil {
			return defaultValue, xerror.New(op, err)
		}
	}
	return result, nil
}

/*
Float64FromEnv returns the float64 representation of the
value of given environment variable.

If not found, empty, conversion or validation fails,
returns the default value.
*/
func Float64FromEnv(envVar string, defaultValue float64, rules ...RuleFloat64) (float64, error) {
	const op string = "xassert.Float64FromEnv"
	value := os.Getenv(envVar)
	result, err := Float64(envVar, value, defaultValue, rules...)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
Bool tries to convert a string to a boolean.

If string is empty or conversion fails, returns the
default value.

The key is used to identify the value.
*/
func Bool(key, value string, defaultValue bool) (bool, error) {
	const op string = "xassert.Bool"
	result := defaultValue
	if value != "" {
		parsedValue, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue, xerror.Invalid(
				op,
				fmt.Sprintf("'%s' is not a boolean, got '%s'", key, value),
				err,
			)
		}
		result = parsedValue
	}
	return result, nil
}

/*
BoolFromEnv returns the boolean representation of the
value of given environment variable.

If not found, empty or conversion fails, returns the
default value.
*/
func BoolFromEnv(envVar string, defaultValue bool) (bool, error) {
	const op string = "xassert.BoolFromEnv"
	value := os.Getenv(envVar)
	result, err := Bool(envVar, value, defaultValue)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}

/*
Bytes tries to convert a string to a int64.

If string is empty or conversion fails, returns the
default value.

The key is used to identify the value.
*/
func Bytes(key, value string, defaultValue int64, rules ...RuleInt64) (int64, error) {
	const op string = "xassert.Bytes"
	result := defaultValue
	if value != "" {
		parsedValue, err := humanize.ParseBigBytes(value)
		if err != nil {
			return defaultValue, xerror.Invalid(
				op,
				fmt.Sprintf("'%s' is not a correct bytes representation, got '%s'", key, value),
				err,
			)
		}
		result = parsedValue.Int64()
	}
	for _, rule := range rules {
		rule.with(key, result)
		if err := rule.validate(); err != nil {
			return defaultValue, xerror.New(op, err)
		}
	}
	return result, nil
}

/*
BytesFromEnv returns the int64 representation of the
value of given environment variable.

If not found, empty or conversion fails, returns the
default value.
*/
func BytesFromEnv(envVar string, defaultValue int64, rules ...RuleInt64) (int64, error) {
	const op string = "xassert.BytesFromEnv"
	value := os.Getenv(envVar)
	result, err := Bytes(envVar, value, defaultValue)
	if err != nil {
		return result, xerror.New(op, err)
	}
	return result, nil
}
