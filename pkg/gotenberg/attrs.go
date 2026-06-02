package gotenberg

import (
	"net/url"
)

// maxAttrRunes bounds the length of a string span attribute to keep payload
// size and backend cardinality in check.
const maxAttrRunes = 256

// CapAttr truncates s to at most [maxAttrRunes] runes, appending an ellipsis
// when it shortens the value. It is multibyte-safe.
func CapAttr(s string) string {
	runes := []rune(s)
	if len(runes) <= maxAttrRunes {
		return s
	}
	return string(runes[:maxAttrRunes-1]) + "…"
}

// RedactURL parses raw and returns a redacted, length-capped form safe to use
// as a span attribute or event value. Userinfo, query, and fragment are
// dropped because they may carry credentials or other sensitive data. It
// returns an empty string when raw is empty or cannot be parsed.
func RedactURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""

	return CapAttr(parsed.String())
}

// MapEnum returns value when it belongs to allowed, otherwise "other". It keeps
// a span attribute or metric dimension bounded even when an upstream tool
// introduces a new enum value.
func MapEnum(value string, allowed ...string) string {
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return "other"
}
