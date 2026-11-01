package pdf

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
// The routes have bound every field they take by then: it rejects the others.
func inputPDFs(form *api.FormData, files map[string]string) ([]string, error) {
	var all []string
	err := form.
		MandatoryPaths([]string{".pdf"}, &all).
		RejectUnknown(FormFieldHint).
		Validate()
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
// as JSON. It reads only the fields the stage knows: [inputPDFs] rejects the
// others.
func stageFromForm(op string, form *api.FormData) (Stages, map[string]string, error) {
	newStage, ok := stageFactories[op]
	if !ok {
		return nil, nil, fmt.Errorf("unknown operation %q", op)
	}

	object := map[string]any{"op": op}
	t := reflect.TypeOf(newStage()).Elem()

	for field := range t.Fields() {
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
				fmt.Sprintf("Invalid form data: form field '%s' is invalid (got '%s', %s)", name, values[0], err),
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
// t. Its error states what the option wants.
func formValue(t reflect.Type, value string) (any, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	kind := t.Kind()

	if kind == reflect.String {
		return value, nil
	}

	if kind == reflect.Bool {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, errors.New("want true or false")
		}

		return b, nil
	}

	if kind == reflect.Int {
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.New("want an integer")
		}

		return n, nil
	}

	if kind == reflect.Float64 {
		n, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, errors.New("want a number")
		}

		return n, nil
	}

	if json.Valid([]byte(value)) {
		return json.RawMessage(value), nil
	}

	if kind == reflect.Slice {
		return nil, errors.New("want a JSON array")
	}

	if kind == reflect.Map || kind == reflect.Struct {
		return nil, errors.New("want a JSON object")
	}

	return nil, errors.New("want JSON")
}

// stagesHint is the fix of a Gotenberg 8 field whose work a stage does now.
func stagesHint(stage string) string {
	return fmt.Sprintf("Gotenberg 9 moved it to a stage: add %s to 'stages'", stage)
}

// filesHint is the fix of a Gotenberg 8 file field: Gotenberg 9 reads every
// file from 'files', and a stage names the ones it reads.
func filesHint(stage string) string {
	return fmt.Sprintf("Gotenberg 9 reads every file from 'files': upload it there, and add %s to 'stages'", stage)
}

// gotenberg8Fields maps the Gotenberg 8 post-processing form fields to the
// stage that does their work now. See the upgrade guide.
var gotenberg8Fields = map[string]string{
	"merge":          stagesHint(`{"op":"merge"}`),
	"splitMode":      stagesHint(`{"op":"split","every":2} or {"op":"split","ranges":["1","2-3"]}`),
	"splitSpan":      stagesHint(`{"op":"split","every":2} or {"op":"split","ranges":["1","2-3"]}`),
	"splitUnify":     stagesHint(`{"op":"select","pages":"1-3"}`),
	"pdfua":          stagesHint(`{"op":"remediate","profile":"PDF/UA-1"}`),
	"metadata":       stagesHint(`{"op":"set-metadata","title":"My title"}`),
	"userPassword":   stagesHint(`{"op":"encrypt","userPassword":"secret"}`),
	"ownerPassword":  stagesHint(`{"op":"encrypt","ownerPassword":"secret"}`),
	"embedsMetadata": stagesHint(`{"op":"attach","file":"data.xml"}`),
	"rotateAngle":    stagesHint(`{"op":"rotate","angle":90,"pages":"1"}`),
	"rotatePages":    stagesHint(`{"op":"rotate","angle":90,"pages":"1"}`),
	"flatten":        stagesHint(`{"op":"flatten-form"}`),
	"optimizeImages": imageQualityHint,
	"imageQuality":   imageQualityHint,
}

// imageQualityHint is the fix of the Gotenberg 8 fields that lowered image
// quality: no stage does, so the hint must not pass optimize off as one.
const imageQualityHint = `Gotenberg 9 never lowers image quality: {"op":"optimize"} is lossless; lower the quality where the PDF is made`

// PdfaRemediateStage returns the remediate stage, as JSON, that makes the
// PDF/A profile a Gotenberg 8 'pdfa' field sent, or PDF/A-2b when no remediate
// profile matches it.
func PdfaRemediateStage(profile string) string {
	if _, ok := pdfaProfiles[profile]; !ok {
		profile = "PDF/A-2b"
	}

	return fmt.Sprintf(`{"op":"remediate","profile":"%s"}`, profile)
}

// gotenberg8FieldPrefixes maps the prefixes of Gotenberg 8 field families to
// the stage that does their work now.
var gotenberg8FieldPrefixes = map[string]string{
	"allow":     stagesHint(`{"op":"encrypt","userPassword":"secret","permissions":{"print":false}}`),
	"watermark": stagesHint(`a stamp-text, stamp-image or stamp-pdf stage`),
	"stamp":     stagesHint(`a stamp-text, stamp-image or stamp-pdf stage`),
	"facturx":   stagesHint(`{"op":"remediate","profile":"Factur-X","xml":"invoice.xml"}`),
}

// gotenberg8FileFields maps the Gotenberg 8 file fields to the stage that
// reads their files now. Other file fields are free-form.
var gotenberg8FileFields = map[string]string{
	"embeds":     filesHint(`{"op":"attach","file":"data.xml"}`),
	"watermark":  filesHint(`{"op":"stamp-pdf","source":"annex.pdf"} or {"op":"stamp-image","image":"logo.png"}`),
	"stamp":      filesHint(`{"op":"stamp-pdf","source":"annex.pdf"} or {"op":"stamp-image","image":"logo.png"}`),
	"facturxXml": filesHint(`{"op":"remediate","profile":"Factur-X","xml":"invoice.xml"}`),
}

// FormFieldHint is the [api.FormData.RejectUnknown] hint of the routes that
// run stages: it points a Gotenberg 8 post-processing field, or file field, to
// the stage that replaced it. "stages" is only unknown to the /forms/pdf
// routes but /forms/pdf/pipe.
func FormFieldHint(field, value string, file bool) string {
	if file {
		return gotenberg8FileFields[field]
	}

	if field == StagesFormField {
		return "this route takes no 'stages': send them to /forms/pdf/pipe"
	}

	if field == "pdfa" {
		return stagesHint(PdfaRemediateStage(value))
	}

	hint, ok := gotenberg8Fields[field]
	if ok {
		return hint
	}

	for _, prefix := range sortedKeys(gotenberg8FieldPrefixes) {
		if strings.HasPrefix(field, prefix) {
			return gotenberg8FieldPrefixes[prefix]
		}
	}

	return ""
}
