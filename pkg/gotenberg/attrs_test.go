package gotenberg

import (
	"errors"
	"fmt"
	"net/url"
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

func TestRedactURLError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"no url error", errors.New("boom"), "boom"},
		{
			"url error",
			&url.Error{Op: "Post", URL: "https://user:pass@example.com/hook?token=secret#frag", Err: errors.New("connection refused")},
			`Post "https://example.com/hook": connection refused`,
		},
		{
			"url error under a wrapper that formats on demand",
			&wrappedForTest{err: &url.Error{Op: "Post", URL: "https://example.com/hook?token=secret", Err: errors.New("connection refused")}},
			`send request: Post "https://example.com/hook": connection refused`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactURLError(tc.err)
			if got != tc.err {
				t.Fatal("expected the same error back")
			}
			if got == nil {
				return
			}
			if got.Error() != tc.want {
				t.Errorf("RedactURLError() = %q, want %q", got.Error(), tc.want)
			}
		})
	}
}

// wrappedForTest formats its message from err each time, as a
// [fmt.Errorf] wrapper does not.
type wrappedForTest struct{ err error }

func (w *wrappedForTest) Error() string { return fmt.Sprintf("send request: %v", w.err) }
func (w *wrappedForTest) Unwrap() error { return w.err }

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
