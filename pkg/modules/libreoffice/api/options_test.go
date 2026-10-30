package api

import (
	"fmt"
	"strings"
	"testing"
)

func TestOptions_String_MasksPassword(t *testing.T) {
	options := DefaultOptions()
	options.Password = "s3cr3t"

	for _, format := range []string{"%v", "%+v", "%s"} {
		got := fmt.Sprintf(format, options)
		if strings.Contains(got, "s3cr3t") {
			t.Errorf("%s leaks the password: %s", format, got)
		}
		if !strings.Contains(got, "Password:***") && format == "%+v" {
			t.Errorf("%s should show a masked password, got: %s", format, got)
		}
	}

	if options.Password != "s3cr3t" {
		t.Errorf("String must not modify the options, got password %q", options.Password)
	}
}
