package pdf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// serve runs route against a request context holding values and the named
// fixtures, and returns the context, the recorder and the handler's error.
func serve(t *testing.T, route api.Route, values map[string][]string, fixtures ...string) (*api.Context, *httptest.ResponseRecorder, error) {
	t.Helper()

	ctx, _ := testContext(t, fixtures...)
	mock := &api.ContextMock{Context: ctx}
	mock.SetValues(values)

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, route.Path, nil), rec)
	c.Set("context", ctx)

	return ctx, rec, route.Handler(c)
}

func TestStageRoute(t *testing.T) {
	for _, tc := range []struct {
		name       string
		op         string
		values     map[string][]string
		fixtures   []string
		wantStatus int
		wantBody   string
		wantOut    int
	}{
		{
			name:     "flat options",
			op:       "rotate",
			values:   map[string][]string{"angle": {"90"}, "pages": {"1"}},
			fixtures: []string{"page_1.pdf", "page_2.pdf"},
			wantOut:  2,
		},
		{
			name:     "JSON option",
			op:       "encrypt",
			values:   map[string][]string{"userPassword": {"u"}, "permissions": {`{"print":false}`}},
			fixtures: []string{"page_1.pdf"},
			wantOut:  1,
		},
		{
			name:     "files read by the stage are not inputs",
			op:       "stamp-pdf",
			values:   map[string][]string{"source": {"page_2.pdf"}},
			fixtures: []string{"page_1.pdf", "page_2.pdf"},
			wantOut:  1,
		},
		{
			name:       "invalid number",
			op:         "rotate",
			values:     map[string][]string{"angle": {"ninety"}},
			fixtures:   []string{"page_1.pdf"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid form data: form field 'angle' is invalid (got 'ninety', want an integer)",
		},
		{
			name:       "invalid option",
			op:         "rotate",
			values:     map[string][]string{"angle": {"45"}},
			fixtures:   []string{"page_1.pdf"},
			wantStatus: http.StatusBadRequest,
			wantBody:   `Invalid form data: rotate: "angle" must be a multiple of 90`,
		},
		{
			name:       "missing file",
			op:         "stamp-image",
			values:     map[string][]string{"image": {"logo.png"}},
			fixtures:   []string{"page_1.pdf"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "form file 'logo.png' is required",
		},
		{
			name:       "only stage files",
			op:         "stamp-pdf",
			values:     map[string][]string{"source": {"page_2.pdf"}},
			fixtures:   []string{"page_2.pdf"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "every uploaded PDF is read by a stage",
		},
		{
			name:     "render answers the images only",
			op:       "render",
			values:   map[string][]string{"dpi": {"20"}},
			fixtures: []string{"page_1.pdf", "pages_3.pdf"},
			wantOut:  4,
		},
		{
			name:       "render a page past the end",
			op:         "render",
			values:     map[string][]string{"pages": {"2"}},
			fixtures:   []string{"page_1.pdf"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Stage 'render' failed on 'page_1.pdf'",
		},
		{
			name:       "page past the end",
			op:         "rotate",
			values:     map[string][]string{"angle": {"90"}, "pages": {"3"}},
			fixtures:   []string{"page_1.pdf"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Stage 'rotate' failed on 'page_1.pdf'",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, err := serve(t, stageRoute(testProcessor(), tc.op), tc.values, tc.fixtures...)

			if tc.wantStatus != 0 {
				status, message := api.ParseError(err)
				if status != tc.wantStatus || !strings.Contains(message, tc.wantBody) {
					t.Fatalf("got %d %q, want %d containing %q", status, message, tc.wantStatus, tc.wantBody)
				}
				return
			}

			if err != nil {
				t.Fatalf("handler: %v", err)
			}

			if got := len((&api.ContextMock{Context: ctx}).OutputPaths()); got != tc.wantOut {
				t.Fatalf("%d outputs, want %d", got, tc.wantOut)
			}
		})
	}
}

func TestPipeRoute(t *testing.T) {
	_, _, err := serve(t, pipeRoute(testProcessor()), nil, "page_1.pdf")
	status, message := api.ParseError(err)
	if status != http.StatusBadRequest || !strings.Contains(message, "'stages' is required") {
		t.Fatalf("no stages: got %d %q", status, message)
	}

	_, _, err = serve(t, pipeRoute(testProcessor()), map[string][]string{"stages": {`[{"op":"nope"}]`}}, "page_1.pdf")
	status, message = api.ParseError(err)
	if status != http.StatusBadRequest || !strings.Contains(message, `form field 'stages' is invalid: stage 0: unknown operation "nope"`) {
		t.Fatalf("unknown op: got %d %q", status, message)
	}

	ctx, _, err := serve(t, pipeRoute(testProcessor()), map[string][]string{"stages": {`[{"op":"merge"},{"op":"set-metadata","title":"Both"}]`}}, "page_1.pdf", "page_2.pdf")
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	if got := len((&api.ContextMock{Context: ctx}).OutputPaths()); got != 1 {
		t.Fatalf("%d outputs, want 1 merged", got)
	}
}

func TestReadRoutes(t *testing.T) {
	// pages_12.pdf carries neither an /Info dictionary nor an XMP packet.
	_, rec, err := serve(t, metadataRoute(testProcessor()), nil, "pages_12.pdf")
	if err != api.ErrNoOutputFile {
		t.Fatalf("handler: %v, want ErrNoOutputFile after writing JSON", err)
	}

	if got := strings.TrimSpace(rec.Body.String()); got != `{"pages_12.pdf":{}}` {
		t.Fatalf("metadata = %s, want an empty object for a PDF without metadata", got)
	}

	_, rec, err = serve(t, infoRoute(testProcessor()), nil, "pages_12.pdf")
	if err != api.ErrNoOutputFile {
		t.Fatalf("handler: %v, want ErrNoOutputFile after writing JSON", err)
	}

	var info map[string]documentInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}

	if info["pages_12.pdf"].Pages != 12 {
		t.Fatalf("info = %+v, want 12 pages", info)
	}

	_, _, err = serve(t, validateRoute(testProcessor()), map[string][]string{"profile": {"PDF/X-4"}}, "page_1.pdf")
	status, message := api.ParseError(err)
	if status != http.StatusBadRequest || !strings.Contains(message, "form field 'profile' must be one of") {
		t.Fatalf("bad profile: got %d %q", status, message)
	}

	_, rec, err = serve(t, validateRoute(testProcessor()), map[string][]string{"profile": {"PDF/A-2b"}}, "page_1.pdf")
	if err != api.ErrNoOutputFile {
		t.Fatalf("handler: %v", err)
	}

	var report map[string]validation
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}

	if got := report["page_1.pdf"]; got.Compliant || len(got.Violations) == 0 {
		t.Fatalf("a plain PDF reported as PDF/A-2b compliant: %+v", got)
	}
	_, rec, err = serve(t, validateRoute(testProcessor()), map[string][]string{"profile": {"Factur-X"}}, "page_1.pdf")
	if err != api.ErrNoOutputFile {
		t.Fatalf("handler: %v", err)
	}

	report = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}

	got := report["page_1.pdf"]
	if got.Compliant || len(got.Violations) == 0 {
		t.Fatalf("a plain PDF reported as a Factur-X invoice: %+v", got)
	}

	for _, v := range got.Violations {
		if v.Clause == "" || v.Severity == "" {
			t.Fatalf("Factur-X violation without clause or severity: %+v", v)
		}
	}
}

// jsonNames returns the JSON names of the fields of v's struct type, in order.
func jsonNames(v any) []string {
	typ := reflect.TypeOf(v)
	names := make([]string, typ.NumField())
	for i := range names {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		names[i] = name
	}

	return names
}

// The metadata route answers the options of the set-metadata stage, so that an
// answer feeds the stage as it is.
func TestDocumentMetadataMatchesSetMetadata(t *testing.T) {
	got, want := jsonNames(documentMetadata{}), jsonNames(setMetadataStage{})
	if !slices.Equal(got, want) {
		t.Fatalf("documentMetadata keys = %v, want the set-metadata options %v", got, want)
	}
}

// readMetadata answers the metadata route on the PDF at path.
func readMetadata(t *testing.T, processor *Processor, path string) documentMetadata {
	t.Helper()

	ctx, _ := testContext(t)
	(&api.ContextMock{Context: ctx}).SetFiles(map[string]string{"doc.pdf": path})

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/forms/pdf/metadata", nil), rec)
	c.Set("context", ctx)

	err := metadataRoute(processor).Handler(c)
	if err != api.ErrNoOutputFile {
		t.Fatalf("handler: %v, want ErrNoOutputFile after writing JSON", err)
	}

	var answer map[string]documentMetadata
	err = json.Unmarshal(rec.Body.Bytes(), &answer)
	if err != nil || len(answer) != 1 {
		t.Fatalf("decode %s: %v, want the answer for one PDF", rec.Body.String(), err)
	}

	// The answer is keyed by the PDF's filename on disk.
	for _, metadata := range answer {
		return metadata
	}

	return documentMetadata{}
}

// setMetadata runs the set-metadata stage, its options raw, on a copy of
// pages_12.pdf, which carries no metadata, and returns the output's path.
func setMetadata(t *testing.T, processor *Processor, raw []byte) string {
	t.Helper()

	var options map[string]any
	err := json.Unmarshal(raw, &options)
	if err != nil {
		t.Fatal(err)
	}
	options["op"] = "set-metadata"

	stages, err := json.Marshal([]any{options})
	if err != nil {
		t.Fatal(err)
	}

	ctx, paths := testContext(t, "pages_12.pdf")
	outputs, err := processor.Run(ctx, mustStages(t, string(stages)), []string{paths["pages_12.pdf"]}, nil)
	if err != nil {
		t.Fatalf("set-metadata %s: %v", stages, err)
	}

	return outputs[0]
}

// Reading the metadata, feeding the answer to set-metadata and reading again
// gives the same answer.
func TestMetadataRoundTrip(t *testing.T) {
	processor := testProcessor()

	want := documentMetadata{
		Title:        "Title",
		Author:       "Author",
		Subject:      "Subject",
		Keywords:     "first, second",
		Creator:      "Creator",
		Producer:     "Producer",
		CreationDate: "D:20250101120000Z",
		ModDate:      "D:20250102120000Z",
	}

	options, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	first := readMetadata(t, processor, setMetadata(t, processor, options))
	if first != want {
		t.Fatalf("first read = %+v, want %+v", first, want)
	}

	answer, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}

	second := readMetadata(t, processor, setMetadata(t, processor, answer))
	if second != first {
		t.Fatalf("second read = %+v, want %+v", second, first)
	}
}
