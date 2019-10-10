package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xassert"
	"github.com/thecodingmachine/gotenberg/test"
)

func TestStringArg(t *testing.T) {
	const (
		resourceDirectoryName string = "foo"
		defaultValue          string = "FOO"
	)
	var expected string
	rule := xassert.StringOneOf([]string{"FOO", "BAR"})
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// empty value, result should be equal
	// to the default value.
	r.WithArg(ResultFilenameArgKey, "")
	v, err := r.StringArg(ResultFilenameArgKey, defaultValue)
	assert.Nil(t, err)
	assert.Equal(t, defaultValue, v)
	// result should be equal to given value
	// as it is one of "FOO" and "BAR".
	expected = "BAR"
	r.WithArg(ResultFilenameArgKey, expected)
	v, err = r.StringArg(ResultFilenameArgKey, defaultValue, rule)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as given value is not
	// one of "FOO" and "BAR".
	expected = defaultValue
	r.WithArg(ResultFilenameArgKey, "BAZ")
	v, err = r.StringArg(ResultFilenameArgKey, defaultValue, rule)
	test.AssertError(t, err)
	assert.Equal(t, expected, v)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestInt64Arg(t *testing.T) {
	const (
		resourceDirectoryName string = "foo"
		defaultValue          int64  = 10
	)
	var expected int64
	rule := xassert.Int64NotInferiorTo(6)
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// empty value, result should be equal
	// to the default value.
	r.WithArg(WaitTimeoutArgKey, "")
	v, err := r.Int64Arg(WaitTimeoutArgKey, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to given value
	// but as integer.
	r.WithArg(WaitTimeoutArgKey, "5")
	v, err = r.Int64Arg(WaitTimeoutArgKey, defaultValue)
	expected = 5
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// should not be OK as given value is not
	// a string representation of an integer.
	r.WithArg(WaitTimeoutArgKey, "foo")
	v, err = r.Int64Arg(WaitTimeoutArgKey, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	test.AssertError(t, err)
	// should not be OK as given value does not
	// validate the rule x >= 6.
	r.WithArg(WaitTimeoutArgKey, "5")
	v, err = r.Int64Arg(WaitTimeoutArgKey, defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	test.AssertError(t, err)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestFloat64Arg(t *testing.T) {
	const (
		resourceDirectoryName string  = "foo"
		defaultValue          float64 = 10.0
	)
	var expected float64
	rule := xassert.Float64NotInferiorTo(6.0)
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// empty value, result should be equal
	// to the default value.
	r.WithArg(WaitTimeoutArgKey, "")
	v, err := r.Float64Arg(WaitTimeoutArgKey, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to given value
	// but as float.
	r.WithArg(WaitTimeoutArgKey, "5.5")
	v, err = r.Float64Arg(WaitTimeoutArgKey, defaultValue)
	expected = 5.5
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// should not be OK as given value is not
	// a string representation of a float.
	r.WithArg(WaitTimeoutArgKey, "foo")
	v, err = r.Float64Arg(WaitTimeoutArgKey, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	test.AssertError(t, err)
	// should not be OK as given value does not
	// validate the rule x >= 6.
	r.WithArg(WaitTimeoutArgKey, "5.0")
	v, err = r.Float64Arg(WaitTimeoutArgKey, defaultValue, rule)
	expected = defaultValue
	assert.Equal(t, expected, v)
	test.AssertError(t, err)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestBoolArg(t *testing.T) {
	const (
		resourceDirectoryName string = "foo"
		defaultValue          bool   = true
	)
	var expected bool
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// empty value, result should be equal
	// to the default value.
	r.WithArg(LandscapeArgKey, "")
	v, err := r.BoolArg(LandscapeArgKey, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// result should be equal to given value
	// but as boolean.
	r.WithArg(LandscapeArgKey, "1")
	v, err = r.BoolArg(LandscapeArgKey, defaultValue)
	expected = true
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	r.WithArg(LandscapeArgKey, "true")
	v, err = r.BoolArg(LandscapeArgKey, defaultValue)
	expected = true
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	r.WithArg(LandscapeArgKey, "0")
	v, err = r.BoolArg(LandscapeArgKey, defaultValue)
	expected = false
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	r.WithArg(LandscapeArgKey, "false")
	v, err = r.BoolArg(LandscapeArgKey, defaultValue)
	expected = false
	assert.Equal(t, expected, v)
	assert.Nil(t, err)
	// should not be OK as given value is not
	// a string representation of a boolean.
	r.WithArg(LandscapeArgKey, "foo")
	v, err = r.BoolArg(LandscapeArgKey, defaultValue)
	expected = defaultValue
	assert.Equal(t, expected, v)
	test.AssertError(t, err)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestFpath(t *testing.T) {
	const resourceDirectoryName string = "foo"
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// file exists.
	fpath := test.MergeFpaths(t)[0]
	f, err := os.Open(fpath)
	assert.Nil(t, err)
	filename := "foo.pdf"
	err = r.WithFile(filename, f)
	assert.Nil(t, err)
	absDirPath, err := filepath.Abs(fmt.Sprintf("%s/%s", TemporaryDirectory, resourceDirectoryName))
	assert.Nil(t, err)
	expected := fmt.Sprintf("%s/%s", absDirPath, filename)
	v, err := r.Fpath(filename)
	assert.Nil(t, err)
	assert.Equal(t, expected, v)
	// should not be OK as file does
	// not exist.
	_, err = r.Fpath("bar.pdf")
	test.AssertError(t, err)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestFpaths(t *testing.T) {
	const resourceDirectoryName string = "foo"
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	fpath := test.MergeFpaths(t)[0]
	f, err := os.Open(fpath)
	assert.Nil(t, err)
	defer f.Close() // nolint: errcheck
	filename := "foo.pdf"
	err = r.WithFile(filename, f)
	assert.Nil(t, err)
	// file extension exists.
	absDirPath, err := filepath.Abs(fmt.Sprintf("%s/%s", TemporaryDirectory, resourceDirectoryName))
	assert.Nil(t, err)
	expected := []string{
		fmt.Sprintf("%s/%s", absDirPath, filename),
	}
	fpaths, err := r.Fpaths(".pdf")
	assert.Nil(t, err)
	assert.Equal(t, expected, fpaths)
	// should not be OK as file extension
	// does not exist.
	_, err = r.Fpaths(".html")
	test.AssertError(t, err)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestFcontent(t *testing.T) {
	const (
		resourceDirectoryName string = "foo"
		defaultValue          string = "Gutenberg"
	)
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	filename := "foo.txt"
	// file does not exist, expecting
	// value.
	v, err := r.Fcontent(filename, defaultValue)
	assert.Nil(t, err)
	assert.Equal(t, defaultValue, v)
	// file exists.
	fpath := test.OfficeFpaths(t)[2]
	f, err := os.Open(fpath)
	assert.Nil(t, err)
	defer f.Close() // nolint: errcheck
	err = r.WithFile(filename, f)
	assert.Nil(t, err)
	v, err = r.Fcontent(filename, defaultValue)
	assert.Nil(t, err)
	assert.Contains(t, v, defaultValue)
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}

func TestGetters(t *testing.T) {
	const resourceDirectoryName string = "foo"
	logger := test.DebugLogger()
	r, err := New(logger, resourceDirectoryName)
	assert.Nil(t, err)
	// has arg.
	assert.Equal(t, false, r.HasArg(LandscapeArgKey))
	r.WithArg(LandscapeArgKey, "foo")
	assert.Equal(t, true, r.HasArg(LandscapeArgKey))
	// directory path.
	absDirPath, err := filepath.Abs(fmt.Sprintf("%s/%s", TemporaryDirectory, resourceDirectoryName))
	assert.Nil(t, err)
	assert.Equal(t, absDirPath, r.DirPath())
	// finally...
	err = r.Close()
	assert.Nil(t, err)
}
