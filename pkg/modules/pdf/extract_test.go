package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	litho "github.com/gotenberg/litho/pkg/pdf"

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

// TestExtractAttachmentsStreamBudget pins that an attachment under a single
// FlateDecode filter, which litho streams, decodes no further than
// --pdf-max-stream-bytes: past it, the stage fails as the client's mistake and
// leaves no partial attachment on disk.
func TestExtractAttachmentsStreamBudget(t *testing.T) {
	const budget = 1 << 20

	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	if _, err := w.Write(make([]byte, 3*budget)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, _ := testContext(t)
	path := filepath.Join(ctx.DirPath(), "bomb.pdf")
	err := os.WriteFile(path, attachmentPDF(z.Bytes()), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	processor := testProcessor()
	processor.limits = litho.Limits{MaxStreamBytes: budget}

	_, err = processor.Run(ctx, mustStages(t, `[{"op":"extract-attachments"}]`), []string{path}, nil)

	status, message := api.ParseError(err)
	want := `Stage 'extract-attachments' failed on 'bomb.pdf': attachment "a.bin": FlateDecode output exceeds 1048576 bytes: decoded stream exceeds the stream size limit`
	if status != http.StatusBadRequest || message != want {
		t.Fatalf("got %d %q, want 400 %q", status, message, want)
	}

	entries, err := os.ReadDir(ctx.DirPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("%d files in the request directory, want only bomb.pdf", len(entries))
	}
}

// attachmentPDF returns a one-page PDF embedding a.bin, whose stream is data
// under a single FlateDecode filter.
func attachmentPDF(data []byte) []byte {
	var b bytes.Buffer
	var offsets []int
	object := func(body string, stream []byte) {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\n", len(offsets), body)
		if stream != nil {
			b.WriteString("stream\n")
			b.Write(stream)
			b.WriteString("\nendstream\n")
		}
		b.WriteString("endobj\n")
	}

	b.WriteString("%PDF-1.7\n")
	object("<< /Type /Catalog /Pages 2 0 R /Names << /EmbeddedFiles << /Names [(a.bin) 4 0 R] >> >> >>", nil)
	object("<< /Type /Pages /Kids [3 0 R] /Count 1 >>", nil)
	object("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>", nil)
	object("<< /Type /Filespec /F (a.bin) /EF << /F 5 0 R >> >>", nil)
	object(fmt.Sprintf("<< /Type /EmbeddedFile /Filter /FlateDecode /Length %d >>", len(data)), data)

	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)

	return b.Bytes()
}
