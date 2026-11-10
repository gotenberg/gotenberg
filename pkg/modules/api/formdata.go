package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	fields         *formFields
	fileFields     map[string]bool
	errors         error
}

// DownloadFromFormField is the form field listing the files to download. The
// API reads it before the route runs.
const DownloadFromFormField = "downloadFrom"

// formFields records the value form fields a request's routes read. A read
// route may bind fields from concurrent goroutines.
type formFields struct {
	mu   sync.Mutex
	read map[string]bool
}

func newFormFields() *formFields {
	return &formFields{read: make(map[string]bool)}
}

// markRead records that a route read key. A nil set records nothing.
func (f *formFields) markRead(key string) {
	if f == nil {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.read[key] = true
}

// wasRead reports whether a route read key.
func (f *formFields) wasRead(key string) bool {
	if f == nil {
		return false
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.read[key]
}

// RejectUnknown populates an error for every value form field no binding of
// the request read so far, so that a misspelled or removed field answers 400
// rather than being ignored. Call it last, once the route has bound every
// field it takes, then call [FormData.Validate]. downloadFrom, which the API
// reads before the route runs, is never unknown.
//
// hint returns how to fix a field, such as the field or the stage that
// replaced one of an earlier major version, or "" for none. value is the
// field's first value, which the fix may name, and "" for a file field. file
// tells a file field, the multipart field files arrived under, from a value
// field. Routes
// read files by filename, whatever field they arrive under, so a file field
// is rejected only when hint returns a fix for it. A nil hint gives none, and
// rejects no file field.
//
//	err := form.
//	  MandatoryString("url", &url).
//	  RejectUnknown(nil).
//	  Validate()
func (form *FormData) RejectUnknown(hint func(field, value string, file bool) string) *FormData {
	fix := func(field, value string, file bool) string {
		if hint == nil {
			return ""
		}

		return hint(field, value, file)
	}

	for _, key := range slices.Sorted(maps.Keys(form.values)) {
		if key == DownloadFromFormField || form.fields.wasRead(key) {
			continue
		}

		var value string
		if values := form.values[key]; len(values) > 0 {
			value = values[0]
		}

		message := fmt.Sprintf("form field '%s' is unknown to this route", key)
		if h := fix(key, value, false); h != "" {
			message += "; " + h
		}

		form.append(errors.New(message))
	}

	for _, field := range slices.Sorted(maps.Keys(form.fileFields)) {
		h := fix(field, "", true)
		if h == "" {
			continue
		}

		form.append(fmt.Errorf("form file field '%s' is unknown to this route; %s", field, h))
	}

	return form
}

// Sent returns the first value of the value form field key as the client sent
// it, or "" when absent, without binding it: [FormData.RejectUnknown] still
// rejects a field only Sent read. A hint reads the other fields its fix
// depends on with it.
func (form *FormData) Sent(key string) string {
	values := form.values[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
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
//
// The message names each invalid value, which the caller sent. The error it
// wraps leaves the values out (see [invalidFieldError]).
func (form *FormData) Validate() error {
	if form.errors == nil {
		return nil
	}

	return WrapError(
		form.errors,
		NewSentinelHttpError(http.StatusBadRequest, fmt.Sprintf("Invalid form data: %s", clientMessage(form.errors))),
	)
}

// invalidFieldError is a form field whose value does not bind: its key, the
// value sent, and why it does not bind.
//
// Its error leaves the value out, since the access log writes the error of a
// request at the error level, whatever --log-level (LOG_LEVEL) is, and a value
// may carry a credential: a cookie, an Authorization header, a password in a
// list of stages. The 400 the caller receives names the value (see
// [invalidFieldError.clientMessage]). See GHSA-g85v-3gpx-4gw5.
type invalidFieldError struct {
	key   string
	value string
	err   error
}

func (e *invalidFieldError) Error() string {
	return fmt.Sprintf("form field '%s' is invalid (%s)", e.key, e.err)
}

func (e *invalidFieldError) Unwrap() error {
	return e.err
}

// clientMessage returns the error with the value, for the caller who sent it.
func (e *invalidFieldError) clientMessage() string {
	return fmt.Sprintf("form field '%s' is invalid (got '%s', %s)", e.key, e.value, e.err)
}

// clientMessage returns the message of err for the caller: as its error, but
// with the value of each invalid form field it joins.
func clientMessage(err error) string {
	var field *invalidFieldError
	if errors.As(err, &field) && err == error(field) {
		return field.clientMessage()
	}

	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return err.Error()
	}

	messages := make([]string, 0, len(joined.Unwrap()))
	for _, e := range joined.Unwrap() {
		messages = append(messages, clientMessage(e))
	}

	return strings.Join(messages, "\n")
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
	return bind(form, key, target, false, parseBool)
}

// MandatoryBool binds a form field to a bool variable. It populates an error
// if the value is not bool, is empty, or the "key" does not exist.
//
//	var foo bool
//
//	ctx.FormData().MandatoryBool("foo", &foo)
func (form *FormData) MandatoryBool(key string, target *bool) *FormData {
	return bind(form, key, target, true, parseBool)
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
			return 0, errors.New("want an integer")
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

		return fmt.Errorf("want %s", OrList(names))
	}
}

// Min returns an [FormData.Int] check that rejects values below minimum.
func Min(minimum int) func(int) error {
	return func(val int) error {
		if val < minimum {
			return fmt.Errorf("want %d or more", minimum)
		}

		return nil
	}
}

// Max returns an [FormData.Int] check that rejects values above maximum.
func Max(maximum int) func(int) error {
	return func(val int) error {
		if val > maximum {
			return fmt.Errorf("want %d or less", maximum)
		}

		return nil
	}
}

// OrList joins values for an error message: "a", "a or b", "a, b or c".
func OrList(values []string) string {
	if len(values) <= 1 {
		return strings.Join(values, "")
	}

	return strings.Join(values[:len(values)-1], ", ") + " or " + values[len(values)-1]
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
	return bind(form, key, target, false, parseDuration)
}

// MandatoryDuration binds a form field to a time.Duration variable. It
// populates an error if the value is not time.Duration, is empty, or the
// "key" does not exist.
//
//	var foo time.Duration
//
//	ctx.FormData().MandatoryDuration("foo", &foo)
func (form *FormData) MandatoryDuration(key string, target *time.Duration) *FormData {
	return bind(form, key, target, true, parseDuration)
}

// JSON binds a JSON form field to a variable. It populates an error if the
// value does not unmarshal into it, which want describes the expected value
// for. An absent or empty field keeps the variable's current value.
//
//	foo := []int{404}
//
//	api.JSON(ctx.FormData(), "foo", &foo, "a JSON array of status codes, such as [499,599]")
func JSON[T any](form *FormData, key string, target *T, want string) *FormData {
	return bind(form, key, target, false, func(value string) (T, error) {
		var v T
		err := json.Unmarshal([]byte(value), &v)
		if err != nil {
			return v, fmt.Errorf("want %s", want)
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

	w, h, ok := PaperSizeInches(value)
	if !ok {
		form.append(&invalidFieldError{key: key, value: value, err: errors.New("want A3, A4, A5, B4, B5, Letter, Legal or Tabloid")})
		return form
	}

	*width = w
	*height = h

	return form
}

// PaperSizeInches returns the width and height, in inches, portrait, of the
// paper size name: A3, A4, A5, B4, B5, Letter, Legal or Tabloid, in any case.
// ok is false for any other name.
func PaperSizeInches(name string) (width, height float64, ok bool) {
	size, ok := paperSizes[strings.ToLower(name)]
	if !ok {
		return 0, 0, false
	}

	return size[0] / 25.4, size[1] / 25.4, true
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

// errWantLength happens when a length form field is not a number, optionally
// suffixed by a unit.
var errWantLength = errors.New("want a number, optionally followed by pt, px, in, mm, cm or pc")

// parseInches converts a length, optionally suffixed by a unit, to inches.
// A bare number is in inches. The units do not suffix one another, so the
// map order does not matter.
func parseInches(value string) (float64, error) {
	number, ratio := value, 1.0
	for unit, r := range inchesPerUnit {
		n, ok := strings.CutSuffix(value, unit)
		if ok {
			number, ratio = n, r
			break
		}
	}

	val, err := parseFiniteFloat(number)
	if err != nil {
		return 0, errWantLength
	}

	return val * ratio, nil
}

// ParseInches converts a length to inches: a number, optionally followed by
// a unit, pt, px, in, mm, cm or pc, a bare number being in inches. It is the
// parser of [FormData.Inches].
func ParseInches(value string) (float64, error) {
	return parseInches(value)
}

// errWantNumber happens when a number form field is not a finite number. NaN
// and infinities, which [strconv.ParseFloat] accepts, are not.
var errWantNumber = errors.New("want a number")

// parseFiniteFloat parses value as a finite float64.
func parseFiniteFloat(value string) (float64, error) {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
		return 0, errWantNumber
	}

	return val, nil
}

// parseBool parses value as a bool.
func parseBool(value string) (bool, error) {
	val, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.New("want true or false")
	}

	return val, nil
}

// parseDuration parses value as a duration.
func parseDuration(value string) (time.Duration, error) {
	val, err := time.ParseDuration(value)
	if err != nil {
		return 0, errors.New("want a duration such as 500ms or 2s")
	}

	return val, nil
}

// parseString is the identity parser of string form fields.
func parseString(value string) (string, error) {
	return value, nil
}

// Custom helps to define a custom binding function for a form field. The
// error assign returns follows the value in the message, as in "form field
// 'foo' is invalid (got 'bar', want a JSON object)": it states what the field
// wants, or why the value is wrong.
//
//	var foo map[string]string
//
//	ctx.FormData().Custom("foo", func(value string) error {
//	  if value == "" {
//	    foo = map[string]string{"bar": "baz"}
//
//	    return nil
//	  }
//
//	  err := json.Unmarshal([]byte(value), &foo)
//	  if err != nil {
//	    return errors.New("want a JSON object")
//	  }
//
//	  return nil
//	})
func (form *FormData) Custom(key string, assign func(value string) error) *FormData {
	value := form.value(key)

	err := assign(value)
	if err != nil {
		form.append(&invalidFieldError{key: key, value: value, err: err})
	}

	return form
}

// MandatoryCustom helps to define a custom binding function for a form field.
// It populates an error if the value is empty or the "key" does not exist. The
// error assign returns follows the value in the message, as with [FormData.Custom].
//
//	var foo map[string]string
//
//	ctx.FormData().MandatoryCustom("foo", func(value string) error {
//	  err := json.Unmarshal([]byte(value), &foo)
//	  if err != nil {
//	    return errors.New("want a JSON object")
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
		form.append(&invalidFieldError{key: key, value: value, err: err})
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
	form.fields.markRead(key)

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

// value returns the first value of a form field, or "" if absent, and
// records that the route reads it.
func (form *FormData) value(key string) string {
	form.fields.markRead(key)

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
		form.append(&invalidFieldError{key: key, value: value, err: err})

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
