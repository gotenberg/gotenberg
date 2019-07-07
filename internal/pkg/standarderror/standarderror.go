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
	// print the current operation in our stack, if any.
	if err.Op != "" {
		fmt.Fprintf(&buf, "%s: ", err.Op)
	}
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
	return "An internal error has occurred. Please contact technical support."
}
