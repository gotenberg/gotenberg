package api

// Credits: https://www.joeshaw.org/error-handling-in-go-http-applications.

// HttpError is an interface allowing to retrieve the HTTP details of an error.
type HttpError interface {
	HttpError() (int, string)
}

// SentinelHttpError is the HTTP sidekick of an error.
type SentinelHttpError struct {
	status  int
	message string
}

// NewSentinelHttpError creates a [SentinelHttpError]. The message will be sent
// as the response's body if returned from a handler, so make sure to not leak
// sensible information.
func NewSentinelHttpError(status int, message string) SentinelHttpError {
	return SentinelHttpError{
		status:  status,
		message: message,
	}
}

// Error returns the message.
func (err SentinelHttpError) Error() string {
	return err.message
}

// HttpError returns the status and message.
func (err SentinelHttpError) HttpError() (int, string) {
	return err.status, err.message
}

// sentinelWrappedError contains both the error which will logged and the
// sidekick [SentinelHttpError].
type sentinelWrappedError struct {
	error
	sentinel SentinelHttpError
}

func (w sentinelWrappedError) Is(err error) bool {
	return w.sentinel == err
}

func (w sentinelWrappedError) HttpError() (int, string) {
	return w.sentinel.HttpError()
}

// WrapError wraps the given error with a [SentinelHttpError]. The wrapped
// error will be displayed in a log, while the [SentinelHttpError] will be sent
// in the response.
//
//	return api.WrapError(
//	  // This first error will be logged.
//	  fmt.Errorf("my action: %w", err),
//	  // The HTTP error will be sent as a response.
//	  api.NewSentinelHttpError(
//	    http.StatusForbidden,
//	    "Hey, you did something wrong!"
//	  ),
//	)
func WrapError(err error, sentinel SentinelHttpError) error {
	return sentinelWrappedError{
		error:    err,
		sentinel: sentinel,
	}
}

// Interface guards.
var (
	_ error     = (*SentinelHttpError)(nil)
	_ HttpError = (*SentinelHttpError)(nil)
	_ error     = (*sentinelWrappedError)(nil)
	_ HttpError = (*sentinelWrappedError)(nil)
)
