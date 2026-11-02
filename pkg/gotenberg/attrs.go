package gotenberg

import (
	"net/url"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
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

// OutboundURLAttrs returns the server.address and url.full attributes of a
// client span that targets raw: the host alone, and the URL redacted with
// [RedactURL]. It returns nil when raw cannot be parsed or has no host.
func OutboundURLAttrs(raw string) []attribute.KeyValue {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return nil
	}

	return []attribute.KeyValue{
		semconv.ServerAddress(parsed.Hostname()),
		semconv.URLFull(RedactURL(raw)),
	}
}

// MapEnum returns value when it belongs to allowed, otherwise "other". It keeps
// a span attribute or metric dimension bounded even when an upstream tool
// introduces a new enum value.
func MapEnum(value string, allowed ...string) string {
	if slices.Contains(allowed, value) {
		return value
	}
	return "other"
}
