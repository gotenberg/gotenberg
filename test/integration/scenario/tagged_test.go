package scenario

import (
	"strings"
	"testing"
)

func TestCheckTagged(t *testing.T) {
	t.Parallel()

	pdfinfo := func(tagged string) string {
		return "Producer:        LibreOffice 26.8\nTagged:          " + tagged + "\nPages:           1\n"
	}

	tests := []struct {
		name    string
		output  string
		want    bool
		wantErr string
	}{
		{
			name:   "tagged PDF expected and found",
			output: pdfinfo("yes"),
			want:   true,
		},
		{
			name:   "untagged PDF expected and found",
			output: pdfinfo("no"),
			want:   false,
		},
		{
			name:    "tagged PDF expected but untagged",
			output:  pdfinfo("no"),
			want:    true,
			wantErr: "expected tagged PDF to be true, but actual is false",
		},
		{
			name:    "untagged PDF expected but tagged",
			output:  pdfinfo("yes"),
			want:    false,
			wantErr: "expected tagged PDF to be false, but actual is true",
		},
		{
			name:    "no Tagged line",
			output:  "Pages:           1\n",
			want:    true,
			wantErr: "invalid pdfinfo output, no Tagged line:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkTagged(tc.output, tc.want)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}
