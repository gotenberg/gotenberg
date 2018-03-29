package context

type contentTypeNotFoundError struct{}

func (e *contentTypeNotFoundError) Error() string {
	return "The 'Content-Type' was not found in request context"
}

type converterNotFoundError struct{}

func (e *converterNotFoundError) Error() string {
	return "The converter was not found in request context"
}
