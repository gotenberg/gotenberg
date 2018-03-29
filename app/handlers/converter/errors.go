package converter

type noFileToConvertError struct{}

func (e *noFileToConvertError) Error() string {
	return "There is no file to convert"
}
