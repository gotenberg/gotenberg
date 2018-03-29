package app

type appConfigError struct{}

func (e *appConfigError) Error() string {
	return "A fatal error occured while setting up the application"
}
