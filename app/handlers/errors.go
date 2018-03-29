package handlers

type requestHasNoContentError struct{}

func (e *requestHasNoContentError) Error() string {
	return "Request has not content"
}
