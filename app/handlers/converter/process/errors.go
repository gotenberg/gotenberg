package process

type impossibleConversionError struct{}

func (e *impossibleConversionError) Error() string {
	return "Impossible conversion"
}

type commandTimeoutError struct{}

func (e *commandTimeoutError) Error() string {
	return "The command has reached timeout"
}
