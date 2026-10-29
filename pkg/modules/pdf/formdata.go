package pdf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// StagesFormField is the form field holding the JSON array of stages.
const StagesFormField = "stages"

// FormDataStages parses the optional "stages" form field and resolves the
// uploaded files the stages read. An invalid "stages" field returns a 400
// error right away. A missing file is recorded on the form instead, so that
// the route's own [api.FormData.Validate] reports it with the other form
// errors.
func FormDataStages(form *api.FormData) (Stages, map[string]string, error) {
	var values []string
	form.Strings(StagesFormField, &values)

	raw := ""
	if len(values) > 0 {
		raw = values[0]
	}

	stages, err := ParseStages(raw)
	if err != nil {
		return nil, nil, handleError(err, "")
	}

	return stages, stageFiles(form, stages), nil
}

// stageFiles resolves the uploaded files stages read to their paths on disk.
// A missing file is recorded on the form, and validated with it.
func stageFiles(form *api.FormData, stages Stages) map[string]string {
	names := stages.Files()
	files := make(map[string]string, len(names))

	for _, name := range names {
		var path string
		form.MandatoryPath(name, &path)
		files[name] = path
	}

	return files
}

// inputPDFs returns the uploaded PDFs no stage reads, in upload-name order.
func inputPDFs(form *api.FormData, files map[string]string) ([]string, error) {
	var all []string
	err := form.MandatoryPaths([]string{".pdf"}, &all).Validate()
	if err != nil {
		return nil, fmt.Errorf("validate form data: %w", err)
	}

	referenced := make([]string, 0, len(files))
	for _, path := range files {
		referenced = append(referenced, path)
	}

	inputs := make([]string, 0, len(all))
	for _, path := range all {
		if !slices.Contains(referenced, path) {
			inputs = append(inputs, path)
		}
	}

	if len(inputs) == 0 {
		return nil, api.NewSentinelHttpError(
			http.StatusBadRequest,
			"Invalid form data: no PDF to process; every uploaded PDF is read by a stage",
		)
	}

	return inputs, nil
}

// stageFromForm builds a single stage named op from the flat form fields of a
// single-operation route. Each field carries one option under its JSON name:
// strings and numbers as is, booleans as true or false, and lists or objects
// as JSON. Fields the stage does not know are ignored.
func stageFromForm(op string, form *api.FormData) (Stages, map[string]string, error) {
	newStage, ok := stageFactories[op]
	if !ok {
		return nil, nil, fmt.Errorf("unknown operation %q", op)
	}

	object := map[string]any{"op": op}
	t := reflect.TypeOf(newStage()).Elem()

	for i := range t.NumField() {
		field := t.Field(i)
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" {
			continue
		}

		var values []string
		form.Strings(name, &values)
		if len(values) == 0 || values[0] == "" {
			continue
		}

		value, err := formValue(field.Type, values[0])
		if err != nil {
			return nil, nil, api.WrapError(err, api.NewSentinelHttpError(
				http.StatusBadRequest,
				fmt.Sprintf("Invalid form data: form field '%s' is invalid: %s", name, err),
			))
		}

		object[name] = value
	}

	raw, err := json.Marshal([]any{object})
	if err != nil {
		return nil, nil, fmt.Errorf("encode stage: %w", err)
	}

	stages, err := ParseStages(string(raw))
	if err != nil {
		return nil, nil, api.WrapError(err, api.NewSentinelHttpError(
			http.StatusBadRequest,
			fmt.Sprintf("Invalid form data: %s", strings.TrimPrefix(err.Error(), ErrInvalidStages.Error()+": stage 0: ")),
		))
	}

	return stages, stageFiles(form, stages), nil
}

// formValue converts a form field to the JSON value of a stage option of type
// t.
func formValue(t reflect.Type, value string) (any, error) {
	if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Bool {
		t = t.Elem()
	}

	kind := t.Kind()

	if kind == reflect.String {
		return value, nil
	}

	if kind == reflect.Bool {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("must be true or false, got %q", value)
		}

		return b, nil
	}

	if kind == reflect.Int || kind == reflect.Float64 {
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("must be a number, got %q", value)
		}

		return n, nil
	}

	if !json.Valid([]byte(value)) {
		return nil, fmt.Errorf("must be valid JSON, got %q", value)
	}

	return json.RawMessage(value), nil
}
