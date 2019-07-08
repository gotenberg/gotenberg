package standarderror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func scenario1() error {
	rootErr := errors.New("root error")
	nestedErr := &Error{
		Code:    Invalid,
		Op:      "bar",
		Message: "nested error",
		Err:     rootErr,
	}
	err := &Error{
		Op:  "foo",
		Err: nestedErr,
	}
	return err
}

func scenario2() error {
	nestedErr := &Error{
		Code:    Invalid,
		Op:      "bar",
		Message: "nested error",
	}
	err := &Error{
		Code: Internal,
		Op:   "foo",
		Err:  nestedErr,
	}
	return err
}

func TestError(t *testing.T) {
	err := scenario1()
	assert.Equal(t, "root error", err.Error())
	err = scenario2()
	assert.Equal(t, "<invalid> nested error", err.Error())
}

func TestCode(t *testing.T) {
	assert.Equal(t, "", Code(nil))
	err := scenario1()
	assert.Equal(t, Invalid, Code(err))
	err = scenario2()
	assert.Equal(t, Internal, Code(err))
	err = errors.New("some error")
	assert.Equal(t, Internal, Code(err))
}

func TestMessage(t *testing.T) {
	assert.Equal(t, "", Message(nil))
	err := scenario1()
	assert.Equal(t, "nested error", Message(err))
	err = errors.New("some error")
	assert.Equal(t, defaultMessage, Message(err))
}

func TestOp(t *testing.T) {
	assert.Equal(t, "", Op(nil))
	err := scenario1()
	assert.Equal(t, "foo: bar", Op(err))
	err = errors.New("some error")
	assert.Equal(t, "", Op(err))
}
