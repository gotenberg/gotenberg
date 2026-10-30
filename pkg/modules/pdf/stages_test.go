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
		{name: "not an array", raw: `{"op":"merge"}`, wantErr: "JSON array"},
		{name: "missing op", raw: `[{"angle":90}]`, wantErr: `"op" must name an operation`},
		{name: "unknown op", raw: `[{"op":"shred"}]`, wantErr: `unknown operation "shred"`},
		{name: "unknown option", raw: `[{"op":"rotate","angle":90,"degrees":90}]`, wantErr: `rotate: unknown option "degrees"`},
		{name: "option names are exact", raw: `[{"op":"set-metadata","Title":"T"}]`, wantErr: `set-metadata: unknown option "Title"`},
		{name: "wrong option type", raw: `[{"op":"rotate","angle":"ninety"}]`, wantErr: "rotate"},
		{name: "invalid angle", raw: `[{"op":"rotate","angle":45}]`, wantErr: "multiple of 90"},
		{name: "missing pages", raw: `[{"op":"select"}]`, wantErr: `"pages" must not be empty`},
		{name: "split needs one mode", raw: `[{"op":"split"}]`, wantErr: `set "every" or "ranges"`},
		{name: "split modes exclusive", raw: `[{"op":"split","every":2,"ranges":"1"}]`, wantErr: "not both"},
		{name: "bad color", raw: `[{"op":"stamp-text","text":"t","color":"red"}]`, wantErr: "#rrggbb"},
		{name: "bad font", raw: `[{"op":"stamp-text","text":"t","font":"Arial"}]`, wantErr: "standard PDF fonts"},
		{name: "bad opacity", raw: `[{"op":"stamp-text","text":"t","opacity":2}]`, wantErr: "between 0 and 1"},
		{name: "bad image", raw: `[{"op":"stamp-image","image":"logo.gif"}]`, wantErr: ".png"},
		{name: "bad profile", raw: `[{"op":"remediate","profile":"PDF/X-4"}]`, wantErr: `"profile" must be one of`},
		{name: "factur-x level needs xml", raw: `[{"op":"remediate","profile":"Factur-X","level":"BASIC"}]`, wantErr: `"level" and "documentType" need "xml"`},
		{name: "factur-x options on pdf/a", raw: `[{"op":"remediate","profile":"PDF/A-2b","level":"BASIC"}]`, wantErr: "only apply to the Factur-X"},
		{name: "encrypt needs a password", raw: `[{"op":"encrypt"}]`, wantErr: "userPassword"},
		{name: "bad algorithm", raw: `[{"op":"encrypt","userPassword":"u","algorithm":"des"}]`, wantErr: "aes256"},
		{name: "timestamp url scheme", raw: `[{"op":"timestamp","url":"file:///etc/passwd"}]`, wantErr: "http:// or https://"},
		{name: "decrypt not first", raw: `[{"op":"merge"},{"op":"decrypt","password":"x"}]`, wantErr: "must be the first stage"},
		{name: "edit after remediate", raw: `[{"op":"remediate","profile":"PDF/A-2b"},{"op":"rotate","angle":90}]`, wantErr: `cannot follow "remediate"`},
		{name: "two remediations", raw: `[{"op":"remediate","profile":"PDF/UA-1"},{"op":"remediate","profile":"PDF/UA-1"}]`, wantErr: `cannot follow "remediate"`},
		{name: "edit after encrypt", raw: `[{"op":"encrypt","userPassword":"u"},{"op":"rotate","angle":90}]`, wantErr: `cannot follow "encrypt"`},
		{name: "edit after sign", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"rotate","angle":90}]`, wantErr: `cannot follow "sign"`},
		{name: "encrypt and sign", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"encrypt","userPassword":"u"}]`, wantErr: `cannot follow "sign"`},
		{name: "encrypted pdf/a", raw: `[{"op":"remediate","profile":"PDF/A-2b"},{"op":"encrypt","userPassword":"u"}]`, wantErr: "PDF/A forbids encryption"},
		{name: "encrypted factur-x", raw: `[{"op":"remediate","profile":"Factur-X","xml":"i.xml","level":"BASIC"},{"op":"encrypt","userPassword":"u"}]`, wantErr: "PDF/A forbids encryption"},
		{name: "sign then timestamp", raw: `[{"op":"sign","certificate":"c.p12"},{"op":"timestamp","url":"https://tsa.example.com"}]`, wantOps: []string{"sign", "timestamp"}},
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
		{"op":"stamp-pdf","pdf":"cover.pdf"},
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
	groups, err := splitGroups(&splitStage{Every: 2}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 3 || !slices.Equal(groups[2], []int{5}) {
		t.Fatalf("every 2 of 5 = %v", groups)
	}

	groups, err = splitGroups(&splitStage{Ranges: "1-2, 4,last"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 3 || !slices.Equal(groups[0], []int{1, 2}) || !slices.Equal(groups[2], []int{5}) {
		t.Fatalf("ranges = %v", groups)
	}

	if _, err := splitGroups(&splitStage{Ranges: "1,9"}, 5); err == nil {
		t.Fatal("a range past the last page must fail")
	}
}
