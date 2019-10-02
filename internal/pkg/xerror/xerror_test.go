package xerror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
Error 1.0: op = "foo"
Error 1.1: op = "bar"
Error 1.2: code = "invalid", op = "baz", message = "nested error"
Error 1.3: message = "root error"
*/
func scenario1() error {
	rootErr := errors.New("root error")
	nestedErr := Invalid("baz", "nested error", rootErr)
	wrappingErr := New("bar", nestedErr)
	return New("foo", wrappingErr)
}

/*
Error 2.0: op = "foo"
Error 2.1: op = "bar"
Error 2.2: code = "timeout", op = "bar", message = "nested error"
*/
func scenario2() error {
	nestedErr := Timeout("bar", "nested error", nil)
	wrappingErr := New("bar", nestedErr)
	return New("foo", wrappingErr)
}

// Error 3.0: code = "", op = "foo"
func scenario3() error {
	return New("foo", nil)
}

func TestError(t *testing.T) {
	// should return the Error 1.3
	// message.
	err := scenario1()
	assert.Equal(t, "root error", err.Error())
	// should return the Error 2.2 message with
	// its code.
	err = scenario2()
	assert.Equal(t, "<timeout> nested error", err.Error())
}

func TestCode(t *testing.T) {
	// should be an empty code if no error.
	assert.Equal(t, "", fmt.Sprintf("%s", Code(nil)))
	// should be the code of Error 1.2.
	err := scenario1()
	assert.Equal(t, InvalidCode, Code(err))
	// should be the code of Error 2.2.
	err = scenario2()
	assert.Equal(t, TimeoutCode, Code(err))
	// should be the default code.
	err = scenario3()
	assert.Equal(t, InternalCode, Code(err))
	err = errors.New("some error")
	assert.Equal(t, InternalCode, Code(err))
}

func TestMessage(t *testing.T) {
	// should be an empty message if no error.
	assert.Equal(t, "", Message(nil))
	// should be the message of Error 1.2.
	err := scenario1()
	assert.Equal(t, "nested error", Message(err))
	// should be the default message.
	err = errors.New("some error")
	assert.Equal(t, defaultMessage, Message(err))
}

func TestOp(t *testing.T) {
	// should be an empty op if no error.
	assert.Equal(t, "", Op(nil))
	// should be the chain of op in this order:
	// Error 1.0 -> Error 1.1 -> Error 1.2.
	err := scenario1()
	assert.Equal(t, "foo: bar: baz", Op(err))
	/*
		should be the chain of op in this order:
		Error 2.0 -> Error 2.1.

		As Error 2.1 and Error 2.2 shares the same
		op, Error 2.2 op is not displayed.
	*/
	err = scenario2()
	assert.Equal(t, "foo: bar", Op(err))
	// should be an empty op if not Error.
	err = errors.New("some error")
	assert.Equal(t, "", Op(err))
}
