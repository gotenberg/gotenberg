package scenario

import (
	"strings"
	"testing"

	"github.com/gotenberg/litho/pkg/models"
)

func TestCheckImageXObjectReuse(t *testing.T) {
	t.Parallel()

	page := func(objects ...int) []models.ImageMeta {
		images := make([]models.ImageMeta, len(objects))
		for i, obj := range objects {
			images[i] = models.ImageMeta{Filename: "Im0", ObjectNum: obj}
		}
		return images
	}

	tests := []struct {
		name    string
		perPage [][]models.ImageMeta
		images  int
		pages   int
		wantErr string
	}{
		{name: "one page baseline", perPage: [][]models.ImageMeta{page(7)}, images: 1, pages: 1},
		{name: "three pages share one image object", perPage: [][]models.ImageMeta{page(7), page(7), page(7)}, images: 1, pages: 3},
		{
			name:    "regression embeds a different object per page",
			perPage: [][]models.ImageMeta{page(7), page(8), page(9)},
			images:  1,
			pages:   3,
			wantErr: "expected 1 unique image XObject(s) shared across 3 page(s), but actual is 3",
		},
		{
			name:    "a page misses its image",
			perPage: [][]models.ImageMeta{page(7), page()},
			images:  1,
			pages:   2,
			wantErr: "expected 1 image appearance(s) on page 2, but actual is 0",
		},
		{
			name:    "an inline image cannot be shared",
			perPage: [][]models.ImageMeta{page(0)},
			images:  1,
			pages:   1,
			wantErr: "direct object",
		},
		{
			name:    "page count mismatch",
			perPage: [][]models.ImageMeta{page(7)},
			images:  1,
			pages:   2,
			wantErr: "expected 2 page(s), but actual is 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkImageXObjectReuse(tc.perPage, tc.images, tc.pages)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
