package pdf

import (
	"net/http"
	"os"
	"testing"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

func TestExtractText(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stages   string
		fixtures []string
		want     map[string]string
		wantErr  string
	}{
		{
			name:     "every page, joined by a form feed",
			stages:   `[{"op":"extract-text"}]`,
			fixtures: []string{"pages_3.pdf"},
			want:     map[string]string{"pages_3.txt": "Page 1\fPage 2\fPage 3"},
		},
		{
			name:     "selected pages, in the order the selection gives",
			stages:   `[{"op":"extract-text","pages":"3,1"}]`,
			fixtures: []string{"pages_3.pdf"},
			want:     map[string]string{"pages_3.txt": "Page 3\fPage 1"},
		},
		{
			name:     "annotations left out",
			stages:   `[{"op":"extract-text"}]`,
			fixtures: []string{"page_1.pdf"},
			want:     map[string]string{"page_1.txt": ""},
		},
		{
			name:     "annotations",
			stages:   `[{"op":"extract-text","annotations":true}]`,
			fixtures: []string{"page_1.pdf", "page_2.pdf"},
			want:     map[string]string{"page_1.txt": "Page 1", "page_2.txt": "Page 2"},
		},
		{
			name:     "a scanned page reads as empty",
			stages:   `[{"op":"extract-text"}]`,
			fixtures: []string{"scanned.pdf", "pages_3.pdf"},
			want:     map[string]string{"scanned.txt": "", "pages_3.txt": "Page 1\fPage 2\fPage 3"},
		},
		{
			name:     "after an encryption",
			stages:   `[{"op":"encrypt","userPassword":"u"},{"op":"extract-text","pages":"2"}]`,
			fixtures: []string{"pages_3.pdf"},
			want:     map[string]string{"pages_3.txt": "Page 2"},
		},
		{
			name:     "a page past the end",
			stages:   `[{"op":"extract-text","pages":"4"}]`,
			fixtures: []string{"pages_3.pdf"},
			wantErr:  "Stage 'extract-text' failed on 'pages_3.pdf'",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, paths := testContext(t, tc.fixtures...)

			inputs := make([]string, len(tc.fixtures))
			for i, f := range tc.fixtures {
				inputs[i] = paths[f]
			}

			outputs, err := testProcessor().Run(ctx, mustStages(t, tc.stages), inputs, nil)
			if tc.wantErr != "" {
				status, message := api.ParseError(err)
				if status != http.StatusBadRequest || len(message) < len(tc.wantErr) || message[:len(tc.wantErr)] != tc.wantErr {
					t.Fatalf("got %d %q, want 400 starting with %q", status, message, tc.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("err = %v, want none", err)
			}

			extracted := outputs[len(inputs):]
			if len(extracted) != len(tc.want) {
				t.Fatalf("%d text files, want %d", len(extracted), len(tc.want))
			}

			for _, path := range extracted {
				name := ctx.OriginalFilename(path)

				want, ok := tc.want[name]
				if !ok {
					t.Fatalf("unexpected text file %q", name)
				}

				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}

				if string(got) != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
		})
	}
}

func TestExtractTextStageOptions(t *testing.T) {
	for _, tc := range []struct {
		stage extractTextStage
		want  int
	}{
		{stage: extractTextStage{}, want: 0},
		{stage: extractTextStage{Invisible: true}, want: 1},
		{stage: extractTextStage{Invisible: true, Artifacts: true, Annotations: true}, want: 3},
	} {
		if got := len(tc.stage.options()); got != tc.want {
			t.Errorf("%+v gives %d options, want %d", tc.stage, got, tc.want)
		}
	}
}
