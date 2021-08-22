package api

// Credits: https://www.joeshaw.org/error-handling-in-go-http-applications.

// HTTPError is an interface allowing to retrieve the HTTP details of an error.
type HTTPError interface {
	HTTPError() (int, string)
}

// SentinelHTTPError is the HTTP sidekick of an error.
type SentinelHTTPError struct {
	status  int
	message string
}

// NewSentinelHTTPError creates a SentinelHTTPError. The message will be sent
// as the response's body if returned from an handler, so make sure to not leak
// sensible information.
func NewSentinelHTTPError(status int, message string) SentinelHTTPError {
	return SentinelHTTPError{
		status:  status,
		message: message,
	}
}

// Error returns the message.
func (err SentinelHTTPError) Error() string {
	return err.message
}

// HTTPError returns the status and message.
func (err SentinelHTTPError) HTTPError() (int, string) {
	return err.status, err.message
}

// sentinelWrappedError contains both the error which will logged and the
// sidekick SentinelHTTPError.
type sentinelWrappedError struct {
	error
	sentinel SentinelHTTPError
}

func (w sentinelWrappedError) Is(err error) bool {
	return w.sentinel == err
}

func (w sentinelWrappedError) HTTPError() (int, string) {
	return w.sentinel.HTTPError()
}

// WrapError wraps the given error with a SentinelHTTPError. The wrapped error
// will be displayed in a log, while the SentinelHTTPError will be sent in the
// response.
//
//  return api.WrapError(
//    // This first error will be logged.
//    fmt.Errorf("my action: %w", err),
//    // The HTTP error will be sent as a response.
//    api.NewSentinelHTTPError(
//      http.StatusForbidden,
//      "Hey, you did something wrong!"
//    ),
//  )
func WrapError(err error, sentinel SentinelHTTPError) error {
	return sentinelWrappedError{
		error:    err,
		sentinel: sentinel,
	}
}

// Interface guards.
var (
	_ error     = (*SentinelHTTPError)(nil)
	_ HTTPError = (*SentinelHTTPError)(nil)
	_ error     = (*sentinelWrappedError)(nil)
	_ HTTPError = (*sentinelWrappedError)(nil)
)
