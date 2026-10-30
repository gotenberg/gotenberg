package gotenberg

import (
	"strings"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
)

func TestParsedFlags_MustString(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.String("foo", "", "")

	err := fs.Parse([]string{"--foo=foo"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustString(tc.name)
		})
	}
}
func TestParsedFlags_MustStringSlice(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.StringSlice("foo", make([]string, 0), "")

	err := fs.Parse([]string{"--foo=foo", "--foo=bar", "--foo=baz"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustStringSlice(tc.name)
		})
	}
}
func TestParsedFlags_MustBool(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Bool("foo", false, "")

	err := fs.Parse([]string{"--foo=true"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustBool(tc.name)
		})
	}
}
func TestParsedFlags_MustInt64(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Int64("foo", 0, "")

	err := fs.Parse([]string{"--foo=1"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustInt64(tc.name)
		})
	}
}
func TestParsedFlags_MustInt(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Int("foo", 0, "")

	err := fs.Parse([]string{"--foo=1"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustInt(tc.name)
		})
	}
}
func TestParsedFlags_MustFloat64(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Float64("foo", 1.0, "")

	err := fs.Parse([]string{"--foo=2.0"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustFloat64(tc.name)
		})
	}
}
func TestParsedFlags_MustDuration(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.Duration("foo", time.Duration(1)*time.Second, "")

	err := fs.Parse([]string{"--foo=2m"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustDuration(tc.name)
		})
	}
}
func TestParsedFlags_MustHumanReadableBytes(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.String("foo", "1MB", "")
	fs.String("bar", "1MB", "")
	fs.String("qux", "", "")

	err := fs.Parse([]string{"--foo=1GB", "--bar=foo"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
		{
			scenario:    "success: empty value",
			name:        "qux",
			expectPanic: false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustHumanReadableBytes(tc.name)
		})
	}
}
func TestParsedFlags_MustRegexp(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.String("foo", "", "")
	fs.String("bar", "", "")

	err := fs.Parse([]string{"--foo=", "--bar=*"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
	}{
		{
			scenario:    "success",
			name:        "foo",
			expectPanic: false,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			parsedFlags.MustRegexp(tc.name)
		})
	}
}
func TestParsedFlags_MustRegexpSlice(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.StringSlice("foo", []string{}, "")

	err := fs.Parse([]string{"--foo=^file:.*", "--foo=^ftp://.*"})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario    string
		name        string
		expectPanic bool
		expectLen   int
	}{
		{
			scenario:    "success with multiple patterns",
			name:        "foo",
			expectPanic: false,
			expectLen:   2,
		},
		{
			scenario:    "non-existing flag",
			name:        "bar",
			expectPanic: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic but got none")
					}
				}()
			}

			if !tc.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("expected no panic but got: %v", r)
					}
				}()
			}

			result := parsedFlags.MustRegexpSlice(tc.name)

			if !tc.expectPanic && len(result) != tc.expectLen {
				t.Errorf("expected %d regexps but got %d", tc.expectLen, len(result))
			}
		})
	}

	// Test empty strings are skipped.
	fs2 := flag.NewFlagSet("tests2", flag.ContinueOnError)
	fs2.StringSlice("baz", []string{""}, "")

	err = fs2.Parse([]string{})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags2 := ParsedFlags{FlagSet: fs2}
	result := parsedFlags2.MustRegexpSlice("baz")
	if len(result) != 0 {
		t.Errorf("expected 0 regexps for empty strings but got %d", len(result))
	}
}
func TestParsedFlags_AllowListWarning(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.StringSlice("chromium-allow-list", []string{}, "")
	fs.Bool("chromium-deny-private-ips", false, "")
	fs.Bool("chromium-deny-public-ips", false, "")
	fs.StringSlice("standalone-allow-list", []string{}, "")

	parsedFlags := ParsedFlags{FlagSet: fs}

	for _, tc := range []struct {
		scenario string
		name     string
		finding  AllowListFinding
		contains []string
	}{
		{
			scenario: "open host names both IP-check flags and their env vars",
			name:     "chromium-allow-list",
			finding:  AllowListFinding{Index: 0, Pattern: `^https://trusted\.example\.com`, Risk: AllowListRiskOpenHost},
			contains: []string{
				"--chromium-allow-list (CHROMIUM_ALLOW_LIST)",
				"entry 1",
				`^https://trusted\.example\.com`,
				"does not terminate the host",
				"--chromium-deny-private-ips (CHROMIUM_DENY_PRIVATE_IPS)",
				"--chromium-deny-public-ips (CHROMIUM_DENY_PUBLIC_IPS)",
				"End the host with",
			},
		},
		{
			scenario: "catch-all tells the operator to restrict or unset",
			name:     "chromium-allow-list",
			finding:  AllowListFinding{Index: 2, Pattern: ".+", Risk: AllowListRiskCatchAll},
			contains: []string{"entry 3", "matches every URL", "Restrict the entry"},
		},
		{
			scenario: "unanchored explains the search semantics",
			name:     "chromium-allow-list",
			finding:  AllowListFinding{Index: 0, Pattern: `trusted\.example\.com`, Risk: AllowListRiskUnanchored},
			contains: []string{"is not anchored with ^", "Anchor every branch with ^"},
		},
		{
			scenario: "module without IP-check flags falls back to a generic sentence",
			name:     "standalone-allow-list",
			finding:  AllowListFinding{Index: 0, Pattern: `^https://a\.example\.com`, Risk: AllowListRiskOpenHost},
			contains: []string{"skips the private and public IP checks"},
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			msg := parsedFlags.allowListWarning(tc.name, tc.finding)

			for _, want := range tc.contains {
				if !strings.Contains(msg, want) {
					t.Fatalf("message %q does not contain %q", msg, want)
				}
			}
			if strings.Contains(msg, "—") {
				t.Fatalf("message must not contain an em dash: %q", msg)
			}
		})
	}
}

func TestParsedFlags_WarnRiskyAllowList_SkipsDenyLists(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.StringSlice("chromium-deny-list", []string{}, "")

	parsedFlags := ParsedFlags{FlagSet: fs}

	// A deny-list is never audited: it always applies and cannot be bypassed,
	// so a loose one is safe. This must also not panic on a nil logger.
	parsedFlags.warnRiskyAllowList("chromium-deny-list", []string{".+", `^file:(?!//\/tmp/).*`})
}

func TestParsedFlags_WarnRiskyAllowList_NilLoggerDoesNotPanic(t *testing.T) {
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.StringSlice("chromium-allow-list", []string{}, "")

	parsedFlags := ParsedFlags{FlagSet: fs}

	// Provision runs after the entry point initializes the logger, but tests
	// and embedders reach this path with no logger at all.
	parsedFlags.warnRiskyAllowList("chromium-allow-list", []string{".+"})
}

func TestEnvVarName(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{"chromium-allow-list", "CHROMIUM_ALLOW_LIST"},
		{"api-download-from-deny-private-ips", "API_DOWNLOAD_FROM_DENY_PRIVATE_IPS"},
		{"log-level", "LOG_LEVEL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EnvVarName(tc.name); got != tc.want {
				t.Fatalf("EnvVarName(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestParsedFlags_RegexpMatchTimeout(t *testing.T) {
	// [DecideOutbound] matches on these patterns directly instead of compiling
	// a private copy per call, so the bound has to come from here. regexp2's
	// own default is math.MaxInt64, which it treats as no
	// timeout at all, so a pattern built without this stamp runs unbounded
	// against a client-controlled string.
	fs := flag.NewFlagSet("tests", flag.ContinueOnError)
	fs.StringSlice("some-deny-list", []string{`^file:`, `^https?://`}, "")
	fs.String("some-pattern", `^file:`, "")

	err := fs.Parse(nil)
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	parsedFlags := ParsedFlags{FlagSet: fs}

	regexps := parsedFlags.MustRegexpSlice("some-deny-list")
	if len(regexps) != 2 {
		t.Fatalf("expected 2 patterns but got %d", len(regexps))
	}

	for _, re := range regexps {
		if re.MatchTimeout != PatternMatchTimeout {
			t.Fatalf("pattern '%s' has MatchTimeout %s, expected %s", re.String(), re.MatchTimeout, PatternMatchTimeout)
		}
	}

	if got := parsedFlags.MustRegexp("some-pattern").MatchTimeout; got != PatternMatchTimeout {
		t.Fatalf("expected MustRegexp MatchTimeout %s but got %s", PatternMatchTimeout, got)
	}
}
