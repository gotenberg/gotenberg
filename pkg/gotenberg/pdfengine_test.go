package gotenberg

import "testing"

func TestFacturXDocumentFileNameFor(t *testing.T) {
	for _, tc := range []struct {
		level string
		want  string
	}{
		{FacturXConformanceMinimum, FacturXDocumentFileName},
		{FacturXConformanceBasicWL, FacturXDocumentFileName},
		{FacturXConformanceBasic, FacturXDocumentFileName},
		{FacturXConformanceEN16931, FacturXDocumentFileName},
		{FacturXConformanceExtended, FacturXDocumentFileName},
		{FacturXConformanceXRechnung, XRechnungDocumentFileName},
	} {
		t.Run(tc.level, func(t *testing.T) {
			if got := FacturXDocumentFileNameFor(tc.level); got != tc.want {
				t.Errorf("FacturXDocumentFileNameFor(%q) = %q, want %q", tc.level, got, tc.want)
			}
		})
	}
}
