package qpdf

import (
	"reflect"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestQpdfPermissionArgs(t *testing.T) {
	allAllowed := gotenberg.PdfPermissions{
		AllowPrinting:     true,
		AllowCopying:      true,
		AllowModifying:    true,
		AllowAnnotating:   true,
		AllowFillingForms: true,
		AllowAssembling:   true,
	}

	for _, tc := range []struct {
		scenario string
		perms    gotenberg.PdfPermissions
		expect   []string
	}{
		{
			scenario: "all allowed yields no flags",
			perms:    allAllowed,
			expect:   nil,
		},
		{
			scenario: "printing denied",
			perms: gotenberg.PdfPermissions{
				AllowPrinting:     false,
				AllowCopying:      true,
				AllowModifying:    true,
				AllowAnnotating:   true,
				AllowFillingForms: true,
				AllowAssembling:   true,
			},
			expect: []string{
				"--print=none",
				"--extract=y",
				"--modify-other=y",
				"--annotate=y",
				"--form=y",
				"--assemble=y",
			},
		},
		{
			scenario: "copying denied",
			perms: gotenberg.PdfPermissions{
				AllowPrinting:     true,
				AllowCopying:      false,
				AllowModifying:    true,
				AllowAnnotating:   true,
				AllowFillingForms: true,
				AllowAssembling:   true,
			},
			expect: []string{
				"--print=full",
				"--extract=n",
				"--modify-other=y",
				"--annotate=y",
				"--form=y",
				"--assemble=y",
			},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			got := qpdfPermissionArgs(tc.perms)
			if !reflect.DeepEqual(got, tc.expect) {
				t.Errorf("expected %v but got %v", tc.expect, got)
			}
		})
	}
}
