package pdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gotenberg/litho/pkg/models"
)

// ErrInvalidStages wraps every error about the stages a request describes: an
// unknown operation, a malformed or missing option, or an order litho cannot
// run. Its message names the stage and the option, so routes return it to the
// client as is.
var ErrInvalidStages = errors.New("invalid stages")

// Stages is an ordered list of operations to run on PDF documents. Build it
// with [ParseStages], which validates every stage and their order.
type Stages []stage

// stage is one operation. Its JSON object carries an "op" naming the
// operation, in litho's wording, and the operation's options in camelCase.
type stage interface {
	// op returns the operation name.
	op() string

	// validate checks the stage's own options.
	validate() error

	// files returns the uploaded filenames the stage reads.
	files() []string
}

// stageFactories maps every operation name to a constructor of its stage.
var stageFactories = map[string]func() stage{
	"merge":                func() stage { return new(mergeStage) },
	"split":                func() stage { return new(splitStage) },
	"select":               func() stage { return new(selectStage) },
	"delete":               func() stage { return new(deleteStage) },
	"insert":               func() stage { return new(insertStage) },
	"rotate":               func() stage { return new(rotateStage) },
	"stamp-text":           func() stage { return new(stampTextStage) },
	"stamp-image":          func() stage { return new(stampImageStage) },
	"stamp-pdf":            func() stage { return new(stampPdfStage) },
	"set-metadata":         func() stage { return new(setMetadataStage) },
	"set-bookmarks":        func() stage { return new(setBookmarksStage) },
	"set-form":             func() stage { return new(setFormStage) },
	"flatten-form":         func() stage { return new(flattenFormStage) },
	"flatten-transparency": func() stage { return new(flattenTransparencyStage) },
	"convert-colors":       func() stage { return new(convertColorsStage) },
	"redact":               func() stage { return new(redactStage) },
	"autotag":              func() stage { return new(autotagStage) },
	"attach":               func() stage { return new(attachStage) },
	"detach":               func() stage { return new(detachStage) },
	"optimize":             func() stage { return new(optimizeStage) },
	"encrypt":              func() stage { return new(encryptStage) },
	"decrypt":              func() stage { return new(decryptStage) },
	"remediate":            func() stage { return new(remediateStage) },
	"sign":                 func() stage { return new(signStage) },
	"timestamp":            func() stage { return new(timestampStage) },
}

// ParseStages decodes and validates a JSON array of stages, such as:
//
//	[{"op":"merge"},{"op":"rotate","angle":90,"pages":"odd"}]
//
// An empty or blank input returns no stages. Every error wraps
// [ErrInvalidStages].
func ParseStages(raw string) (Stages, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var objects []json.RawMessage
	err := json.Unmarshal([]byte(raw), &objects)
	if err != nil {
		return nil, fmt.Errorf("%w: stages must be a JSON array of objects: %w", ErrInvalidStages, err)
	}

	stages := make(Stages, 0, len(objects))
	for i, object := range objects {
		s, err := decodeStage(object)
		if err != nil {
			return nil, fmt.Errorf("%w: stage %d: %w", ErrInvalidStages, i, err)
		}

		stages = append(stages, s)
	}

	err = stages.validateOrder()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidStages, err)
	}

	return stages, nil
}

// decodeStage decodes one stage object, refusing options the operation does
// not know.
func decodeStage(object json.RawMessage) (stage, error) {
	var fields map[string]json.RawMessage
	err := json.Unmarshal(object, &fields)
	if err != nil {
		return nil, fmt.Errorf("must be a JSON object: %w", err)
	}

	var op string
	err = json.Unmarshal(fields["op"], &op)
	if err != nil || op == "" {
		return nil, errors.New(`"op" must name an operation`)
	}

	newStage, ok := stageFactories[op]
	if !ok {
		return nil, fmt.Errorf("unknown operation %q", op)
	}

	delete(fields, "op")

	s := newStage()

	// encoding/json matches keys case-insensitively, so "Title" would set the
	// "title" option. Option names are exact.
	known := optionNames(s)
	for _, key := range sortedKeys(fields) {
		if !known[key] {
			return nil, fmt.Errorf("%s: unknown option %q", op, key)
		}
	}

	options, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("%s: re-encode options: %w", op, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(options))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = s.validate()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s, nil
}

// Files returns the uploaded filenames the stages read, in stage order and
// without duplicates.
func (stages Stages) Files() []string {
	var names []string
	seen := make(map[string]bool)

	for _, s := range stages {
		for _, name := range s.files() {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	return names
}

// validateOrder checks the constraints that span stages. litho refuses an edit
// after a remediation, the output of a signature must not be rewritten, and
// encryption applies to the final write only.
func (stages Stages) validateOrder() error {
	remediated, encrypted, signed := -1, -1, -1

	for i, s := range stages {
		op := s.op()

		switch {
		case signed >= 0 && op != "sign" && op != "timestamp":
			return fmt.Errorf("stage %d: %q cannot follow %q, which must come last", i, op, stages[signed].op())
		case encrypted >= 0 && op != "optimize":
			return fmt.Errorf("stage %d: %q cannot follow \"encrypt\", which must come last", i, op)
		case remediated >= 0 && !afterRemediation(op):
			return fmt.Errorf("stage %d: %q cannot follow \"remediate\": only encrypt, optimize, sign and timestamp may", i, op)
		}

		switch op {
		case "decrypt":
			if i != 0 {
				return fmt.Errorf("stage %d: \"decrypt\" must be the first stage", i)
			}
		case "remediate":
			if remediated >= 0 {
				return fmt.Errorf("stage %d: only one \"remediate\" stage is allowed", i)
			}
			remediated = i
		case "encrypt":
			if encrypted >= 0 {
				return fmt.Errorf("stage %d: only one \"encrypt\" stage is allowed", i)
			}
			encrypted = i
		case "sign", "timestamp":
			if signed < 0 {
				signed = i
			}
		}
	}

	if encrypted >= 0 && signed >= 0 {
		return errors.New(`"encrypt" cannot be combined with "sign" or "timestamp": a signature covers the written bytes`)
	}

	if encrypted >= 0 && remediated >= 0 {
		profile := stages[remediated].(*remediateStage).Profile
		if strings.HasPrefix(profile, "PDF/A") || profile == facturXProfile {
			return fmt.Errorf("%q cannot be encrypted: PDF/A forbids encryption", profile)
		}
	}

	return nil
}

// afterRemediation reports whether op may follow a remediation.
func afterRemediation(op string) bool {
	switch op {
	case "encrypt", "optimize", "sign", "timestamp":
		return true
	default:
		return false
	}
}

// pdfaProfiles maps the remediate profiles to litho's PDF/A profiles.
var pdfaProfiles = map[string]models.PDFAProfile{
	"PDF/A-1b": models.PDFA1b,
	"PDF/A-1a": models.PDFA1a,
	"PDF/A-2b": models.PDFA2b,
	"PDF/A-2u": models.PDFA2u,
	"PDF/A-2a": models.PDFA2a,
	"PDF/A-3b": models.PDFA3b,
	"PDF/A-3u": models.PDFA3u,
	"PDF/A-3a": models.PDFA3a,
	"PDF/A-4":  models.PDFA4,
	"PDF/A-4e": models.PDFA4e,
	"PDF/A-4f": models.PDFA4f,
}

// pdfuaProfiles maps the remediate profiles to litho's PDF/UA profiles.
var pdfuaProfiles = map[string]models.PDFUAProfile{
	"PDF/UA-1": models.PDFUA1,
	"PDF/UA-2": models.PDFUA2,
}

// facturXProfile names the Factur-X remediation: a PDF/A-3 document carrying
// the invoice XML.
const facturXProfile = "Factur-X"

// facturXLevels maps the Factur-X conformance levels to litho's.
var facturXLevels = map[string]models.FacturXLevel{
	string(models.FacturXMinimum):   models.FacturXMinimum,
	string(models.FacturXBasicWL):   models.FacturXBasicWL,
	string(models.FacturXBasic):     models.FacturXBasic,
	string(models.FacturXEN16931):   models.FacturXEN16931,
	string(models.FacturXExtended):  models.FacturXExtended,
	string(models.FacturXXRechnung): models.FacturXXRechnung,
}

// optionNames returns the JSON names of a stage's options.
func optionNames(s stage) map[string]bool {
	t := reflect.TypeOf(s).Elem()
	names := make(map[string]bool, t.NumField())

	for i := range t.NumField() {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if name != "" {
			names[name] = true
		}
	}

	return names
}
