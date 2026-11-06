package pdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
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
	"render":               func() stage { return new(renderStage) },
	"extract-attachments":  func() stage { return new(extractAttachmentsStage) },
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
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		// The decoder would name the Go type it decodes into.
		return nil, fmt.Errorf("%w: stages must be a JSON array of objects", ErrInvalidStages)
	}
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
	if err != nil || fields == nil {
		// The array decoded, so only the element's kind can be wrong, and the
		// decoder would name the Go type it decodes into.
		return nil, errors.New("must be a JSON object")
	}

	var op string
	err = json.Unmarshal(fields["op"], &op)
	if err != nil || op == "" {
		return nil, errors.New(`"op" must name an operation`)
	}

	newStage, ok := stageFactories[op]
	if !ok {
		return nil, fmt.Errorf("unknown operation %q; the operations are %s", op, strings.Join(sortedKeys(stageFactories), ", "))
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
		return nil, fmt.Errorf("%s: %w", op, optionError(op, fields, err))
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
//
// It checks the stages in three passes, so that a list breaking several rules
// gets the most precise message: the positions and duplicates first, then the
// combinations no order allows, then what may follow what.
func (stages Stages) validateOrder() error {
	err := stages.validatePositions()
	if err != nil {
		return err
	}

	err = stages.validateCombinations()
	if err != nil {
		return err
	}

	return stages.validateSequence()
}

// validatePositions checks the stages bound to a position, and the stages
// that may appear once. litho's writer keeps one signer, so a second sign or
// timestamp stage would replace the first rather than add a signature.
func (stages Stages) validatePositions() error {
	seen := make(map[string]int)

	for i, s := range stages {
		op := s.op()
		_, again := seen[op]

		switch {
		case op == "decrypt" && i != 0:
			return fmt.Errorf("stage %d: \"decrypt\" must be the first stage", i)
		// Render and extract-attachments read the final documents, and
		// write none.
		case readsFinal(op) && i != len(stages)-1:
			return fmt.Errorf("stage %d: %q must be the last stage", i, op)
		// litho remediates once, a pair of profiles included.
		case op == "remediate" && again:
			return fmt.Errorf("stage %d: only one \"remediate\" stage is allowed; to remediate to a PDF/A profile and a PDF/UA one together, give one stage both, such as %s", i, pairStage(stages[seen[op]].(*remediateStage), s.(*remediateStage)))
		case (op == "encrypt" || op == "sign" || op == "timestamp") && again:
			return fmt.Errorf("stage %d: only one %q stage is allowed", i, op)
		}

		if !again {
			seen[op] = i
		}
	}

	return nil
}

// pairStage returns, as JSON, the remediate stage that pairs the profiles of
// the stages first and second, when each names one and they form a pair, or
// an example pair otherwise.
func pairStage(first, second *remediateStage) string {
	pair := profileSet{"PDF/A-2b", "PDF/UA-1"}

	if len(first.Profile) == 1 && len(second.Profile) == 1 {
		candidate := profileSet{first.Profile[0], second.Profile[0]}
		if target, pdfua, err := candidate.split(); err == nil {
			pair = profileSet{target, pdfua}
		}
	}

	return fmt.Sprintf(`{"op":"remediate","profile":%s}`, pair)
}

// validateCombinations checks the stages that no order allows together.
func (stages Stages) validateCombinations() error {
	remediated, encrypted, decrypted, signed := -1, -1, false, false

	for i, s := range stages {
		switch s.op() {
		case "remediate":
			remediated = i
		case "encrypt":
			encrypted = i
		case "decrypt":
			decrypted = true
		case "sign", "timestamp":
			signed = true
		}
	}

	// litho signs in an incremental update of the bytes it read, and the
	// signature of an encrypted PDF belongs in its encrypted bytes, which litho
	// cannot write: it refuses to sign a PDF it reads encrypted, even once
	// decrypted in the same run (litho.ErrSignFullRewriteIncompatible).
	if decrypted && signed {
		return errors.New(`"decrypt" cannot be combined with "sign" or "timestamp": a PDF that was encrypted cannot be signed in the request that decrypts it; decrypt it in a request of its own, then sign the result`)
	}

	if encrypted < 0 {
		return nil
	}

	if signed {
		return errors.New(`"encrypt" cannot be combined with "sign" or "timestamp": a signature covers the written bytes`)
	}

	// The PDF/A side of a pair forbids encryption as it does alone.
	if remediated >= 0 {
		profile := stages[remediated].(*remediateStage).Profile
		// The stage validated its profile when decoded.
		target, _, err := profile.split()
		if err != nil {
			return fmt.Errorf("stage %d: %w", remediated, err)
		}
		switch {
		case target == facturXProfile:
			return fmt.Errorf("%s cannot be encrypted: Factur-X is PDF/A-3, which forbids encryption", profile)
		case strings.HasPrefix(target, "PDF/A"):
			return fmt.Errorf("%s cannot be encrypted: PDF/A forbids encryption", profile)
		}
	}

	return nil
}

// validateSequence checks what may follow a signature, an encryption and a
// remediation.
func (stages Stages) validateSequence() error {
	remediated, encrypted, signed := false, false, -1

	for i, s := range stages {
		op := s.op()

		switch {
		case signed >= 0 && op != "sign" && op != "timestamp" && !readsFinal(op):
			return fmt.Errorf("stage %d: %q cannot follow %q: only sign, timestamp, render and extract-attachments may", i, op, stages[signed].op())
		case encrypted && op != "optimize" && !readsFinal(op):
			return fmt.Errorf("stage %d: %q cannot follow \"encrypt\": only optimize, render and extract-attachments may", i, op)
		case remediated && !afterRemediation(op):
			return fmt.Errorf("stage %d: %q cannot follow \"remediate\": only encrypt, optimize, sign, timestamp, render and extract-attachments may", i, op)
		}

		switch op {
		case "remediate":
			remediated = true
		case "encrypt":
			encrypted = true
		case "sign", "timestamp":
			if signed < 0 {
				signed = i
			}
		}
	}

	return nil
}

// remediates reports whether the stages remediate.
func (stages Stages) remediates() bool {
	return slices.ContainsFunc(stages, func(s stage) bool { return s.op() == "remediate" })
}

// afterRemediation reports whether op may follow a remediation.
func afterRemediation(op string) bool {
	switch op {
	case "encrypt", "optimize", "sign", "timestamp":
		return true
	default:
		return readsFinal(op)
	}
}

// readsFinal reports whether op reads the final documents rather than writes
// one: it may follow any stage, and comes last.
func readsFinal(op string) bool {
	return op == "render" || op == "extract-attachments"
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

// maxOptionValueInMessage bounds the option value an error message quotes.
const maxOptionValueInMessage = 64

// optionError rewords an error decoding the options of the stage op, whose
// JSON fields are fields, in the client's terms: encoding/json names the Go
// types and fields of the stages, which mean nothing to a client.
func optionError(op string, fields map[string]json.RawMessage, err error) error {
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) || typeErr.Field == "" {
		return err
	}

	name, _, isNested := strings.Cut(typeErr.Field, ".")

	want := wantedJSON(typeErr.Type)

	// A nested option's value is not at hand: its JSON kind stands for it.
	if isNested {
		return fmt.Errorf("%q is invalid (got %s, want %s)", typeErr.Field, sentJSON(typeErr.Value), want)
	}

	got := strings.TrimSpace(string(fields[name]))
	if len(got) > maxOptionValueInMessage {
		got = got[:maxOptionValueInMessage] + "..."
	}

	return fmt.Errorf("%q is invalid (got %s, want %s)", name, got, want)
}

// sentJSON describes the JSON value [json.UnmarshalTypeError.Value] names,
// such as "string" or "number 1.5".
func sentJSON(value string) string {
	if number, ok := strings.CutPrefix(value, "number "); ok {
		return number
	}

	switch value {
	case "string", "number":
		return "a " + value
	case "object", "array":
		return "an " + value
	case "bool":
		return "a boolean"
	default:
		return value
	}
}

// wantedJSON describes the JSON value an option of type t takes.
func wantedJSON(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	kind := t.Kind()

	switch {
	case kind == reflect.Bool:
		return "true or false"
	case kind >= reflect.Int && kind <= reflect.Uint64:
		return "an integer"
	case kind == reflect.Float32 || kind == reflect.Float64:
		return "a number"
	case kind == reflect.String:
		return "a string"
	case (kind == reflect.Slice || kind == reflect.Array) && t.Elem().Kind() == reflect.String:
		return "a JSON array of strings"
	case kind == reflect.Slice || kind == reflect.Array:
		return "a JSON array"
	case kind == reflect.Map || kind == reflect.Struct:
		return "a JSON object"
	default:
		return "another JSON value"
	}
}

// optionNames returns the JSON names of a stage's options.
func optionNames(s stage) map[string]bool {
	t := reflect.TypeOf(s).Elem()
	names := make(map[string]bool, t.NumField())

	for field := range t.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" {
			names[name] = true
		}
	}

	return names
}
