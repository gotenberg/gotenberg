package config

type fileConfigError struct{}

func (e *fileConfigError) Error() string {
	return "An error occured while trying to load the configuration file"
}

type wrongLoggingLevelError struct{}

func (e *wrongLoggingLevelError) Error() string {
	return "Accepted values for logging level: DEBUG, INFO, WARN, ERROR, FATAL, PANIC"
}

type wrongLoggingFormatError struct{}

func (e *wrongLoggingFormatError) Error() string {
	return "Accepted value for logging format: text, json"
}

type wrongHTMLCommandTemplate struct{}

func (e *wrongHTMLCommandTemplate) Error() string {
	return "An error occured while trying to parse the HTML command's template"
}

type wrongOfficeCommandTemplate struct{}

func (e *wrongOfficeCommandTemplate) Error() string {
	return "An error occured while trying to parse the Office command's template"
}

type wrongMergeCommandTemplate struct{}

func (e *wrongMergeCommandTemplate) Error() string {
	return "An error occured while trying to parse the merge command's template"
}

type readFileError struct{}

func (e *readFileError) Error() string {
	return "An error occured while trying to read the configuration file"
}

type unmarshalError struct{}

func (e *unmarshalError) Error() string {
	return "An error occured while trying to decode the configuration file as YAML"
}
