package gotenberg

import (
	"testing"

	"github.com/dlclark/regexp2"
)

func TestAuditAllowList(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		pattern  string
		want     AllowListRisk
	}{
		// Safe: the host is terminated before anything can leave it.
		{"idiomatic terminator group", `^https?://internal\.svc(:|/|$)`, ""},
		{"trailing slash", `^https://trusted\.example\.com/`, ""},
		{"optional port then terminator", `^https://example\.com(:[0-9]+)?(/|$)`, ""},
		{"positive class cannot leave authority", `^https://[a-z0-9.-]+\.s3\.amazonaws\.com/`, ""},
		{"leading mandatory group", `^(https|http)://a\.example\.com/`, ""},
		{"optional subdomain group", `^https://(www\.)?example\.com/`, ""},
		{"port terminator", `^https://example\.com:8443/`, ""},
		{"end anchor", `^https://example\.com$`, ""},
		{"alternation both anchored and terminated", `^https://a\.example/|^https://b\.example/`, ""},
		{"no authority to check", `^file:///tmp/`, ""},
		{"digit class in host", `^https://node\d+\.example\.com/`, ""},
		{"class of only terminators", `^https://example\.com[:/]`, ""},
		{"feature file pattern, fixed", `^https?://host\.docker\.internal(:[0-9]+)?/`, ""},

		// Unanchored: regexp2 searches, so these match anywhere in the URL.
		{"no anchor", `trusted\.example\.com`, AllowListRiskUnanchored},
		{"no anchor with scheme", `https://trusted\.example\.com/`, AllowListRiskUnanchored},

		// Only the first branch anchored.
		{"second branch unanchored", `^http://a\.example/|http://b\.example/`, AllowListRiskUnanchoredBranch},

		// Catch-all: matches every URL.
		{"dot plus", `.+`, AllowListRiskCatchAll},
		{"dot star", `.*`, AllowListRiskCatchAll},
		{"anchored dot star", `^.*`, AllowListRiskCatchAll},
		{"anchored dot plus", `^.+`, AllowListRiskCatchAll},

		// Open host: the reported vulnerability class.
		{"advisory pattern", `^http://trusted\.example\.com`, AllowListRiskOpenHost},
		{"gotenberg.dev internet-facing recipe", `^https?://[^/]+\.internal\.example\.com`, AllowListRiskOpenHost},
		{"gotenberg.dev strict whitelist recipe", `^https://(api|cdn|images)\.internal\.example\.com`, AllowListRiskOpenHost},
		{"gotenberg.dev hooks recipe", `^https?://hooks\.internal\.example\.com`, AllowListRiskOpenHost},
		{"feature file pattern", `^https?://host.docker.internal.*`, AllowListRiskOpenHost},
		{"scheme only", `^https?://`, AllowListRiskOpenHost},
		{"wildcard subdomain", `^https://.+\.example\.com/`, AllowListRiskOpenHost},
		{"escaped dot is not a terminator", `^https?://example\.com\.`, AllowListRiskOpenHost},
		{"negated class in host", `^https://[^.]+\.example\.com/`, AllowListRiskOpenHost},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			findings := AuditAllowList([]string{tc.pattern})

			if tc.want == "" {
				if len(findings) != 0 {
					t.Fatalf("AuditAllowList(%q) = %+v, want no finding", tc.pattern, findings)
				}
				return
			}

			if len(findings) != 1 {
				t.Fatalf("AuditAllowList(%q) returned %d findings, want 1", tc.pattern, len(findings))
			}
			if findings[0].Risk != tc.want {
				t.Fatalf("AuditAllowList(%q) risk = %q, want %q", tc.pattern, findings[0].Risk, tc.want)
			}
		})
	}
}

// TestAuditAllowList_FlaggedPatternsAreActuallyExploitable proves the audit is
// not merely syntactic: every pattern it flags as open-host really does admit
// a host the operator did not intend.
func TestAuditAllowList_FlaggedPatternsAreActuallyExploitable(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		attack  string
	}{
		{`^http://trusted\.example\.com`, "http://trusted.example.com.attacker.example/"},
		{`^https?://[^/]+\.internal\.example\.com`, "http://a.internal.example.com.attacker.example/"},
		{`^https://(api|cdn|images)\.internal\.example\.com`, "https://api.internal.example.com.attacker.example/"},
		{`^https?://hooks\.internal\.example\.com`, "http://hooks.internal.example.com.attacker.example/"},
		{`^https?://host.docker.internal.*`, "http://host.docker.internal.attacker.example/"},
		{`^https://.+\.example\.com/`, "https://attacker.example/#x.example.com/"},
		{`^https?://example\.com\.`, "http://example.com.attacker.example/"},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			findings := AuditAllowList([]string{tc.pattern})
			if len(findings) == 0 {
				t.Fatalf("pattern %q was not flagged", tc.pattern)
			}

			ok, err := regexp2.MustCompile(tc.pattern, 0).MatchString(tc.attack)
			if err != nil {
				t.Fatalf("match %q: %v", tc.attack, err)
			}
			if !ok {
				t.Fatalf("pattern %q does not match %q, so the finding is a false positive", tc.pattern, tc.attack)
			}
		})
	}
}

// TestAuditAllowList_SafePatternsRejectTheAttacks is the converse: the shapes
// the audit stays silent about really do reject the same attacks.
func TestAuditAllowList_SafePatternsRejectTheAttacks(t *testing.T) {
	safe := []string{
		`^https?://internal\.svc(:|/|$)`,
		`^https://trusted\.example\.com/`,
		`^https://example\.com(:[0-9]+)?(/|$)`,
		`^https://[a-z0-9.-]+\.s3\.amazonaws\.com/`,
	}
	attacks := []string{
		"https://internal.svc.attacker.example/",
		"https://trusted.example.com.attacker.example/",
		"https://trusted.example.com@169.254.169.254/",
		"https://example.com.attacker.example/",
		"https://example.com@10.0.0.5/",
		"https://bucket.s3.amazonaws.com.attacker.example/",
		"https://bucket.s3.amazonaws.com@127.0.0.1/",
	}

	for _, pattern := range safe {
		t.Run(pattern, func(t *testing.T) {
			if findings := AuditAllowList([]string{pattern}); len(findings) != 0 {
				t.Fatalf("safe pattern %q was flagged as %q", pattern, findings[0].Risk)
			}

			re := regexp2.MustCompile(pattern, 0)
			for _, attack := range attacks {
				ok, err := re.MatchString(attack)
				if err != nil {
					t.Fatalf("match %q: %v", attack, err)
				}
				if ok {
					t.Fatalf("pattern %q matches attack %q but was not flagged", pattern, attack)
				}
			}
		})
	}
}

func TestAuditAllowList_SkipsEmptyAndOversized(t *testing.T) {
	oversized := make([]byte, maxAuditedPatternLength+1)
	for i := range oversized {
		oversized[i] = 'a'
	}

	findings := AuditAllowList([]string{"", string(oversized)})
	if len(findings) != 0 {
		t.Fatalf("AuditAllowList returned %+v, want no finding", findings)
	}
}

func TestAuditAllowList_ReportsIndex(t *testing.T) {
	findings := AuditAllowList([]string{
		`^https://ok\.example\.com/`,
		`^https://open\.example\.com`,
	})
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	if findings[0].Index != 1 {
		t.Fatalf("findings[0].Index = %d, want 1", findings[0].Index)
	}
}

// TestAuditAllowList_ShippedChromiumDenyListIsNotAudited guards the rule that
// deny-lists are never audited. The shipped Chromium deny-list uses a
// lookaround and has no authority, so auditing it would produce noise.
func TestAuditAllowList_LookaroundIsNotFlaggedForHost(t *testing.T) {
	findings := AuditAllowList([]string{`^file:(?!//\/tmp/).*`})
	if len(findings) != 0 {
		t.Fatalf("AuditAllowList returned %+v, want no finding", findings)
	}
}
