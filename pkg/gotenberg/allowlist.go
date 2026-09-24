package gotenberg

import (
	"strings"
)

// AllowListRisk classifies why an allow-list pattern is dangerous. A URL that
// matches an allow-list skips the private and public IP checks, so a pattern
// that matches more than its author intended silently widens outbound access.
// See [AuditAllowList].
type AllowListRisk string

const (
	// AllowListRiskUnanchored marks a pattern with no leading "^". regexp2
	// searches rather than matches, so the pattern hits anywhere in the URL,
	// including the query string.
	AllowListRiskUnanchored AllowListRisk = "unanchored"

	// AllowListRiskUnanchoredBranch marks an alternation whose later branches
	// have no leading "^". Anchoring only the first branch is a common slip.
	AllowListRiskUnanchoredBranch AllowListRisk = "unanchored-branch"

	// AllowListRiskCatchAll marks a pattern with no literal prefix, such as
	// ".+", which matches every URL and disables filtering entirely.
	AllowListRiskCatchAll AllowListRisk = "catch-all"

	// AllowListRiskOpenHost marks a pattern whose host is not terminated, so
	// it also matches attacker-chosen suffix hosts. For example
	// "^https://trusted\.example\.com" matches
	// "https://trusted.example.com.attacker.example/".
	AllowListRiskOpenHost AllowListRisk = "open-host"
)

// AllowListFinding reports one risky entry of an allow-list.
type AllowListFinding struct {
	// Index is the zero-based position of the pattern within the flag value.
	Index int

	// Pattern is the operator's pattern, verbatim.
	Pattern string

	// Risk is why the pattern is dangerous.
	Risk AllowListRisk
}

// maxAuditedPatternLength bounds the patterns [AuditAllowList] inspects. A
// pathological pattern is not worth walking, and reporting nothing is better
// than reporting a partial verdict.
const maxAuditedPatternLength = 4096

// AuditAllowList reports the entries of an allow-list that match more URLs
// than their author is likely to intend. It is a lint over the pattern source,
// not a parser: it recognizes the shapes that are dangerous in practice and
// stays silent when it cannot be sure.
//
// Callers use the findings to warn operators. Never use them to reject a
// configuration: existing deployments rely on loose patterns, and a pattern
// this function does not flag is not thereby safe.
func AuditAllowList(patterns []string) []AllowListFinding {
	var findings []AllowListFinding

	for i, pattern := range patterns {
		if pattern == "" || len(pattern) > maxAuditedPatternLength {
			continue
		}

		risk, ok := auditPattern(pattern)
		if ok {
			findings = append(findings, AllowListFinding{Index: i, Pattern: pattern, Risk: risk})
		}
	}

	return findings
}

// auditPattern classifies a single pattern, reporting the first risk found.
func auditPattern(pattern string) (AllowListRisk, bool) {
	body := trimInlineFlags(pattern)

	branches := splitTopLevelAlternation(body)
	for i, branch := range branches {
		branch = strings.TrimSpace(branch)

		anchored := hasStartAnchor(branch)
		rest := strings.TrimPrefix(strings.TrimPrefix(branch, `\A`), "^")

		// Catch-all first: a pattern that constrains nothing matches every URL
		// whether or not it is anchored, and saying so is more useful than
		// telling the operator to anchor it.
		if literalPrefix(rest) == "" {
			return AllowListRiskCatchAll, true
		}

		if !anchored {
			if i == 0 {
				return AllowListRiskUnanchored, true
			}
			return AllowListRiskUnanchoredBranch, true
		}

		// A lookaround invalidates the token walk, so skip the host check for
		// this branch rather than guess. The anchor and catch-all checks above
		// still applied.
		if containsLookaround(rest) {
			continue
		}

		if hostIsOpen(rest) {
			return AllowListRiskOpenHost, true
		}
	}

	return "", false
}

// trimInlineFlags removes a leading inline flag group such as "(?i)" so that
// the anchor check sees the pattern proper.
func trimInlineFlags(pattern string) string {
	if !strings.HasPrefix(pattern, "(?") {
		return pattern
	}

	end := strings.Index(pattern, ")")
	if end == -1 {
		return pattern
	}

	// Only a flag group qualifies. "(?:", "(?=", "(?!" and "(?<" open a real
	// group and must stay.
	flags := pattern[2:end]
	if flags == "" || strings.ContainsAny(flags, ":=!<") {
		return pattern
	}
	for _, r := range flags {
		if !strings.ContainsRune("imsUx-", r) {
			return pattern
		}
	}

	return pattern[end+1:]
}

// hasStartAnchor reports whether branch begins with a start-of-input anchor.
func hasStartAnchor(branch string) bool {
	return strings.HasPrefix(branch, "^") || strings.HasPrefix(branch, `\A`)
}

// containsLookaround reports whether the pattern uses a lookaround, which the
// token walk in [hostIsOpen] cannot reason about.
func containsLookaround(s string) bool {
	return strings.Contains(s, "(?=") || strings.Contains(s, "(?!") || strings.Contains(s, "(?<")
}

// splitTopLevelAlternation splits on "|" at paren depth zero, honoring escapes
// and character classes.
func splitTopLevelAlternation(s string) []string {
	var (
		parts   []string
		current strings.Builder
		depth   int
		inClass bool
	)

	for i := 0; i < len(s); i++ {
		c := s[i]

		switch {
		case c == '\\' && i+1 < len(s):
			current.WriteByte(c)
			current.WriteByte(s[i+1])
			i++
			continue
		case c == '[' && !inClass:
			inClass = true
		case c == ']' && inClass:
			inClass = false
		case c == '(' && !inClass:
			depth++
		case c == ')' && !inClass:
			depth--
		case c == '|' && !inClass && depth == 0:
			parts = append(parts, current.String())
			current.Reset()
			continue
		}

		current.WriteByte(c)
	}

	parts = append(parts, current.String())

	return parts
}

// literalPrefix returns the characters a matching URL must start with. It
// stops at the first optional or non-literal token, and descends one level
// into a leading mandatory group so that "^(https|http)://" is not mistaken
// for a catch-all. An empty result means the pattern constrains nothing.
func literalPrefix(s string) string {
	var prefix strings.Builder

	for i := 0; i < len(s); {
		// A group: descend once when it is mandatory, otherwise stop.
		if s[i] == '(' {
			end := matchingParen(s, i)
			if end == -1 {
				break
			}
			if isQuantified(s, end+1) {
				break
			}

			inner := trimInlineFlags(s[i+1 : end])
			branches := splitTopLevelAlternation(inner)

			common := literalPrefix(branches[0])
			for _, b := range branches[1:] {
				common = commonPrefix(common, literalPrefix(b))
			}
			prefix.WriteString(common)

			// Only the leading group is worth descending into.
			break
		}

		var token string
		switch {
		case s[i] == '\\' && i+1 < len(s):
			token = s[i : i+2]
		case s[i] == '[':
			end := matchingBracket(s, i)
			if end == -1 {
				return prefix.String()
			}
			token = s[i : end+1]
		default:
			token = s[i : i+1]
		}

		next := i + len(token)
		if isQuantified(s, next) {
			break
		}

		// Only a plain literal or an escaped literal contributes.
		switch {
		case len(token) == 2 && token[0] == '\\' && !isEscapeClass(token[1]):
			prefix.WriteByte(token[1])
		case len(token) == 1 && !strings.ContainsAny(token, `.[]()^$*+?{}|`):
			prefix.WriteByte(token[0])
		default:
			return prefix.String()
		}

		i = next
	}

	return prefix.String()
}

// hostIsOpen reports whether the authority part of the pattern can be left
// without crossing a terminator, which means the pattern also matches
// attacker-chosen suffix hosts or userinfo.
//
// It walks the tokens after "://" and classifies each one. A terminator ends
// the authority, so the pattern is safe. A crosser can match "@", "?" or "#"
// and therefore lets a matching URL escape the authority, so the pattern is
// open. Reaching the end without a terminator is open too, which is the
// classic "^https://trusted\.example\.com" case.
func hostIsOpen(s string) bool {
	_, after, ok := strings.Cut(s, "://")
	if !ok {
		// No authority to reason about, for example "^file:///tmp/".
		return false
	}

	rest := after

	for i := 0; i < len(rest); {
		var token string
		switch {
		case rest[i] == '\\' && i+1 < len(rest):
			token = rest[i : i+2]
		case rest[i] == '[':
			end := matchingBracket(rest, i)
			if end == -1 {
				return true
			}
			token = rest[i : end+1]
		case rest[i] == '(':
			end := matchingParen(rest, i)
			if end == -1 {
				return true
			}
			token = rest[i : end+1]
		default:
			token = rest[i : i+1]
		}

		next := i + len(token)
		optional := isOptionalQuantifier(rest, next)

		switch classifyHostToken(token) {
		case hostTokenTerminator:
			// An optional terminator does not end anything, since the URL may
			// match without it.
			if !optional {
				return false
			}
		case hostTokenCrosser:
			return true
		case hostTokenNeutral:
			// Part of the host itself, so keep walking.
		}

		i = next
		for i < len(rest) && isQuantifierByte(rest[i]) {
			if rest[i] == '{' {
				end := strings.IndexByte(rest[i:], '}')
				if end == -1 {
					return true
				}
				i += end + 1
				continue
			}
			i++
		}
	}

	return true
}

// hostTokenKind is how a token affects the walk in [hostIsOpen].
type hostTokenKind int

const (
	hostTokenNeutral hostTokenKind = iota
	hostTokenTerminator
	hostTokenCrosser
)

// hostTerminators are the characters that end the authority of a URL.
const hostTerminators = "/:#?"

// crosserClassChars are the characters that, if a class can match them, let a
// match escape the authority. "/" is deliberately absent: it ends the
// authority rather than escaping it, so a class such as "[:/]" is safe.
const crosserClassChars = "@?#"

// classifyHostToken classifies one token of the authority walk.
func classifyHostToken(token string) hostTokenKind {
	switch {
	case token == ".":
		// The wildcard matches "@", "#" and "?", so a host built on it can be
		// left without ever reaching a terminator.
		return hostTokenCrosser

	case token == "$":
		return hostTokenTerminator

	case len(token) == 1 && strings.Contains(hostTerminators, token):
		return hostTokenTerminator

	case len(token) == 2 && token[0] == '\\':
		switch token[1] {
		case 'S', 'D', 'W':
			return hostTokenCrosser
		case 'd', 'w', 's':
			return hostTokenNeutral
		case 'p', 'P':
			return hostTokenCrosser
		}
		if strings.Contains(hostTerminators, token[1:]) {
			return hostTokenTerminator
		}
		return hostTokenNeutral

	case strings.HasPrefix(token, "["):
		inner := strings.TrimSuffix(strings.TrimPrefix(token, "["), "]")
		if strings.HasPrefix(inner, "^") {
			// A negated class almost always admits "@".
			return hostTokenCrosser
		}
		if classContainsAny(inner, crosserClassChars) {
			return hostTokenCrosser
		}
		if classOnlyTerminators(inner) {
			return hostTokenTerminator
		}
		return hostTokenNeutral

	case strings.HasPrefix(token, "("):
		return classifyGroup(token)
	}

	return hostTokenNeutral
}

// classifyGroup classifies a parenthesized group. A group whose every branch
// starts with a terminator ends the authority, which is what makes the
// idiomatic "(:|/|$)" safe. A group containing a crosser is a crosser.
func classifyGroup(token string) hostTokenKind {
	inner := trimInlineFlags(strings.TrimSuffix(strings.TrimPrefix(token, "("), ")"))
	inner = strings.TrimPrefix(inner, "?:")

	branches := splitTopLevelAlternation(inner)

	allTerminate := true
	for _, branch := range branches {
		if branch == "" {
			allTerminate = false
			continue
		}

		kind := classifyHostToken(firstToken(branch))
		if kind == hostTokenCrosser {
			return hostTokenCrosser
		}
		if kind != hostTokenTerminator {
			allTerminate = false
		}

		// A crosser anywhere inside the branch still escapes the authority.
		if branchHasCrosser(branch) {
			return hostTokenCrosser
		}
	}

	if allTerminate {
		return hostTokenTerminator
	}

	return hostTokenNeutral
}

// branchHasCrosser reports whether any token of branch is a crosser.
func branchHasCrosser(branch string) bool {
	for i := 0; i < len(branch); {
		token := tokenAt(branch, i)
		if token == "" {
			return true
		}
		if classifyHostToken(token) == hostTokenCrosser {
			return true
		}
		i += len(token)
	}

	return false
}

// firstToken returns the first regex token of s.
func firstToken(s string) string {
	return tokenAt(s, 0)
}

// tokenAt returns the regex token starting at index i, or "" if it is
// malformed.
func tokenAt(s string, i int) string {
	if i >= len(s) {
		return ""
	}

	switch {
	case s[i] == '\\' && i+1 < len(s):
		return s[i : i+2]
	case s[i] == '[':
		end := matchingBracket(s, i)
		if end == -1 {
			return ""
		}
		return s[i : end+1]
	case s[i] == '(':
		end := matchingParen(s, i)
		if end == -1 {
			return ""
		}
		return s[i : end+1]
	}

	return s[i : i+1]
}

// classOnlyTerminators reports whether every character a class can match ends
// the authority, which makes the class itself a terminator. A range is never
// treated as one.
func classOnlyTerminators(class string) bool {
	if class == "" {
		return false
	}

	for i := 0; i < len(class); i++ {
		if class[i] == '\\' && i+1 < len(class) {
			if !strings.Contains(hostTerminators, class[i+1:i+2]) {
				return false
			}
			i++
			continue
		}

		if i+2 < len(class) && class[i+1] == '-' {
			return false
		}

		if !strings.Contains(hostTerminators, class[i:i+1]) {
			return false
		}
	}

	return true
}

// classContainsAny reports whether a character class body can match any of the
// given characters, expanding simple ranges.
func classContainsAny(class, chars string) bool {
	for i := 0; i < len(class); i++ {
		if class[i] == '\\' && i+1 < len(class) {
			// An escape class such as \S inside a class admits everything.
			if strings.ContainsRune("SDW", rune(class[i+1])) {
				return true
			}
			if strings.ContainsRune(chars, rune(class[i+1])) {
				return true
			}
			i++
			continue
		}

		if i+2 < len(class) && class[i+1] == '-' {
			lo, hi := class[i], class[i+2]
			for _, c := range []byte(chars) {
				if c >= lo && c <= hi {
					return true
				}
			}
			i += 2
			continue
		}

		if strings.ContainsRune(chars, rune(class[i])) {
			return true
		}
	}

	return false
}

// matchingParen returns the index of the ")" closing the "(" at start.
func matchingParen(s string, start int) int {
	depth := 0
	inClass := false

	for i := start; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			i++
		case s[i] == '[' && !inClass:
			inClass = true
		case s[i] == ']' && inClass:
			inClass = false
		case s[i] == '(' && !inClass:
			depth++
		case s[i] == ')' && !inClass:
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// matchingBracket returns the index of the "]" closing the "[" at start.
func matchingBracket(s string, start int) int {
	for i := start + 1; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			i++
		case s[i] == ']':
			// A "]" immediately after "[" or "[^" is a literal.
			if i == start+1 || (i == start+2 && s[start+1] == '^') {
				continue
			}
			return i
		}
	}

	return -1
}

// isQuantifierByte reports whether c opens a quantifier.
func isQuantifierByte(c byte) bool {
	return c == '?' || c == '*' || c == '+' || c == '{'
}

// isQuantified reports whether a quantifier starts at index i.
func isQuantified(s string, i int) bool {
	return i < len(s) && isQuantifierByte(s[i])
}

// isOptionalQuantifier reports whether the quantifier at index i lets the
// preceding token match nothing.
func isOptionalQuantifier(s string, i int) bool {
	if i >= len(s) {
		return false
	}

	switch s[i] {
	case '?', '*':
		return true
	case '{':
		return strings.HasPrefix(s[i:], "{0")
	}

	return false
}

// isEscapeClass reports whether c after a backslash denotes a character class
// rather than a literal.
func isEscapeClass(c byte) bool {
	return strings.ContainsRune("dDwWsSbBAzZpP", rune(c))
}

// commonPrefix returns the longest common prefix of a and b.
func commonPrefix(a, b string) string {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return a[:i]
		}
	}

	return a[:n]
}
