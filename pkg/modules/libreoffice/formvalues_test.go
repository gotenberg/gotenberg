package libreoffice

import "testing"

func TestFormValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		parse   func(string) (int, error)
		value   string
		want    int
		wantErr bool
	}{
		{"default watermark color", parseColor, "#7ffe7f", 8388223, false},
		{"upper-case color", parseColor, "#FF0000", 16711680, false},
		{"color without hash", parseColor, "ff0000", 0, true},
		{"short color", parseColor, "#fff", 0, true},
		{"color name", parseColor, "red", 0, true},
		{"rotation in degrees", parseRotation, "45", 450, false},
		{"fractional rotation", parseRotation, "12.5", 125, false},
		{"negative rotation", parseRotation, "-90", -900, false},
		{"full turn", parseRotation, "405", 450, false},
		{"rotation not a number", parseRotation, "45deg", 0, true},
		{"initial view", initialViews.parse, "outline", 1, false},
		{"magnification", magnifications.parse, "zoom", 4, false},
		{"page layout", pageLayouts.parse, "twoColumnRight", 4, false},
		{"page layout number", pageLayouts.parse, "2", 0, true},
		{"case matters", initialViews.parse, "Outline", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parse(tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parse(%q) = %d, want an error", tc.value, got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("parse(%q) = %d, %v, want %d", tc.value, got, err, tc.want)
			}
		})
	}
}
