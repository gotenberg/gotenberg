package pdf

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseStages(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     string
		wantOps []string
		wantErr string
	}{
		{name: "blank", raw: "  ", wantOps: nil},
		{name: "empty array", raw: "[]", wantOps: []string{}},
		{
			name:    "every operation",
			raw:     `[{"op":"decrypt","password":"x"},{"op":"merge"},{"op":"select","pages":"1"},{"op":"rotate","angle":90},{"op":"stamp-text","text":"t","color":"#ff0000","font":"Courier"},{"op":"set-metadata","title":"T"},{"op":"remediate","profile":"PDF/UA-1"},{"op":"encrypt","userPassword":"u"},{"op":"optimize"}]`,
			wantOps: []string{"decrypt", "merge", "select", "rotate", "stamp-text", "set-metadata", "remediate", "encrypt", "optimize"},
		},
		{name: "not an array", raw: `{"op":"merge"}`, wantErr: "stages must be a JSON array of objects"},
		{name: "not an object", raw: `[1]`, wantErr: "stage 0: must be a JSON object"},
		{name: "null stage", raw: `[null]`, wantErr: "stage 0: must be a JSON object"},
		{name: "missing op", raw: `[{"angle":90}]`, wantErr: `"op" must name an operation`},
		{name: "unknown op", raw: `[{"op":"shred"}]`, wantErr: `unknown operation "shred"`},
		{name: "unknown option", raw: `[{"op":"rotate","angle":90,"degrees":90}]`, wantErr: `rotate: unknown option "degrees"`},
		{name: "option names are exact", raw: `[{"op":"set-metadata","Title":"T"}]`, wantErr: `set-metadata: unknown option "Title"`},
		{name: "wrong option type", raw: `[{"op":"rotate","angle":"ninety"}]`, wantErr: `stage 0: rotate: "angle" is invalid (got "ninety", want an integer)`},
		{name: "fraction for an integer", raw: `[{"op":"rotate","angle":90.5}]`, wantErr: `rotate: "angle" is invalid (got 90.5, want an integer)`},
		{name: "bool for a string", raw: `[{"op":"set-metadata","title":true}]`, wantErr: `set-metadata: "title" is invalid (got true, want a string)`},
		{name: "string for a bool", raw: `[{"op":"optimize","dedup":"yes"}]`, wantErr: `optimize: "dedup" is invalid (got "yes", want true or false)`},
		{name: "object for a list", raw: `[{"op":"set-bookmarks","bookmarks":{"title":"A"}}]`, wantErr: `set-bookmarks: "bookmarks" is invalid (got {"title":"A"}, want a JSON array)`},
		{name: "nested option type", raw: `[{"op":"redact","regions":[{"page":"1","x":0,"y":0,"width":1,"height":1}]}]`, wantErr: `redact: "regions.0.page" is invalid (got a string, want an integer)`},
		{name: "check box state as a bool", raw: `[{"op":"set-form","fields":{"agree":true}}]`, wantErr: `set-form: "fields" values must be strings, or arrays of strings for a list that selects several; a check box takes its state name, such as "Yes" or "Off"`},
		{name: "list selections as numbers", raw: `[{"op":"set-form","fields":{"tags":[1,2]}}]`, wantErr: `set-form: "fields" values must be strings, or arrays of strings`},
		{name: "fields as a string", raw: `[{"op":"set-form","fields":"name"}]`, wantErr: `set-form: "fields" is invalid (got "name", want a JSON object)`},
		{name: "list selections", raw: `[{"op":"set-form","fields":{"name":"x","tags":["a","c"]}}]`, wantOps: []string{"set-form"}},
		{name: "invalid angle", raw: `[{"op":"rotate","angle":45}]`, wantErr: "multiple of 90"},
		{name: "bookmarks missing", raw: `[{"op":"set-bookmarks"}]`, wantErr: `set-bookmarks: "bookmarks" must list at least one bookmark`},
		{name: "bookmarks empty", raw: `[{"op":"set-bookmarks","bookmarks":[]}]`, wantErr: `"bookmarks" must list at least one bookmark`},
		{name: "bookmark without title", raw: `[{"op":"set-bookmarks","bookmarks":[{"title":"A","pageNumber":1},{"pageNumber":1}]}]`, wantErr: `set-bookmarks: bookmarks[1].title is required`},
		{name: "nested bookmark without title", raw: `[{"op":"set-bookmarks","bookmarks":[{"title":"A","pageNumber":1,"children":[{"title":" ","pageNumber":1}]}]}]`, wantErr: `set-bookmarks: bookmarks[0].children[0].title is required`},
		{name: "unknown permission", raw: `[{"op":"encrypt","userPassword":"u","permissions":{"printt":false}}]`, wantErr: `encrypt: "permissions" has an unknown key "printt"; the keys are print, printHighQuality, modifyContent, extractText, modifyAnnotations, fillForms and assemble`},
		{name: "permissions", raw: `[{"op":"encrypt","userPassword":"u","permissions":{"print":false,"assemble":true}}]`, wantOps: []string{"encrypt"}},
		{name: "unknown op lists the operations", raw: `[{"op":"explode"}]`, wantErr: `unknown operation "explode"; the operations are attach, autotag, convert-colors, decrypt, delete,`},
		{name: "missing pages", raw: `[{"op":"select"}]`, wantErr: `"pages" must not be empty`},
		{name: "split needs one mode", raw: `[{"op":"split"}]`, wantErr: `set "every" or "ranges"`},
		{name: "split modes exclusive", raw: `[{"op":"split","every":2,"ranges":["1"]}]`, wantErr: "not both"},
		{name: "bad color", raw: `[{"op":"stamp-text","text":"t","color":"red"}]`, wantErr: "#rrggbb"},
		{name: "bad font", raw: `[{"op":"stamp-text","text":"t","font":"Arial"}]`, wantErr: "standard PDF fonts"},
		{name: "bad opacity", raw: `[{"op":"stamp-text","text":"t","opacity":2}]`, wantErr: "between 0 and 1"},
		{name: "bad image", raw: `[{"op":"stamp-image","image":"logo.gif"}]`, wantErr: ".png"},
		{name: "bad profile", raw: `[{"op":"remediate","profile":"PDF/X-4"}]`, wantErr: `"profile" must be one of`},
		{name: "factur-x level needs xml", raw: `[{"op":"remediate","profile":"Factur-X","level":"BASIC"}]`, wantErr: `"level" and "documentType" need "xml"`},
		{name: "factur-x options on pdf/a", raw: `[{"op":"remediate","profile":"PDF/A-2b","level":"BASIC"}]`, wantErr: "only apply to the Factur-X"},
		{name: "encrypt needs a password", raw: `[{"op":"encrypt"}]`, wantErr: "userPassword"},
		{name: "bad algorithm", raw: `[{"op":"encrypt","userPassword":"u","algorithm":"des"}]`, wantErr: "aes256"},
		{name: "split range not a list", raw: `[{"op":"split","ranges":"1-3,4"}]`, wantErr: `split: "ranges" is invalid (got "1-3,4", want a JSON array of strings)`},
		{name: "empty split range", raw: `[{"op":"split","ranges":["1-3",""]}]`, wantErr: `"ranges" entry 1 must select pages`},
		{name: "timestamp url scheme", raw: `[{"op":"timestamp","timestampUrl":"file:///etc/passwd"}]`, wantErr: "http:// or https://"},
		{name: "decrypt not first", raw: `[{"op":"merge"},{"op":"decrypt","password":"x"}]`, wantErr: "must be the first stage"},
		{name: "edit after remediate", raw: `[{"op":"remediate","profile":"PDF/A-2b"},{"op":"rotate","angle":90}]`, wantErr: `cannot follow "remediate"`},
		{name: "decrypt after encrypt", raw: `[{"op":"encrypt","userPassword":"u"},{"op":"decrypt","password":"x"}]`, wantErr: `stage 1: "decrypt" must be the first stage`},
		{name: "two remediations", raw: `[{"op":"remediate","profile":"PDF/UA-1"},{"op":"remediate","profile":"PDF/UA-1"}]`, wantErr: `stage 1: only one "remediate" stage is allowed`},
		{name: "two encryptions", raw: `[{"op":"encrypt","userPassword":"u"},{"op":"encrypt","userPassword":"v"}]`, wantErr: `stage 1: only one "encrypt" stage is allowed`},
		{name: "two signatures", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"sign","certificate":"d.p12"}]`, wantErr: `stage 1: only one "sign" stage is allowed`},
		{name: "two timestamps", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"timestamp","timestampUrl":"https://tsa.example.com"},{"op":"timestamp","timestampUrl":"https://tsa.example.com"}]`, wantErr: `stage 2: only one "timestamp" stage is allowed`},
		{name: "edit after encrypt", raw: `[{"op":"encrypt","userPassword":"u"},{"op":"rotate","angle":90}]`, wantErr: `stage 1: "rotate" cannot follow "encrypt": only optimize, render and extract-attachments may`},
		{name: "remediate after encrypt", raw: `[{"op":"encrypt","userPassword":"u"},{"op":"remediate","profile":"PDF/UA-1"}]`, wantErr: `stage 1: "remediate" cannot follow "encrypt": only optimize, render and extract-attachments may`},
		{name: "edit after sign", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"rotate","angle":90}]`, wantErr: `stage 1: "rotate" cannot follow "sign": only sign, timestamp, render and extract-attachments may`},
		{name: "edit after timestamp", raw: `[{"op":"timestamp","timestampUrl":"https://tsa.example.com"},{"op":"optimize"}]`, wantErr: `stage 1: "optimize" cannot follow "timestamp": only sign, timestamp, render and extract-attachments may`},
		{name: "encrypt after sign", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"encrypt","userPassword":"u"}]`, wantErr: `"encrypt" cannot be combined with "sign" or "timestamp"`},
		{name: "decrypt then sign", raw: `[{"op":"decrypt","password":"x"},{"op":"sign","certificate":"c.p12"}]`, wantErr: `"decrypt" cannot be combined with "sign" or "timestamp": a PDF that was encrypted cannot be signed in the request that decrypts it; decrypt it in a request of its own, then sign the result`},
		{name: "decrypt then timestamp", raw: `[{"op":"decrypt","password":"x"},{"op":"rotate","angle":90},{"op":"timestamp","timestampUrl":"https://tsa.example.com"}]`, wantErr: `"decrypt" cannot be combined with "sign" or "timestamp"`},
		{name: "encrypt before timestamp", raw: `[{"op":"encrypt","userPassword":"u"},{"op":"timestamp","timestampUrl":"https://tsa.example.com"}]`, wantErr: `"encrypt" cannot be combined with "sign" or "timestamp"`},
		{name: "encrypted pdf/a", raw: `[{"op":"remediate","profile":"PDF/A-2b"},{"op":"encrypt","userPassword":"u"}]`, wantErr: `"PDF/A-2b" cannot be encrypted: PDF/A forbids encryption`},
		{name: "encrypted factur-x", raw: `[{"op":"remediate","profile":"Factur-X","xml":"i.xml","level":"BASIC"},{"op":"encrypt","userPassword":"u"}]`, wantErr: `"Factur-X" cannot be encrypted: Factur-X is PDF/A-3, which forbids encryption`},
		{name: "render not last", raw: `[{"op":"render"},{"op":"rotate","angle":90}]`, wantErr: `"render" must be the last stage`},
		{name: "extract-attachments not last", raw: `[{"op":"extract-attachments"},{"op":"rotate","angle":90}]`, wantErr: `stage 0: "extract-attachments" must be the last stage`},
		{name: "extract-attachments before render", raw: `[{"op":"extract-attachments"},{"op":"render"}]`, wantErr: `stage 0: "extract-attachments" must be the last stage`},
		{name: "extract-attachments after sign", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"extract-attachments"}]`, wantOps: []string{"sign", "extract-attachments"}},
		{name: "extract-attachments after remediate and encrypt", raw: `[{"op":"remediate","profile":"PDF/UA-1"},{"op":"encrypt","userPassword":"u"},{"op":"extract-attachments","name":"factur-x.xml"}]`, wantOps: []string{"remediate", "encrypt", "extract-attachments"}},
		{name: "bad render format", raw: `[{"op":"render","format":"webp"}]`, wantErr: "png or jpeg"},
		{name: "quality on png", raw: `[{"op":"render","quality":80}]`, wantErr: "only applies to the jpeg"},
		{name: "bad render quality", raw: `[{"op":"render","format":"jpeg","quality":101}]`, wantErr: "between 1 and 100"},
		{name: "negative dpi", raw: `[{"op":"render","dpi":-1}]`, wantErr: `"dpi" must be at least 1, got -1`},
		{name: "zero dpi", raw: `[{"op":"render","dpi":0}]`, wantErr: `"dpi" must be at least 1, got 0`},
		{name: "zero jpeg quality", raw: `[{"op":"render","format":"jpeg","quality":0}]`, wantErr: `"quality" must be between 1 and 100, got 0`},
		{name: "zero png quality", raw: `[{"op":"render","quality":0}]`, wantErr: `"quality" only applies to the jpeg format`},
		{name: "jpeg quality", raw: `[{"op":"render","format":"jpeg","quality":1}]`, wantOps: []string{"render"}},
		{name: "rotate without angle", raw: `[{"op":"rotate"}]`, wantErr: `rotate: "angle" is required, a multiple of 90 such as 90, 180 or 270`},
		{name: "rotate by zero", raw: `[{"op":"rotate","angle":0}]`, wantOps: []string{"rotate"}},
		{name: "zero font size", raw: `[{"op":"stamp-text","text":"t","fontSize":0}]`, wantErr: `stamp-text: "fontSize" must be greater than 0, got 0`},
		{name: "negative font size", raw: `[{"op":"stamp-text","text":"t","fontSize":-4}]`, wantErr: `stamp-text: "fontSize" must be greater than 0, got -4`},
		{name: "negative scale", raw: `[{"op":"stamp-pdf","source":"a.pdf","scale":-1}]`, wantErr: `stamp-pdf: "scale" must be greater than 0, got -1`},
		{name: "zero scale", raw: `[{"op":"stamp-pdf","source":"a.pdf","scale":0}]`, wantErr: `stamp-pdf: "scale" must be greater than 0, got 0`},
		{name: "split every zero", raw: `[{"op":"split","every":0}]`, wantErr: `split: "every" must be at least 1, got 0`},
		{name: "split every negative", raw: `[{"op":"split","every":-1}]`, wantErr: `split: "every" must be at least 1, got -1`},
		{name: "creation date in ISO form", raw: `[{"op":"set-metadata","creationDate":"2026-10-01"}]`, wantErr: `set-metadata: "creationDate" must be a PDF date, such as "D:20261001090000Z"`},
		{name: "mod date month 13", raw: `[{"op":"set-metadata","modDate":"D:20261301"}]`, wantErr: `set-metadata: "modDate" must be a PDF date`},
		{name: "PDF dates", raw: `[{"op":"set-metadata","creationDate":"D:20261001090000Z","modDate":"D:20261001090000+02'00'"}]`, wantOps: []string{"set-metadata"}},
		{name: "PDF date to the day", raw: `[{"op":"set-metadata","creationDate":"D:20261001"}]`, wantOps: []string{"set-metadata"}},
		{name: "render after sign", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"render"}]`, wantOps: []string{"sign", "render"}},
		{name: "render after remediate and encrypt", raw: `[{"op":"remediate","profile":"PDF/UA-1"},{"op":"encrypt","userPassword":"u"},{"op":"render"}]`, wantOps: []string{"remediate", "encrypt", "render"}},
		{name: "sign then timestamp", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"timestamp","timestampUrl":"https://tsa.example.com"}]`, wantOps: []string{"sign", "timestamp"}},
		{name: "optimize after encrypt", raw: `[{"op":"encrypt","userPassword":"u"},{"op":"optimize"}]`, wantOps: []string{"encrypt", "optimize"}},
		{name: "encrypted pdf/ua", raw: `[{"op":"remediate","profile":"PDF/UA-1"},{"op":"encrypt","userPassword":"u"}]`, wantOps: []string{"remediate", "encrypt"}},
		{name: "pdf/a and pdf/ua pair", raw: `[{"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"]}]`, wantOps: []string{"remediate"}},
		{name: "pair in either order", raw: `[{"op":"remediate","profile":["PDF/UA-1","PDF/A-1a"]}]`, wantOps: []string{"remediate"}},
		{name: "pdf/a-4 and pdf/ua-2 pair", raw: `[{"op":"remediate","profile":["PDF/A-4f","PDF/UA-2"]}]`, wantOps: []string{"remediate"}},
		{name: "factur-x and pdf/ua-1 pair", raw: `[{"op":"remediate","profile":["Factur-X","PDF/UA-1"],"xml":"i.xml","level":"BASIC"}]`, wantOps: []string{"remediate"}},
		{name: "factur-x repair and pdf/ua-1 pair", raw: `[{"op":"remediate","profile":["Factur-X","PDF/UA-1"]}]`, wantOps: []string{"remediate"}},
		{name: "one profile in an array", raw: `[{"op":"remediate","profile":["PDF/A-2b"]}]`, wantOps: []string{"remediate"}},
		{name: "pdf/a-2b and pdf/ua-2", raw: `[{"op":"remediate","profile":["PDF/A-2b","PDF/UA-2"]}]`, wantErr: `remediate: "profile" pairs a PDF/A profile with the PDF/UA profile of its PDF version: PDF/A-1a, PDF/A-1b, PDF/A-2a, PDF/A-2b, PDF/A-2u, PDF/A-3a, PDF/A-3b, PDF/A-3u or Factur-X with PDF/UA-1, and PDF/A-4, PDF/A-4e or PDF/A-4f with PDF/UA-2; got ["PDF/A-2b","PDF/UA-2"]`},
		{name: "pdf/a-4 and pdf/ua-1", raw: `[{"op":"remediate","profile":["PDF/A-4","PDF/UA-1"]}]`, wantErr: `got ["PDF/A-4","PDF/UA-1"]`},
		{name: "factur-x and pdf/ua-2", raw: `[{"op":"remediate","profile":["Factur-X","PDF/UA-2"]}]`, wantErr: `got ["Factur-X","PDF/UA-2"]`},
		{name: "two pdf/a profiles", raw: `[{"op":"remediate","profile":["PDF/A-2b","PDF/A-3b"]}]`, wantErr: `"profile" pairs a PDF/A profile with the PDF/UA profile of its PDF version`},
		{name: "two pdf/ua profiles", raw: `[{"op":"remediate","profile":["PDF/UA-1","PDF/UA-2"]}]`, wantErr: `"profile" pairs a PDF/A profile with the PDF/UA profile of its PDF version`},
		{name: "the same profile twice", raw: `[{"op":"remediate","profile":["PDF/UA-1","PDF/UA-1"]}]`, wantErr: `"profile" pairs a PDF/A profile with the PDF/UA profile of its PDF version`},
		{name: "three profiles", raw: `[{"op":"remediate","profile":["PDF/A-2b","PDF/UA-1","PDF/A-3b"]}]`, wantErr: `"profile" must be one of PDF/A-1a, PDF/A-1b, PDF/A-2a, PDF/A-2b, PDF/A-2u, PDF/A-3a, PDF/A-3b, PDF/A-3u, PDF/A-4, PDF/A-4e, PDF/A-4f, PDF/UA-1, PDF/UA-2 or Factur-X, or a pair of a PDF/A profile and a PDF/UA one`},
		{name: "no profile", raw: `[{"op":"remediate","profile":[]}]`, wantErr: `"profile" must be one of`},
		{name: "profile missing", raw: `[{"op":"remediate"}]`, wantErr: `PDF/A-4e or PDF/A-4f with PDF/UA-2; got ""`},
		{name: "profile as a number", raw: `[{"op":"remediate","profile":2}]`, wantErr: `remediate: "profile" must be a string, or a JSON array of two strings that pair a PDF/A profile with a PDF/UA one`},
		{name: "factur-x options on a pdf/a pair", raw: `[{"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"],"level":"BASIC"}]`, wantErr: "only apply to the Factur-X"},
		{name: "encrypted pdf/a pair", raw: `[{"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"]},{"op":"encrypt","userPassword":"u"}]`, wantErr: `["PDF/A-2b","PDF/UA-1"] cannot be encrypted: PDF/A forbids encryption`},
		{name: "encrypted factur-x pair", raw: `[{"op":"remediate","profile":["PDF/UA-1","Factur-X"]},{"op":"encrypt","userPassword":"u"}]`, wantErr: `["PDF/UA-1","Factur-X"] cannot be encrypted: Factur-X is PDF/A-3, which forbids encryption`},
		{name: "a remediation per profile of a pair", raw: `[{"op":"remediate","profile":"PDF/A-2b"},{"op":"remediate","profile":"PDF/UA-1"}]`, wantErr: `stage 1: only one "remediate" stage is allowed; to remediate to a PDF/A profile and a PDF/UA one together, give one stage both, such as {"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"]}`},
		{name: "a remediation per profile of a pdf/a-4 pair", raw: `[{"op":"remediate","profile":"PDF/UA-2"},{"op":"remediate","profile":"PDF/A-4e"}]`, wantErr: `such as {"op":"remediate","profile":["PDF/A-4e","PDF/UA-2"]}`},
		{name: "a remediation per profile of no pair", raw: `[{"op":"remediate","profile":"PDF/A-4"},{"op":"remediate","profile":"PDF/UA-1"}]`, wantErr: `such as {"op":"remediate","profile":["PDF/A-2b","PDF/UA-1"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stages, err := ParseStages(tc.raw)

			if tc.wantErr != "" {
				if !errors.Is(err, ErrInvalidStages) {
					t.Fatalf("err = %v, want it to wrap ErrInvalidStages", err)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
				}
				// The client reads the message: no Go type or field.
				if strings.Contains(err.Error(), "Go ") || strings.Contains(err.Error(), "Stage.") {
					t.Fatalf("err = %v, names a Go type", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("err = %v, want none", err)
			}

			var ops []string
			if stages != nil {
				ops = []string{}
				for _, s := range stages {
					ops = append(ops, s.op())
				}
			}

			if !slices.Equal(ops, tc.wantOps) {
				t.Fatalf("ops = %v, want %v", ops, tc.wantOps)
			}
		})
	}
}

func TestStagesFiles(t *testing.T) {
	stages, err := ParseStages(`[
		{"op":"stamp-image","image":"logo.png"},
		{"op":"stamp-pdf","source":"cover.pdf"},
		{"op":"attach","file":"logo.png"},
		{"op":"insert","source":"annex.pdf","at":1},
		{"op":"sign","certificate":"cert.pem","key":"key.pem"}
	]`)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"logo.png", "cover.pdf", "annex.pdf", "cert.pem", "key.pem"}
	if got := stages.Files(); !slices.Equal(got, want) {
		t.Fatalf("Files() = %v, want %v", got, want)
	}
}

func TestRunEnd(t *testing.T) {
	stages, err := ParseStages(`[
		{"op":"rotate","angle":90},
		{"op":"delete","pages":"1"},
		{"op":"stamp-text","text":"t"},
		{"op":"merge"},
		{"op":"set-metadata","title":"T"}
	]`)
	if err != nil {
		t.Fatal(err)
	}

	// The delete changes the page count, so it closes the first run; the
	// merge stands alone.
	for _, tc := range []struct{ start, want int }{{0, 2}, {2, 3}, {4, 5}} {
		if got := runEnd(stages, tc.start); got != tc.want {
			t.Errorf("runEnd(%d) = %d, want %d", tc.start, got, tc.want)
		}
	}
}

func TestSplitGroups(t *testing.T) {
	groups, err := splitGroups(&splitStage{Every: new(2)}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 3 || !slices.Equal(groups[2], []int{5}) {
		t.Fatalf("every 2 of 5 = %v", groups)
	}

	groups, err = splitGroups(&splitStage{Ranges: []string{"1-2", " 4", "last", "1,3,5"}}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 4 || !slices.Equal(groups[0], []int{1, 2}) || !slices.Equal(groups[2], []int{5}) || !slices.Equal(groups[3], []int{1, 3, 5}) {
		t.Fatalf("ranges = %v", groups)
	}

	if _, err := splitGroups(&splitStage{Ranges: []string{"1", "9"}}, 5); err == nil {
		t.Fatal("a range past the last page must fail")
	}
}

func TestParsePages(t *testing.T) {
	for _, tc := range []struct {
		spec    string
		want    []int
		wantErr bool
	}{
		{spec: "all", want: []int{1, 2, 3}},
		{spec: " ALL ", want: []int{1, 2, 3}},
		{spec: "2-3", want: []int{2, 3}},
		{spec: "last", want: []int{3}},
		{spec: "all,1", wantErr: true},
		{spec: "4", wantErr: true},
	} {
		t.Run(tc.spec, func(t *testing.T) {
			got, err := parsePages(tc.spec, 3)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, want error %t", err, tc.wantErr)
			}

			if !tc.wantErr && !slices.Equal(got, tc.want) {
				t.Fatalf("pages = %v, want %v", got, tc.want)
			}
		})
	}
}
