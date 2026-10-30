package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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
//	   MandatoryString("foo", &foo).
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

// String binds a form field to a string variable. An absent or empty field
// keeps the variable's current value.
//
//	foo := "bar"
//
//	ctx.FormData().String("foo", &foo)
func (form *FormData) String(key string, target *string) *FormData {
	return bind(form, key, target, false, parseString)
}

// MandatoryString binds a form field to a string variable. It populates an
// error if the value is empty or the "key" does not exist.
//
//	var foo string
//
//	ctx.FormData().MandatoryString("foo", &foo)
func (form *FormData) MandatoryString(key string, target *string) *FormData {
	return bind(form, key, target, true, parseString)
}

// Bool binds a form field to a bool variable. It populates an error if the
// value is not bool. An absent or empty field keeps the variable's current
// value.
//
//	foo := true
//
//	ctx.FormData().Bool("foo", &foo)
func (form *FormData) Bool(key string, target *bool) *FormData {
	return bind(form, key, target, false, strconv.ParseBool)
}

// MandatoryBool binds a form field to a bool variable. It populates an error
// if the value is not bool, is empty, or the "key" does not exist.
//
//	var foo bool
//
//	ctx.FormData().MandatoryBool("foo", &foo)
func (form *FormData) MandatoryBool(key string, target *bool) *FormData {
	return bind(form, key, target, true, strconv.ParseBool)
}

// Int binds a form field to an int variable. It populates an error if the
// value is not int, or if one of the checks rejects it. An absent or empty
// field keeps the variable's current value.
//
//	foo := 2
//
//	ctx.FormData().Int("foo", &foo, api.Min(1))
func (form *FormData) Int(key string, target *int, checks ...func(int) error) *FormData {
	return bind(form, key, target, false, parseCheckedInt(checks))
}

// MandatoryInt binds a form field to an int variable. It populates an error
// if the value is not int, if one of the checks rejects it, if it is empty,
// or if the "key" does not exist.
//
//	var foo int
//
//	ctx.FormData().MandatoryInt("foo", &foo, api.OneOf(1, 2))
func (form *FormData) MandatoryInt(key string, target *int, checks ...func(int) error) *FormData {
	return bind(form, key, target, true, parseCheckedInt(checks))
}

// parseCheckedInt parses an int that every check accepts.
func parseCheckedInt(checks []func(int) error) func(string) (int, error) {
	return func(value string) (int, error) {
		val, err := strconv.Atoi(value)
		if err != nil {
			return 0, err
		}

		for _, check := range checks {
			err = check(val)
			if err != nil {
				return 0, err
			}
		}

		return val, nil
	}
}

// OneOf returns an [FormData.Int] check that accepts only the given values.
func OneOf(values ...int) func(int) error {
	return func(val int) error {
		if slices.Contains(values, val) {
			return nil
		}

		names := make([]string, len(values))
		for i, v := range values {
			names[i] = strconv.Itoa(v)
		}
		if len(names) == 1 {
			return fmt.Errorf("value is not %s", names[0])
		}

		return fmt.Errorf("value is not %s or %s", strings.Join(names[:len(names)-1], ", "), names[len(names)-1])
	}
}

// Min returns an [FormData.Int] check that rejects values below minimum.
func Min(minimum int) func(int) error {
	return func(val int) error {
		if val < minimum {
			return fmt.Errorf("value is inferior to %d", minimum)
		}

		return nil
	}
}

// Max returns an [FormData.Int] check that rejects values above maximum.
func Max(maximum int) func(int) error {
	return func(val int) error {
		if val > maximum {
			return fmt.Errorf("value is superior to %d", maximum)
		}

		return nil
	}
}

// Float64 binds a form field to a float64 variable. It populates an error if
// the value is not a finite float64. An absent or empty field keeps the
// variable's current value.
//
//	foo := 2.0
//
//	ctx.FormData().Float64("foo", &foo)
func (form *FormData) Float64(key string, target *float64) *FormData {
	return bind(form, key, target, false, parseFiniteFloat)
}

// MandatoryFloat64 binds a form field to a float64 variable. It populates an
// error if the value is not a finite float64, is empty, or the "key" does not
// exist.
//
//	var foo float64
//
//	ctx.FormData().MandatoryFloat64("foo", &foo)
func (form *FormData) MandatoryFloat64(key string, target *float64) *FormData {
	return bind(form, key, target, true, parseFiniteFloat)
}

// Duration binds a form field to a time.Duration variable. It populates an
// error if the value is not time.Duration. An absent or empty field keeps the
// variable's current value.
//
//	foo := 2 * time.Second
//
//	ctx.FormData().Duration("foo", &foo)
func (form *FormData) Duration(key string, target *time.Duration) *FormData {
	return bind(form, key, target, false, time.ParseDuration)
}

// MandatoryDuration binds a form field to a time.Duration variable. It
// populates an error if the value is not time.Duration, is empty, or the
// "key" does not exist.
//
//	var foo time.Duration
//
//	ctx.FormData().MandatoryDuration("foo", &foo)
func (form *FormData) MandatoryDuration(key string, target *time.Duration) *FormData {
	return bind(form, key, target, true, time.ParseDuration)
}

// JSON binds a JSON form field to a variable. It populates an error if the
// value does not unmarshal into it. An absent or empty field keeps the
// variable's current value.
//
//	foo := []int{404}
//
//	api.JSON(ctx.FormData(), "foo", &foo)
func JSON[T any](form *FormData, key string, target *T) *FormData {
	return bind(form, key, target, false, func(value string) (T, error) {
		var v T
		err := json.Unmarshal([]byte(value), &v)
		if err != nil {
			return v, fmt.Errorf("unmarshal %s: %w", key, err)
		}
		return v, nil
	})
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
	value := form.value(key)
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

// Inches binds a form field to a float64 variable, in inches. The value is
// a number, optionally suffixed by a unit: pt, px, in, mm, cm or pc. It
// populates an error if the value cannot be converted to inches. An absent
// or empty field keeps the variable's current value.
//
//	foo := 2.0
//
//	ctx.FormData().Inches("foo", &foo)
func (form *FormData) Inches(key string, target *float64) *FormData {
	return bind(form, key, target, false, parseInches)
}

// MandatoryInches binds a form field to a float64 variable, in inches. It
// populates an error if the value cannot be converted to inches, is empty, or
// the "key" does not exist.
//
//	var foo float64
//
//	ctx.FormData().MandatoryInches("foo", &foo)
func (form *FormData) MandatoryInches(key string, target *float64) *FormData {
	return bind(form, key, target, true, parseInches)
}

// inchesPerUnit converts a length unit to inches.
var inchesPerUnit = map[string]float64{
	"pt": 1.0 / 72.0,
	"px": 1.0 / 96.0,
	"in": 1.0,
	"mm": 1.0 / 25.4,
	"cm": 1.0 / 2.54,
	"pc": 1.0 / 6.0,
}

// parseInches converts a length, optionally suffixed by a unit, to inches.
// A bare number is in inches. The units do not suffix one another, so the
// map order does not matter.
func parseInches(value string) (float64, error) {
	for unit, ratio := range inchesPerUnit {
		number, ok := strings.CutSuffix(value, unit)
		if !ok {
			continue
		}

		val, err := parseFiniteFloat(number)
		if err != nil {
			return 0, err
		}

		return val * ratio, nil
	}

	return parseFiniteFloat(value)
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

// parseString is the identity parser of string form fields.
func parseString(value string) (string, error) {
	return value, nil
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
	value := form.value(key)

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
	value := form.value(key)
	if value == "" {
		form.append(fmt.Errorf("form field '%s' is required", key))
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

// Content binds the content of a form data file to a string variable. An
// absent file keeps the variable's current value.
//
//	content := "bar"
//
//	ctx.FormData().Content("foo.txt", &content)
func (form *FormData) Content(filename string, target *string) *FormData {
	var path string
	form.path(filename, &path)

	if path == "" {
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
	if values, ok := form.values[key]; ok {
		*target = slices.Clone(values)
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

// value returns the first value of a form field, or "" if absent.
func (form *FormData) value(key string) string {
	val, ok := form.values[key]
	if !ok {
		return ""
	}

	return val[0]
}

// bind parses the first value of a form field into target. An absent or empty
// field keeps target's current value, unless mandatory, which populates an
// error instead. A value that parse rejects populates an error and leaves
// target untouched.
func bind[T any](form *FormData, key string, target *T, mandatory bool, parse func(string) (T, error)) *FormData {
	value := form.value(key)
	if value == "" {
		if mandatory {
			form.append(fmt.Errorf("form field '%s' is required", key))
		}

		return form
	}

	parsed, err := parse(value)
	if err != nil {
		form.append(
			fmt.Errorf("form field '%s' is invalid (got '%s', resulting to %w)", key, value, err),
		)

		return form
	}

	*target = parsed

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
