package xerror

import (
	"bytes"
	"fmt"
	"strings"
)

// ErrorCode is machine-readable error code.
type ErrorCode string

const (
	// InternalCode is an internal error.
	InternalCode ErrorCode = "internal"
	// InvalidCode occurs when a validation
	// failed.
	InvalidCode ErrorCode = "invalid"
	// TimeoutCode occurs when something
	// timed out.
	TimeoutCode ErrorCode = "timeout"
)

// Error defines our standard application
// error.
type Error struct {
	code    ErrorCode
	message string
	op      string
	err     error
}

// Error returns the string representation of the error message.
func (e Error) Error() string {
	var buf bytes.Buffer
	// if wrapping an error, print its Error() message.
	// Otherwise print the error code & message.
	if e.err != nil {
		buf.WriteString(e.err.Error())
	} else {
		if e.code != "" {
			fmt.Fprintf(&buf, "<%s> ", e.code)
		}
		buf.WriteString(e.message)
	}
	return buf.String()
}

/*
New returns a xerror.Error.

Should be used for wrapping an error
at the end of a function.
*/
func New(op string, previous error) error {
	return &Error{
		op:  op,
		err: previous,
	}
}

/*
Invalid returns a xerror.Error.

Should be used when an input
is wrong.
*/
func Invalid(op, message string, previous error) error {
	return &Error{
		code:    InvalidCode,
		message: message,
		op:      op,
		err:     previous,
	}
}

/*
Timeout returns a xerror.Error.

Should be used when a timeout occurs.
*/
func Timeout(op, message string, previous error) error {
	return &Error{
		code:    TimeoutCode,
		message: message,
		op:      op,
		err:     previous,
	}
}

// Code returns the code of the root error, if available.
// Otherwise returns InternalCode.
func Code(err error) ErrorCode {
	if err == nil {
		return ""
	}
	e, ok := err.(*Error)
	if ok && e.code != "" {
		return e.code
	}
	if ok && e.err != nil {
		return Code(e.err)
	}
	return InternalCode
}

const defaultMessage string = "an internal error has occurred: please contact technical support"

// Message returns the human-readable message of the error, if available.
// Otherwise returns a generic error message.
func Message(err error) string {
	if err == nil {
		return ""
	}
	e, ok := err.(*Error)
	if ok && e.message != "" {
		return e.message
	}
	if ok && e.err != nil {
		return Message(e.err)
	}
	return defaultMessage
}

// Op returns the logical operation of the error, if available.
// Otherwise returns an empty string.
func Op(err error) string {
	if err == nil {
		return ""
	}
	e, ok := err.(*Error)
	if !ok {
		return ""
	}
	var buf bytes.Buffer
	nestedOp := Op(e.err)
	if nestedOp != "" {
		// we want to avoid having the same op chained.
		if e.op != "" && !strings.Contains(nestedOp, e.op) {
			fmt.Fprintf(&buf, "%s: %s", e.op, nestedOp)
		} else {
			fmt.Fprintf(&buf, "%s", nestedOp)
		}
	} else if e.op != "" {
		fmt.Fprintf(&buf, "%s", e.op)
	}
	return buf.String()
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = error(new(Error))
)
