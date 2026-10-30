package chromium

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dlclark/regexp2/v2"
)

func TestExtraHttpHeader_String_MasksValue(t *testing.T) {
	headers := []ExtraHttpHeader{
		{Name: "Authorization", Value: "Bearer s3cr3t"},
		{Name: "X-Scoped", Value: "s3cr3t", Scope: regexp2.MustCompile(`^https://example\.com`, regexp2.None)},
	}

	got := fmt.Sprintf("%+v", headers)
	if strings.Contains(got, "s3cr3t") {
		t.Fatalf("formatting leaks a header value: %s", got)
	}
	if !strings.Contains(got, "Authorization") || !strings.Contains(got, `^https://example\.com`) {
		t.Fatalf("formatting should keep names and scopes, got: %s", got)
	}
}
