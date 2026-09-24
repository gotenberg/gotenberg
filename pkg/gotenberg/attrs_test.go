package gotenberg

import (
	"strings"
	"testing"
)

func TestRedactURL(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", ""},
		{"strips userinfo query fragment", "https://user:pass@example.com/path?token=secret#frag", "https://example.com/path"},
		{"keeps host and path", "http://example.com/a/b", "http://example.com/a/b"},
		{"parse error", "http://example.com/%zz", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RedactURL(tc.raw); got != tc.want {
				t.Errorf("RedactURL(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestRedactURL_Caps(t *testing.T) {
	raw := "https://example.com/" + strings.Repeat("a", 400)
	got := RedactURL(raw)
	if n := len([]rune(got)); n != maxAttrRunes {
		t.Errorf("expected capped length %d, got %d", maxAttrRunes, n)
	}
}

func TestCapAttr(t *testing.T) {
	t.Run("short unchanged", func(t *testing.T) {
		if got := CapAttr("short"); got != "short" {
			t.Errorf("expected unchanged, got %q", got)
		}
	})

	t.Run("multibyte truncated to rune cap", func(t *testing.T) {
		got := CapAttr(strings.Repeat("é", 400))
		if n := len([]rune(got)); n != maxAttrRunes {
			t.Errorf("expected %d runes, got %d", maxAttrRunes, n)
		}
		if !strings.HasSuffix(got, "…") {
			t.Error("expected an ellipsis suffix on a truncated value")
		}
	})
}

func TestMapEnum(t *testing.T) {
	allowed := []string{"document", "stylesheet", "script"}

	if got := MapEnum("script", allowed...); got != "script" {
		t.Errorf("expected member passthrough, got %q", got)
	}
	if got := MapEnum("websocket", allowed...); got != "other" {
		t.Errorf("expected non-member to map to other, got %q", got)
	}
}
