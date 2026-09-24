package chromium

import "testing"

func TestResolvePdfOptions(t *testing.T) {
	for _, tc := range []struct {
		scenario           string
		generateOutline    bool
		generateTaggedIn   bool
		generateTaggedWant bool
	}{
		{
			scenario:           "outline requested forces tagged PDF",
			generateOutline:    true,
			generateTaggedIn:   false,
			generateTaggedWant: true,
		},
		{
			scenario:           "outline requested keeps tagged PDF on",
			generateOutline:    true,
			generateTaggedIn:   true,
			generateTaggedWant: true,
		},
		{
			scenario:           "no outline leaves tagged PDF off",
			generateOutline:    false,
			generateTaggedIn:   false,
			generateTaggedWant: false,
		},
		{
			scenario:           "no outline keeps tagged PDF on",
			generateOutline:    false,
			generateTaggedIn:   true,
			generateTaggedWant: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			options := DefaultPdfOptions()
			options.GenerateDocumentOutline = tc.generateOutline
			options.GenerateTaggedPdf = tc.generateTaggedIn

			got := resolvePdfOptions(options)

			if got.GenerateTaggedPdf != tc.generateTaggedWant {
				t.Errorf("expected GenerateTaggedPdf=%t, got %t", tc.generateTaggedWant, got.GenerateTaggedPdf)
			}
			if got.GenerateDocumentOutline != tc.generateOutline {
				t.Errorf("expected GenerateDocumentOutline=%t, got %t", tc.generateOutline, got.GenerateDocumentOutline)
			}
		})
	}
}
