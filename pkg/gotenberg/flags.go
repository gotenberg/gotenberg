package gotenberg

import (
	"time"

	"github.com/dlclark/regexp2"
	"github.com/labstack/gommon/bytes"
	flag "github.com/spf13/pflag"
)

// ParsedFlags wraps a [flag.FlagSet] so that retrieving the typed values is
// easier.
type ParsedFlags struct {
	*flag.FlagSet
}

// MustString returns the string value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustString(name string) string {
	val, err := f.GetString(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedString returns the string value of a deprecated flag if it was
// explicitly set or the string value of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedString(deprecated string, newName string) string {
	if f.Changed(deprecated) {
		return f.MustString(deprecated)
	}

	return f.MustString(newName)
}

// MustStringSlice returns the string slice value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustStringSlice(name string) []string {
	val, err := f.GetStringSlice(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedStringSlice returns the string slice value of a deprecated
// flag if it was explicitly set or the string slice value of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedStringSlice(deprecated string, newName string) []string {
	if f.Changed(deprecated) {
		return f.MustStringSlice(deprecated)
	}

	return f.MustStringSlice(newName)
}

// MustBool returns the boolean value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustBool(name string) bool {
	val, err := f.GetBool(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedBool returns the boolean value of a deprecated flag if it was
// explicitly set or the int value of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedBool(deprecated string, newName string) bool {
	if f.Changed(deprecated) {
		return f.MustBool(deprecated)
	}

	return f.MustBool(newName)
}

// MustInt64 returns the int64 value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustInt64(name string) int64 {
	val, err := f.GetInt64(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedInt64 returns the int64 value of a deprecated flag if it was
// explicitly set or the int64 value of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedInt64(deprecated string, newName string) int64 {
	if f.Changed(deprecated) {
		return f.MustInt64(deprecated)
	}

	return f.MustInt64(newName)
}

// MustInt returns the int value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustInt(name string) int {
	val, err := f.GetInt(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedInt returns the int value of a deprecated flag if it was
// explicitly set or the int value of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedInt(deprecated string, newName string) int {
	if f.Changed(deprecated) {
		return f.MustInt(deprecated)
	}

	return f.MustInt(newName)
}

// MustFloat64 returns the float value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustFloat64(name string) float64 {
	val, err := f.GetFloat64(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedFloat64 returns the float value of a deprecated flag if it was
// explicitly set or the float value of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedFloat64(deprecated string, newName string) float64 {
	if f.Changed(deprecated) {
		return f.MustFloat64(deprecated)
	}

	return f.MustFloat64(newName)
}

// MustDuration returns the time.Duration value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustDuration(name string) time.Duration {
	val, err := f.GetDuration(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedDuration returns the time.Duration value of a deprecated flag
// if it was explicitly set or the time.Duration value of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedDuration(deprecated string, newName string) time.Duration {
	if f.Changed(deprecated) {
		return f.MustDuration(deprecated)
	}

	return f.MustDuration(newName)
}

// MustHumanReadableBytesString returns the human-readable bytes string of a
// flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustHumanReadableBytesString(name string) string {
	val, err := f.GetString(name)
	if err != nil {
		panic(err)
	}

	_, err = bytes.Parse(val)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDeprecatedHumanReadableBytesString returns the human-readable bytes
// string of a deprecated flag if it was explicitly set or the human-readable
// bytes string of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedHumanReadableBytesString(deprecated string, newName string) string {
	if f.Changed(deprecated) {
		return f.MustHumanReadableBytesString(deprecated)
	}

	return f.MustHumanReadableBytesString(newName)
}

// MustRegexp returns the regular expression of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustRegexp(name string) *regexp2.Regexp {
	val, err := f.GetString(name)
	if err != nil {
		panic(err)
	}

	return regexp2.MustCompile(val, 0)
}

// MustDeprecatedRegexp returns the regular expression of a deprecated flag if
// it was explicitly set or the regular expression of the new flag.
// It panics if an error occurs.
func (f *ParsedFlags) MustDeprecatedRegexp(deprecated string, newName string) *regexp2.Regexp {
	if f.Changed(deprecated) {
		return f.MustRegexp(deprecated)
	}

	return f.MustRegexp(newName)
}
