package gotenberg

import (
	"regexp"
	"time"

	"github.com/labstack/gommon/bytes"
	flag "github.com/spf13/pflag"
)

// ParsedFlags wraps a flag.FlagSet so that retrieving the typed values is
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

// MustStringSlice returns the string slice value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustStringSlice(name string) []string {
	val, err := f.GetStringSlice(name)
	if err != nil {
		panic(err)
	}

	return val
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

// MustInt returns the int value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustInt(name string) int {
	val, err := f.GetInt(name)
	if err != nil {
		panic(err)
	}

	return val
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

// MustDuration returns the time.Duration value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustDuration(name string) time.Duration {
	val, err := f.GetDuration(name)
	if err != nil {
		panic(err)
	}

	return val
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

// MustRegexp returns the regular expression of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustRegexp(name string) *regexp.Regexp {
	val, err := f.GetString(name)
	if err != nil {
		panic(err)
	}

	return regexp.MustCompile(val)
}
