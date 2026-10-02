package scenario

import (
	"fmt"
	"strings"
	"testing"
)

func TestCheckImageXObjectReuse(t *testing.T) {
	t.Parallel()

	const header = `page   num  type   width height color comp bpc  enc interp  object ID x-ppi y-ppi size ratio
--------------------------------------------------------------------------------------------
`
	imageRow := func(page, object, generation int) string {
		return fmt.Sprintf("%d %d image 32 32 rgb 3 8 image no %d %d 96 96 3104B 101%%\n", page, page-1, object, generation)
	}

	tests := []struct {
		name    string
		output  string
		pages   int
		wantErr string
	}{
		{
			name:   "one page baseline",
			output: header + imageRow(1, 7, 0),
			pages:  1,
		},
		{
			name:   "three pages share one image object",
			output: header + imageRow(1, 7, 0) + imageRow(2, 7, 0) + imageRow(3, 7, 0),
			pages:  3,
		},
		{
			name:    "regression embeds a different object per page",
			output:  header + imageRow(1, 7, 0) + imageRow(2, 8, 0) + imageRow(3, 9, 0),
			pages:   3,
			wantErr: "expected 1 unique image XObject(s) shared across 3 page(s), but actual is 3",
		},
		{
			name:    "object generation is part of its identity",
			output:  header + imageRow(1, 7, 0) + imageRow(2, 7, 1) + imageRow(3, 7, 0),
			pages:   3,
			wantErr: "expected 1 unique image XObject(s) shared across 3 page(s), but actual is 2",
		},
		{
			name:    "image missing from a page",
			output:  header + imageRow(1, 7, 0) + imageRow(3, 7, 0),
			pages:   3,
			wantErr: "expected 1 image appearance(s) on page 2, but actual is 0",
		},
		{
			name:    "no image cannot pass",
			output:  header,
			pages:   1,
			wantErr: "expected 1 image appearance(s) on page 1, but actual is 0",
		},
		{
			name:    "repeated image only on first page cannot pass",
			output:  header + imageRow(1, 7, 0) + imageRow(1, 7, 0) + imageRow(1, 7, 0),
			pages:   3,
			wantErr: "expected 1 image appearance(s) on page 1, but actual is 3",
		},
		{
			name:    "image on unexpected page",
			output:  header + imageRow(1, 7, 0) + imageRow(2, 7, 0),
			pages:   1,
			wantErr: "image on page 2 outside expected range 1-1",
		},
		{
			name:    "tool error cannot pass",
			output:  "Syntax Error: Couldn't find trailer dictionary",
			pages:   1,
			wantErr: "invalid pdfimages -list output:",
		},
		{
			name:    "malformed image row",
			output:  header + "1 0 image\n",
			pages:   1,
			wantErr: "invalid pdfimages -list row:",
		},
		{
			name:    "inline image has no reusable object ID",
			output:  header + "1 0 image 32 32 rgb 3 8 image no [inline] - 96 96 3104B 101%\n",
			pages:   1,
			wantErr: "parse image object ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkImageXObjectReuse(tt.output, 1, tt.pages)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
