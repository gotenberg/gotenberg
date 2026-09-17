package gotenberg

import (
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2/v2"
)

// mustPattern compiles a pattern the way the production lists are built.
func mustPattern(t *testing.T, expr string) *regexp2.Regexp {
	t.Helper()

	re := regexp2.MustCompile(expr, regexp2.None)
	re.MatchTimeout = PatternMatchTimeout

	return re
}

func TestMatchPattern(t *testing.T) {
	// The abort [MatchPattern] absorbs cannot be staged here: it needs the
	// whole process to lose the CPU around one specific match, which no test
	// can schedule. What is testable is the other half of the contract, that
	// retrying never turns a genuine runaway into a pass.
	// See https://github.com/gotenberg/gotenberg/issues/1659.
	for _, tc := range []struct {
		scenario    string
		pattern     *regexp2.Regexp
		s           string
		expectMatch bool
		expectError bool
	}{
		{
			scenario:    "deny-list match",
			pattern:     mustPattern(t, `^file:(?!//\/tmp/).*`),
			s:           "file:///etc/passwd",
			expectMatch: true,
		},
		{
			scenario:    "no match against Gotenberg's own working directory",
			pattern:     mustPattern(t, `^file:(?!//\/tmp/).*`),
			s:           "file:///tmp/1a2b3c4d/5e6f7a8b/9c0d1e2f.html",
			expectMatch: false,
		},
		{
			scenario:    "no match",
			pattern:     mustPattern(t, `^https://example\.com/`),
			s:           "https://example.org/",
			expectMatch: false,
		},
		{
			scenario:    "catastrophic backtracking still aborts",
			pattern:     mustPattern(t, `^https://example\.com/(a+)+$`),
			s:           "https://example.com/" + strings.Repeat("a", 40) + "!",
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ok, err := MatchPattern(tc.pattern, tc.s)

			if tc.expectError && err == nil {
				t.Fatal("expected an error but got none")
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if ok != tc.expectMatch {
				t.Fatalf("expected match %t but got %t", tc.expectMatch, ok)
			}
		})
	}
}

func TestMatchPatternBoundsCatastrophicPattern(t *testing.T) {
	// Proving a runaway is genuine costs one ceiling per attempt, so the
	// worst case is patternMatchAttempts of them plus regexp2's clock period
	// on each, roughly a second. The bound that matters is the one this
	// replaced: before the ceiling existed, the same match was allowed to
	// burn a core for the caller's whole 30s budget.
	pattern := mustPattern(t, `^https://example\.com/(a+)+$`)
	s := "https://example.com/" + strings.Repeat("a", 40) + "!"

	start := time.Now()
	_, err := MatchPattern(pattern, s)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a catastrophic pattern")
	}

	// Generous headroom over the expected second keeps this stable on a
	// loaded CI box while still failing if the bound is gone.
	if elapsed > 5*time.Second {
		t.Fatalf("match took %s, want at most %d ceilings of %s", elapsed, patternMatchAttempts, PatternMatchTimeout)
	}
}
