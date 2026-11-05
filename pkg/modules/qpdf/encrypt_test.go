package qpdf

import (
	"reflect"
	"strings"
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

func TestQpdfEncryptArgs(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		userPassword  string
		ownerPassword string
		perms         gotenberg.PdfPermissions
		expect        []string
	}{
		{
			scenario:      "user and owner passwords",
			userPassword:  "foo",
			ownerPassword: "bar",
			perms:         gotenberg.PdfPermissions{AllowPrinting: true, AllowCopying: true, AllowModifying: true, AllowAnnotating: true, AllowFillingForms: true, AllowAssembling: true},
			expect:        []string{"--encrypt", "--user-password=foo", "--owner-password=bar", "--bits=256", "--"},
		},
		{
			scenario:      "owner password only",
			ownerPassword: "bar",
			perms:         gotenberg.PdfPermissions{AllowPrinting: false, AllowCopying: true, AllowModifying: true, AllowAnnotating: true, AllowFillingForms: true, AllowAssembling: true},
			expect:        []string{"--encrypt", "--owner-password=bar", "--bits=256", "--print=none", "--extract=y", "--modify-other=y", "--annotate=y", "--form=y", "--assemble=y", "--"},
		},
		{
			// QPDF reads an argument that starts with "@" as a file of
			// further arguments, one per line.
			scenario:      "argument file syntax stays inside the option values",
			userPassword:  "\n256\n--\n--add-attachment\n/etc/passwd\n--\n--encrypt\nA\nB",
			ownerPassword: "@/proc/self/cmdline",
			perms:         gotenberg.PdfPermissions{AllowPrinting: true, AllowCopying: true, AllowModifying: true, AllowAnnotating: true, AllowFillingForms: true, AllowAssembling: true},
			expect:        []string{"--encrypt", "--user-password=\n256\n--\n--add-attachment\n/etc/passwd\n--\n--encrypt\nA\nB", "--owner-password=@/proc/self/cmdline", "--bits=256", "--"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			got := qpdfEncryptArgs(tc.userPassword, tc.ownerPassword, tc.perms)
			if !reflect.DeepEqual(got, tc.expect) {
				t.Errorf("expected %q but got %q", tc.expect, got)
			}

			for _, arg := range got {
				if strings.HasPrefix(arg, "@") {
					t.Errorf("argument %q starts with '@', QPDF would read it as an argument file", arg)
				}
			}
		})
	}
}
