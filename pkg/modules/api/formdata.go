package api

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// FormData is a helper for validating and hydrating values from a
// "multipart/form-data" request.
//
//	form := ctx.FormData()
type FormData struct {
	values         map[string][]string
	files          map[string]string
	diskToOriginal map[string]string
	fileOrder      map[string]int
	fileBase       map[string]string
	errors         error
}

// Validate returns nil or an error related to the [FormData] values, with a
// [SentinelHttpError] (status code 400, errors' details as a message) wrapped
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
// error if the value is not int, or is empty, or the "key" does not exist.
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
// an error if the value is not float64, is empty, or the "key" does not exist.
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
// populates an error if the value is not time.Duration, or is empty, or the
// "key" does not exist.
//
//	var foo time.Duration
//
//	ctx.FormData().MandatoryDuration("foo", &foo)
func (form *FormData) MandatoryDuration(key string, target *time.Duration) *FormData {
	return form.mustMandatoryField(key, target)
}

// paperSizes maps the paper size names to their width and height in
// millimeters, portrait.
var paperSizes = map[string][2]float64{
	"a3":      {297, 420},
	"a4":      {210, 297},
	"a5":      {148, 210},
	"b4":      {250, 353},
	"b5":      {176, 250},
	"letter":  {215.9, 279.4},
	"legal":   {215.9, 355.6},
	"tabloid": {279.4, 431.8},
}

// PaperSize binds the width and height, in inches, of the paper size a form
// field names: A3, A4, A5, B4, B5, Letter, Legal or Tabloid, in any case. It
// leaves the targets untouched when the field is empty, and populates an
// error for an unknown name.
//
//	var width, height float64
//
//	ctx.FormData().PaperSize("paperSize", &width, &height)
func (form *FormData) PaperSize(key string, width, height *float64) *FormData {
	var value string
	form.mustValue(key, &value, "")

	if value == "" {
		return form
	}

	size, ok := paperSizes[strings.ToLower(value)]
	if !ok {
		form.append(
			fmt.Errorf("form field '%s' is invalid (got '%s', want A3, A4, A5, B4, B5, Letter, Legal or Tabloid)", key, value),
		)
		return form
	}

	*width = size[0] / 25.4
	*height = size[1] / 25.4

	return form
}

// Inches bind a form field to a float64 variable. It populates an error
// if the value cannot be computed back to inches.
//
//	var foo float64
//
//	ctx.FormData().Inches("foo", &foo, 2.0)
func (form *FormData) Inches(key string, target *float64, defaultValue float64) *FormData {
	form.inches(key, target)
	if *target == -math.MaxFloat64 {
		*target = defaultValue
	}
	return form
}

// MandatoryInches binds a form field to a float64 variable. It populates
// an error if the value cannot be computed back to inches, is empty, or the
// "key" does not exist.
//
//	var foo float64
//
//	ctx.FormData().MandatoryInches("foo", &foo)
func (form *FormData) MandatoryInches(key string, target *float64) *FormData {
	val, ok := form.values[key]
	if !ok || val[0] == "" {
		form.append(
			fmt.Errorf("form field '%s' is required", key),
		)
		return form
	}
	return form.inches(key, target)
}

// inches tries to compute a string value to inches.
func (form *FormData) inches(key string, target *float64) *FormData {
	var value string
	form.mustValue(key, &value, "")

	if value == "" {
		*target = -math.MaxFloat64
		return form
	}

	for _, unit := range []string{"pt", "px", "in", "mm", "cm", "pc"} {
		if !strings.HasSuffix(value, unit) {
			continue
		}

		val, err := parseFiniteFloat(strings.TrimSuffix(value, unit))
		if err != nil {
			form.append(
				fmt.Errorf("form field '%s' is invalid (got '%s', resulting to %w)", key, value, err),
			)
			return form
		}

		switch unit {
		case "pt":
			*target = val * (1.0 / 72.0)
		case "px":
			*target = val * (1.0 / 96.0)
		case "in":
			*target = val
		case "mm":
			*target = val * (1.0 / 25.4)
		case "cm":
			*target = val * (1.0 / 2.54)
		case "pc":
			*target = val * (1.0 / 6.0)
		}

		return form
	}

	val, err := parseFiniteFloat(value)
	if err != nil {
		form.append(
			fmt.Errorf("form field '%s' is invalid (got '%s', resulting to %w)", key, value, err),
		)
		return form
	}

	*target = val
	return form
}

// errNotFinite happens when a number form field is NaN or infinite, which
// [strconv.ParseFloat] accepts.
var errNotFinite = errors.New("not a finite number")

// parseFiniteFloat parses value as a finite float64.
func parseFiniteFloat(value string) (float64, error) {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0, errNotFinite
	}

	return val, nil
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
	form.path(filename, target)
	return form
}

// MandatoryPath binds the absolute path of a form data file to a string
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

// Paths bind the absolute paths of form data files, according to a list of
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

// Strings binds every value submitted for key, in submission order. A field
// repeated in the multipart body (e.g. multiple "stampSource") contributes one
// entry per occurrence, which lets a route read parallel field arrays.
func (form *FormData) Strings(key string, target *[]string) *FormData {
	if form.errors != nil {
		return form
	}

	if values, ok := form.values[key]; ok {
		*target = values
	}

	return form
}

// paths bind the absolute paths of form data files, according to a list of
// file extensions, to a string slice variable.
func (form *FormData) paths(extensions []string, target *[]string) *FormData {
	// Collect (originalFilename, diskPath) pairs so that we can sort by
	// original filename rather than by UUID-based disk name.
	// See https://github.com/gotenberg/gotenberg/issues/1500.
	type entry struct {
		original string
		disk     string
	}
	var entries []entry

	for filename, path := range form.files {
		for _, ext := range extensions {
			// See https://github.com/gotenberg/gotenberg/issues/228.
			if strings.ToLower(filepath.Ext(filename)) == ext {
				entries = append(entries, entry{original: filename, disk: path})
			}
		}
	}

	// See https://github.com/gotenberg/gotenberg/issues/139.
	//
	// Sort on the filename as received rather than on the map key. The key
	// carries the suffix uniqueFilename adds when two uploads share a name,
	// and that suffix would otherwise decide the order: "doc (2).pdf" sorts
	// before "doc.pdf". Ordering on the received name keeps the pair adjacent,
	// and the arrival index breaks the tie, so duplicates merge in the order
	// the caller sent them. A file with a unique name is unaffected, since its
	// received name and its key are the same string.
	sort.SliceStable(entries, func(i, j int) bool {
		nameI := form.receivedName(entries[i].disk, entries[i].original)
		nameJ := form.receivedName(entries[j].disk, entries[j].original)
		if nameI != nameJ {
			return gotenberg.AlphanumericSort{nameI, nameJ}.Less(0, 1)
		}

		return form.fileOrder[entries[i].disk] < form.fileOrder[entries[j].disk]
	})

	for _, e := range entries {
		*target = append(*target, e.disk)
	}

	return form
}

// receivedName returns the filename the file at disk arrived under, before
// de-duplication, falling back to fallback.
func (form *FormData) receivedName(disk, fallback string) string {
	base, ok := form.fileBase[disk]
	if ok {
		return base
	}

	return fallback
}

// append adds an error to the list of errors.
func (form *FormData) append(err error) {
	form.errors = errors.Join(form.errors, err)
}

// mustValue binds the target interface with a form field. If the value is
// empty or the "key" does not exist, it binds the default value. Currently,
// only the string, bool, int, float64 and time.Duration types are bindable.
func (form *FormData) mustValue(key string, target any, defaultValue any) *FormData {
	val, ok := form.values[key]

	if !ok || val[0] == "" {
		switch t := target.(type) {
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
func (form *FormData) mustMandatoryField(key string, target any) *FormData {
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
func (form *FormData) mustAssign(key, value string, target any) *FormData {
	var err error

	switch t := target.(type) {
	case *string:
		*t = value
	case *bool:
		*t, err = strconv.ParseBool(value)
	case *int:
		*t, err = strconv.Atoi(value)
	case *float64:
		*t, err = parseFiniteFloat(value)
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

// path binds the absolute path of a form data file to a string variable. It
// reports whether the filename was found: an ambiguous one populates an
// error instead.
func (form *FormData) path(filename string, target *string) bool {
	path, ok := form.files[filename]
	if ok {
		*target = path
		return true
	}

	// See https://github.com/gotenberg/gotenberg/issues/228.
	var matches []string
	for name, path := range form.files {
		nameLowerExt := strings.TrimSuffix(name, filepath.Ext(name)) + strings.ToLower(filepath.Ext(name))
		if nameLowerExt == filename {
			matches = append(matches, path)
		}
	}

	switch len(matches) {
	case 0:
		return false
	case 1:
		*target = matches[0]
		return true
	default:
		form.append(
			fmt.Errorf("form file '%s' is ambiguous: %d files differ only by the case of their extension", filename, len(matches)),
		)
		return true
	}
}

// mandatoryPath binds the absolute path of a form data file to a string
// variable. It populates an error if the file does not exist.
func (form *FormData) mandatoryPath(filename string, target *string) *FormData {
	if form.path(filename, target) {
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
