package standarderror

import (
	"bytes"
	"fmt"
)

const (
	// Internal is a code
	// for internal errors.
	Internal = "internal"
	// Invalid is a code
	// for validation errors.
	Invalid = "invalid"
	// Timeout is a code
	// for timeout errors.
	Timeout = "timeout"
)

// Error defines a standard application
// error.
type Error struct {
	// Code is a machine-readable
	// error code.
	Code string
	// Message is a human-readable
	// message.
	Message string
	// Op is a logical operation.
	Op string
	// Err is a nested error.
	Err error
}

// Error returns the string representation of the error message.
func (err *Error) Error() string {
	var buf bytes.Buffer
	// if wrapping an error, print its Error() message.
	// Otherwise print the error code & message.
	if err.Err != nil {
		buf.WriteString(err.Err.Error())
	} else {
		if err.Code != "" {
			fmt.Fprintf(&buf, "<%s> ", err.Code)
		}
		buf.WriteString(err.Message)
	}
	return buf.String()
}

// Code returns the code of the root error, if available.
// Otherwise returns Internal.
func Code(err error) string {
	if err == nil {
		return ""
	}
	e, ok := err.(*Error)
	if ok && e.Code != "" {
		return e.Code
	}
	if ok && e.Err != nil {
		return Code(e.Err)
	}
	return Internal
}

const defaultMessage = "an internal error has occurred: please contact technical support"

// Message returns the human-readable message of the error, if available.
// Otherwise returns a generic error message.
func Message(err error) string {
	if err == nil {
		return ""
	}
	e, ok := err.(*Error)
	if ok && e.Message != "" {
		return e.Message
	}
	if ok && e.Err != nil {
		return Message(e.Err)
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
	if e.Op != "" {
		fmt.Fprintf(&buf, "%s", e.Op)
	}
	if nestedOp := Op(e.Err); nestedOp != "" {
		fmt.Fprintf(&buf, ": %s", nestedOp)
	}
	return buf.String()
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = error(new(Error))
)
