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

func TestOptions_String_ShowsWatermarkRotation(t *testing.T) {
	options := DefaultOptions()

	got := fmt.Sprintf("%v", options)
	if !strings.Contains(got, "NativeWatermarkRotateAngle:<nil>") {
		t.Errorf("an unset rotation should read <nil>, got: %s", got)
	}

	options.NativeWatermarkRotateAngle = new(450)

	got = fmt.Sprintf("%v", options)
	if !strings.Contains(got, "NativeWatermarkRotateAngle:450") || strings.Contains(got, "0x") {
		t.Errorf("a set rotation should read its value, not its address, got: %s", got)
	}

	if options.NativeWatermarkRotateAngle == nil || *options.NativeWatermarkRotateAngle != 450 {
		t.Errorf("String must not modify the options, got rotation %v", options.NativeWatermarkRotateAngle)
	}
}
