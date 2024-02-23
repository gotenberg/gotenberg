package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/multierr"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// FormData is a helper for validating and hydrating values from a
// "multipart/form-data" request.
//
//	form := ctx.FormData()
type FormData struct {
	values map[string][]string
	files  map[string]string
	errors error
}

// Validate returns nil or an error related to the [FormData] values, with a
// [SentinelHttpError] (status code 400, errors' details as message) wrapped
// inside.
//
//	var foo string
//
//	err := ctx.FormData().
//	   MandatoryString("foo", &foo, "bar").
//	   Validate()
func (form *FormData) Validate() error {
	if form.errors == nil {
		return nil
	}

	return WrapError(
		form.errors,
		NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid form data: %s", form.errors)),
	)
}

// String binds a form field to a string variable.
//
//	var foo string
//
//	ctx.FormData().String("foo", &foo, "bar")
func (form *FormData) String(key string, target *string, defaultValue string) *FormData {
	return form.mustValue(key, target, defaultValue)
}

// MandatoryString binds a form field to a string variable. It populates
// an error if the value is empty or the "key" does not exist.
//
//	var foo string
//
//	ctx.FormData().MandatoryString("foo", &foo)
func (form *FormData) MandatoryString(key string, target *string) *FormData {
	return form.mustMandatoryField(key, target)
}

// Bool binds a form field to a bool variable. It populates an error if
// the value is not bool.
//
//	var foo bool
//
//	ctx.FormData().Bool("foo", &foo, true)
func (form *FormData) Bool(key string, target *bool, defaultValue bool) *FormData {
	return form.mustValue(key, target, defaultValue)
}

// MandatoryBool binds a form field to a bool variable. It populates an
// error if the value is not bool, is empty, or the "key" does not exist.
//
//	var foo bool
//
//	ctx.FormData().MandatoryBool("foo", &foo)
func (form *FormData) MandatoryBool(key string, target *bool) *FormData {
	return form.mustMandatoryField(key, target)
}

// Int binds a form field to an int variable. It populates an error if the
// value is not int.
//
//	var foo int
//
//	ctx.FormData().Int("foo", &foo, 2)
func (form *FormData) Int(key string, target *int, defaultValue int) *FormData {
	return form.mustValue(key, target, defaultValue)
}

// MandatoryInt binds a form field to an int variable. It populates an
// error if the value is not int, is empty, or the "key" does not exist.
//
//	var foo int
//
//	ctx.FormData().MandatoryInt("foo", &foo)
func (form *FormData) MandatoryInt(key string, target *int) *FormData {
	return form.mustMandatoryField(key, target)
}

// Float64 binds a form field to a float64 variable. It populates an error
// if the value is not float64.
//
//	var foo float64
//
//	ctx.FormData().Float64("foo", &foo, 2.0)
func (form *FormData) Float64(key string, target *float64, defaultValue float64) *FormData {
	return form.mustValue(key, target, defaultValue)
}

// MandatoryFloat64 binds a form field to a float64 variable. It populates
// an error if the is not float64, is empty, or the "key" does not exist.
//
//	var foo float64
//
//	ctx.FormData().MandatoryFloat64("foo", &foo)
func (form *FormData) MandatoryFloat64(key string, target *float64) *FormData {
	return form.mustMandatoryField(key, target)
}

// Duration binds a form field to a time.Duration variable. It populates
// an error if the form field is not time.Duration.
//
//	var foo time.Duration
//
//	ctx.FormData().Duration("foo", &foo, time.Duration(2) * time.Second)
func (form *FormData) Duration(key string, target *time.Duration, defaultValue time.Duration) *FormData {
	return form.mustValue(key, target, defaultValue)
}

// MandatoryDuration binds a form field to a time.Duration variable. It
// populates an error if the value is not time.Duration, is empty, or the "key"
// does not exist.
//
//	var foo time.Duration
//
//	ctx.FormData().MandatoryDuration("foo", &foo)
func (form *FormData) MandatoryDuration(key string, target *time.Duration) *FormData {
	return form.mustMandatoryField(key, target)
}

// Custom helps to define a custom binding function for a form field.
//
//	var foo map[string]string
//
//	ctx.FormData().Custom("foo", func(value string) error {
//	  if value == "" {
//	    foo = "bar"
//
//	    return nil
//	  }
//
//	  err := json.Unmarshal([]byte(value), &foo)
//	  if err != nil {
//	    return fmt.Errorf("unmarshal foo: %w", err)
//	  }
//
//	  return nil
//	})
func (form *FormData) Custom(key string, assign func(value string) error) *FormData {
	var value string
	form.mustValue(key, &value, "")

	err := assign(value)
	if err != nil {
		form.append(
			fmt.Errorf("form field '%s' is invalid (got '%s', resulting to %w)", key, value, err),
		)
	}

	return form
}

// MandatoryCustom helps to define a custom binding function for a form field.
// It populates an error if the value is empty or the "key" does not exist.
//
//	var foo map[string]string
//
//	ctx.FormData().MandatoryCustom("foo", func(value string) error {
//	  err := json.Unmarshal([]byte(value), &foo)
//	  if err != nil {
//	    return fmt.Errorf("unmarshal foo: %w", err)
//	  }
//
//	  return nil
//	})
func (form *FormData) MandatoryCustom(key string, assign func(value string) error) *FormData {
	var value string
	form.mustMandatoryField(key, &value)

	if value == "" {
		return form
	}

	err := assign(value)
	if err != nil {
		form.append(
			fmt.Errorf("form field '%s' is invalid (got '%s', resulting to %w)", key, value, err),
		)
	}

	return form
}

// Path binds the absolute path of a form data file to a string variable.
//
//	var path string
//
//	ctx.FormData().Path("foo.txt", &path)
func (form *FormData) Path(filename string, target *string) *FormData {
	return form.path(filename, target)
}

// MandatoryPath binds the absolute path ofa  form data file to a string
// variable. It populates an error if the file does not exist.
//
//	var path string
//
//	ctx.FormData().MandatoryPath("foo.txt", &path)
func (form *FormData) MandatoryPath(filename string, target *string) *FormData {
	return form.mandatoryPath(filename, target)
}

// Content binds the content of a form data file to a string variable.
//
//	var content string
//
//	ctx.FormData().Content("foo.txt", &content, "bar")
func (form *FormData) Content(filename string, target *string, defaultValue string) *FormData {
	var path string
	form.path(filename, &path)

	if path == "" {
		*target = defaultValue

		return form
	}

	return form.readFile(path, filename, target)
}

// MandatoryContent binds the content of a form data file to a string variable.
// It populates an error if the file does not exist.
//
//	var content string
//
//	ctx.FormData().MandatoryContent("foo.txt", &content)
func (form *FormData) MandatoryContent(filename string, target *string) *FormData {
	var path string
	form.mandatoryPath(filename, &path)

	if path == "" {
		return form
	}

	return form.readFile(path, filename, target)
}

// Paths binds the absolute paths of form data files, according to a list of
// file extensions, to a string slice variable.
//
//	var paths []string
//
//	ctx.FormData().Paths([]string{".txt"}, &paths)
func (form *FormData) Paths(extensions []string, target *[]string) *FormData {
	return form.paths(extensions, target)
}

// MandatoryPaths binds the absolute paths of form data files, according to a
// list of file extensions, to a string slice variable. It populates an error
// if there is no file for given file extensions.
//
//	var paths []string
//
//	ctx.FormData().MandatoryPaths([]string{".txt"}, &paths)
func (form *FormData) MandatoryPaths(extensions []string, target *[]string) *FormData {
	form.paths(extensions, target)

	if len(*target) > 0 {
		return form
	}

	form.append(
		fmt.Errorf("no form file found for extensions: %v", extensions),
	)

	return form
}

// paths binds the absolute paths of form data files, according to a list of
// file extensions, to a string slice variable.
func (form *FormData) paths(extensions []string, target *[]string) *FormData {
	for filename, path := range form.files {
		for _, ext := range extensions {
			// See https://github.com/gotenberg/gotenberg/issues/228.
			if strings.ToLower(filepath.Ext(filename)) == ext {
				*target = append(*target, path)
			}
		}
	}

	// See https://github.com/gotenberg/gotenberg/issues/139.
	sort.Sort(gotenberg.AlphanumericSort(*target))

	return form
}

// append adds an error to the list of errors.
func (form *FormData) append(err error) {
	form.errors = multierr.Append(form.errors, err)
}

// mustValue binds the target interface with a form field. If the value is
// empty or the "key" does not exist, it binds the default value. Currently,
// only the string, bool, int, float64 and time.Duration types are bindable.
func (form *FormData) mustValue(key string, target interface{}, defaultValue interface{}) *FormData {
	val, ok := form.values[key]

	if !ok || val[0] == "" {
		switch t := (target).(type) {
		case *string:
			*t = defaultValue.(string)
		case *bool:
			*t = defaultValue.(bool)
		case *int:
			*t = defaultValue.(int)
		case *float64:
			*t = defaultValue.(float64)
		case *time.Duration:
			*t = defaultValue.(time.Duration)
		default:
			panic("target type not supported")
		}

		return form
	}

	return form.mustAssign(key, val[0], target)
}

// mustMandatoryField binds the target interface with a form field. It
// populates an error if the value is empty or the "key" does not exist.
// Currently, only the string, bool, int, float64 and time.Duration types are
// bindable.
func (form *FormData) mustMandatoryField(key string, target interface{}) *FormData {
	val, ok := form.values[key]

	if !ok || val[0] == "" {
		form.append(
			fmt.Errorf("form field '%s' is required", key),
		)

		return form
	}

	form.mustAssign(key, val[0], target)

	return form
}

// mustAssign parses the string value and tries to convert it to the target
// interface real type. Currently, only the string, bool, int, float64 and
// time.Duration types are bindable.
func (form *FormData) mustAssign(key, value string, target interface{}) *FormData {
	var err error

	switch t := (target).(type) {
	case *string:
		*t = value
	case *bool:
		*t, err = strconv.ParseBool(value)
	case *int:
		*t, err = strconv.Atoi(value)
	case *float64:
		*t, err = strconv.ParseFloat(value, 64)
	case *time.Duration:
		*t, err = time.ParseDuration(value)
	default:
		panic("target type not supported")
	}

	if err != nil {
		form.append(
			fmt.Errorf("form field '%s' is invalid (got '%s', resulting to %w)", key, value, err),
		)
	}

	return form
}

// path binds the absolute path of a form data file to a string variable.
func (form *FormData) path(filename string, target *string) *FormData {
	for name, path := range form.files {
		// See https://github.com/gotenberg/gotenberg/issues/228.
		nameLowerExt := strings.TrimSuffix(name, filepath.Ext(name)) + strings.ToLower(filepath.Ext(name))
		if name == filename || nameLowerExt == filename {
			*target = path
			return form
		}
	}

	return form
}

// mandatoryPath binds the absolute path of a form data file to a string
// variable. It populates an error if the file does not exist.
func (form *FormData) mandatoryPath(filename string, target *string) *FormData {
	form.path(filename, target)

	if *target != "" {
		return form
	}

	form.append(
		fmt.Errorf("form file '%s' is required", filename),
	)

	return form
}

// readFile binds the content of a file to a string variable. It populates an
// error if it fails to read the file content.
func (form *FormData) readFile(path, filename string, target *string) *FormData {
	b, err := os.ReadFile(path)
	if err != nil {
		form.append(
			fmt.Errorf("form file '%s' is invalid (%w)", filename, err),
		)

		return form
	}

	*target = string(b)

	return form
}
