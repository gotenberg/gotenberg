package chromium

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// hostResolverRules is a parsed --chromium-host-resolver-rules value. The
// pinning proxy resolves every destination Chromium sends it, so Chromium
// never applies these rules itself; the outbound checks apply them instead,
// with the semantics of Chromium's net::HostMappingRules, so that a request
// connects where Chromium would have sent it.
// See https://source.chromium.org/chromium/chromium/src/+/main:net/base/host_mapping_rules.cc.
type hostResolverRules struct {
	maps     []hostMapRule
	excludes []string
}

type hostMapRule struct {
	pattern string
	host    string
	// port is empty when the rule keeps the request's port.
	port string
}

// parseHostResolverRules parses comma-separated "MAP <pattern> <host[:port]>"
// and "EXCLUDE <pattern>" rules. Chromium logs and skips a rule it cannot
// parse; this returns an error instead, so that a mistyped rule does not
// silently send requests elsewhere. Empty entries, such as the one a trailing
// comma leaves, are skipped.
func parseHostResolverRules(value string) (hostResolverRules, error) {
	var rules hostResolverRules

	for rule := range strings.SplitSeq(value, ",") {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		parts := strings.Split(rule, " ")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		switch {
		case len(parts) == 2 && strings.EqualFold(parts[0], "exclude"):
			rules.excludes = append(rules.excludes, strings.ToLower(parts[1]))
		case len(parts) == 3 && strings.EqualFold(parts[0], "map"):
			host, port, err := parseReplacement(parts[2])
			if err != nil {
				return hostResolverRules{}, fmt.Errorf("rule '%s': %w", rule, err)
			}
			rules.maps = append(rules.maps, hostMapRule{pattern: strings.ToLower(parts[1]), host: host, port: port})
		default:
			return hostResolverRules{}, fmt.Errorf("rule '%s' is neither 'MAP <pattern> <host[:port]>' nor 'EXCLUDE <pattern>'", rule)
		}
	}

	return rules, nil
}

// parseReplacement splits the replacement of a MAP rule into its host and
// optional port, like Chromium's net::ParseHostAndPort: the port separator is
// the last colon after any closing bracket, and a bracketed host must be an
// IPv6 literal, returned without its brackets.
func parseReplacement(value string) (string, string, error) {
	if strings.Contains(value, "@") {
		return "", "", fmt.Errorf("replacement '%s' must not hold credentials", value)
	}

	host, port := value, ""
	hasPort := false

	bracketEnd := strings.LastIndex(value, "]")
	if !strings.HasPrefix(value, "[") || bracketEnd != -1 {
		colon := strings.LastIndex(value, ":")
		if colon > bracketEnd {
			host, port, hasPort = value[:colon], value[colon+1:], true
		}
	}

	if host == "" {
		return "", "", fmt.Errorf("replacement '%s' has no host", value)
	}

	if hasPort {
		number, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return "", "", fmt.Errorf("replacement '%s' has an invalid port", value)
		}
		port = strconv.FormatUint(number, 10)
	}

	if after, ok := strings.CutPrefix(host, "["); ok {
		inner := strings.TrimSuffix(after, "]")
		addr, err := netip.ParseAddr(inner)
		if !strings.HasSuffix(host, "]") || err != nil || !addr.Is6() {
			return "", "", fmt.Errorf("replacement '%s' has an invalid IPv6 literal", value)
		}
		host = inner
	}

	return host, port, nil
}

// errHostNotFound reports a host that a rule maps to Chromium's ^NOTFOUND or
// ~NOTFOUND, which Chromium answers with ERR_NAME_NOT_RESOLVED.
var errHostNotFound = errors.New("mapped to NOTFOUND")

// rewrite returns the host and port Chromium connects to for host and port.
// The first MAP rule whose pattern matches the host, or the host and port,
// applies, unless an EXCLUDE pattern matches the host.
func (r hostResolverRules) rewrite(host, port string) (string, string, error) {
	hostPort := net.JoinHostPort(host, port)

	for _, rule := range r.maps {
		if !matchHostPattern(host, rule.pattern) && !matchHostPattern(hostPort, rule.pattern) {
			continue
		}

		for _, exclude := range r.excludes {
			if matchHostPattern(host, exclude) {
				return host, port, nil
			}
		}

		if rule.host == "^NOTFOUND" || rule.host == "~NOTFOUND" {
			return "", "", fmt.Errorf("'%s': %w", host, errHostNotFound)
		}

		if rule.port != "" {
			port = rule.port
		}

		return rule.host, port, nil
	}

	return host, port, nil
}

// matchHostPattern reports whether value matches pattern with the semantics of
// Chromium's base::MatchPattern: "*" matches any run of characters, "?" matches
// zero or one character, and "\" escapes either. Each literal run after a
// wildcard matches at its first possible position, without backtracking.
func matchHostPattern(value, pattern string) bool {
	str, pat := []rune(value), []rune(pattern)

	for {
		maxDistance := eatWildcards(&pat)
		if !searchForChars(&pat, &str, maxDistance) {
			return false
		}
		if len(pat) == 0 {
			return true
		}
	}
}

// eatWildcards consumes the wildcards at the start of pattern and returns how
// many characters they may skip, -1 meaning any number.
func eatWildcards(pattern *[]rune) int {
	questionMarks, asterisk := 0, false

	for len(*pattern) > 0 {
		switch (*pattern)[0] {
		case '?':
			questionMarks++
		case '*':
			asterisk = true
		default:
			if asterisk {
				return -1
			}
			return questionMarks
		}
		*pattern = (*pattern)[1:]
	}

	if asterisk {
		return -1
	}
	return questionMarks
}

// searchForChars matches the literal run at the start of pattern against str,
// trying up to maxDistance later starting positions in str, any number when
// negative. It stops at the next unescaped wildcard. At the end of the
// pattern, only the end of str matches.
func searchForChars(pattern, str *[]rune, maxDistance int) bool {
	patternStart, strStart := *pattern, *str
	escape := false

	for {
		if len(*pattern) == 0 {
			if len(*str) == 0 {
				return true
			}
		} else {
			c := (*pattern)[0]
			if !escape && (c == '*' || c == '?') {
				return true
			}

			if !escape && c == '\\' {
				escape = true
				*pattern = (*pattern)[1:]
				continue
			}

			escape = false

			if len(*str) == 0 {
				return false
			}

			if c == (*str)[0] {
				*pattern = (*pattern)[1:]
				*str = (*str)[1:]
				continue
			}
		}

		if maxDistance == 0 || len(strStart) == 0 {
			return false
		}

		maxDistance--
		*pattern = patternStart
		strStart = strStart[1:]
		*str = strStart
	}
}
