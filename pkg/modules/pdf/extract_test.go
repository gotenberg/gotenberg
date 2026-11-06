package pdf

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

func TestExtractAttachments(t *testing.T) {
	processor := testProcessor()
	ctx, paths := testContext(t, "page_1.pdf", "page_2.pdf", "embed_1.xml", "embed_2.xml")
	files := map[string]string{"embed_1.xml": paths["embed_1.xml"], "embed_2.xml": paths["embed_2.xml"], "page_2.pdf": paths["page_2.pdf"]}

	attached, err := processor.Run(ctx, mustStages(t, `[{"op":"attach","file":"embed_1.xml"},{"op":"attach","file":"embed_2.xml"},{"op":"attach","file":"page_2.pdf"}]`), []string{paths["page_1.pdf"]}, files)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		stages  string
		inputs  []string
		want    map[string]string
		wantErr string
	}{
		{
			name:   "every attachment",
			stages: `[{"op":"extract-attachments"}]`,
			inputs: attached,
			want:   map[string]string{"page_1_embed_1.xml": "embed_1.xml", "page_1_embed_2.xml": "embed_2.xml", "page_1_page_2.pdf": "page_2.pdf"},
		},
		{
			name:   "one attachment",
			stages: `[{"op":"extract-attachments","name":"embed_2.xml"}]`,
			inputs: attached,
			want:   map[string]string{"page_1_embed_2.xml": "embed_2.xml"},
		},
		{
			name:   "after an encryption",
			stages: `[{"op":"encrypt","userPassword":"u"},{"op":"extract-attachments","name":"embed_1.xml"}]`,
			inputs: attached,
			want:   map[string]string{"page_1_embed_1.xml": "embed_1.xml"},
		},
		{
			name:    "a missing attachment",
			stages:  `[{"op":"extract-attachments","name":"nope.xml"}]`,
			inputs:  attached,
			wantErr: `Stage 'extract-attachments' failed on 'page_1.pdf': no attachment named "nope.xml"; the PDF's attachments are: embed_1.xml, embed_2.xml, page_2.pdf`,
		},
		{
			name:    "no attachment",
			stages:  `[{"op":"extract-attachments"}]`,
			inputs:  []string{paths["page_1.pdf"]},
			wantErr: `Stage 'extract-attachments' failed on 'page_1.pdf': the PDF has no attachments`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outputs, err := processor.Run(ctx, mustStages(t, tc.stages), tc.inputs, files)
			if tc.wantErr != "" {
				status, message := api.ParseError(err)
				if status != http.StatusBadRequest || message != tc.wantErr {
					t.Fatalf("got %d %q, want 400 %q", status, message, tc.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("err = %v, want none", err)
			}

			// The final document comes first, then the attachments.
			extracted := outputs[len(tc.inputs):]
			if len(extracted) != len(tc.want) {
				t.Fatalf("%d attachments, want %d", len(extracted), len(tc.want))
			}

			for _, path := range extracted {
				name := ctx.OriginalFilename(path)

				fixture, ok := tc.want[name]
				if !ok {
					t.Fatalf("unexpected attachment %q", name)
				}

				if filepath.Ext(path) != filepath.Ext(name) {
					t.Errorf("%s on disk as %s, want the same extension", name, filepath.Base(path))
				}

				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}

				want, err := os.ReadFile(filepath.Join(testdata, fixture))
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(got, want) {
					t.Errorf("%s differs from %s", name, fixture)
				}
			}
		})
	}
}

func TestAttachmentFilename(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "factur-x.xml", want: "invoice_factur-x.xml"},
		{name: "../../etc/passwd", want: "invoice_passwd"},
		{name: `C:\temp\data.xml`, want: "invoice_data.xml"},
		{name: "da\x00ta\x1f.xml", want: "invoice_data.xml"},
		{name: "", want: "invoice_embedded"},
		{name: "..", want: "invoice_embedded"},
		{name: "dir/", want: "invoice_embedded"},
	} {
		if got := attachmentFilename("invoice", tc.name); got != tc.want {
			t.Errorf("attachmentFilename(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestUniqueName(t *testing.T) {
	taken := make(map[string]bool)

	var got []string
	for _, name := range []string{"a_2.xml", "a.xml", "a.xml", "a.xml", "b"} {
		got = append(got, uniqueName(name, taken))
	}

	want := []string{"a_2.xml", "a.xml", "a_3.xml", "a_4.xml", "b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uniqueName = %q, want %q", got, want)
		}
	}
}
