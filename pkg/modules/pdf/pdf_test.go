package pdf

import "testing"

func TestPDF_Debug(t *testing.T) {
	got := new(PDF).Debug()

	version, ok := got["litho_version"].(string)
	if !ok || version == "" || version == "unknown" {
		t.Fatalf("Debug() = %v, want the litho module version", got)
	}
}
