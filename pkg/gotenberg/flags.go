package gotenberg

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dlclark/regexp2/v2"
	"github.com/labstack/gommon/bytes"
	flag "github.com/spf13/pflag"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg/internal/log"
)

// ParsedFlags wraps a [flag.FlagSet] so that retrieving the typed values is
// easier.
type ParsedFlags struct {
	*flag.FlagSet
}

// MustString returns the string value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustString(name string) string {
	val, err := f.GetString(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustStringSlice returns the string slice value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustStringSlice(name string) []string {
	val, err := f.GetStringSlice(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustBool returns the boolean value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustBool(name string) bool {
	val, err := f.GetBool(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustInt64 returns the int64 value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustInt64(name string) int64 {
	val, err := f.GetInt64(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustInt returns the int value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustInt(name string) int {
	val, err := f.GetInt(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustDuration returns the time.Duration value of a flag given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustDuration(name string) time.Duration {
	val, err := f.GetDuration(name)
	if err != nil {
		panic(err)
	}

	return val
}

// MustHumanReadableBytes returns the human-readable bytes string of a flag
// given by name.
// It panics if an error occurs.
func (f *ParsedFlags) MustHumanReadableBytes(name string) int64 {
	val, err := f.GetString(name)
	if err != nil {
		panic(err)
	}

	if val == "" {
		return 0
	}

	b, err := bytes.Parse(val)
	if err != nil {
		panic(err)
	}

	return b
}

// PatternMatchTimeout bounds a single match against an operator-supplied
// allow-list or deny-list pattern.
//
// regexp2 backtracks, and the strings matched against these patterns are
// client-controlled: a request URL, a CONNECT host. A pattern that backtracks
// catastrophically would otherwise burn a core for as long as the caller's
// deadline allows, which is --api-timeout (env API_TIMEOUT), 30 seconds by
// default. The ceiling mirrors the one the Chromium module already applies to
// the per-request extraHttpHeaders scope pattern.
//
// [ParsedFlags.MustRegexp] and [ParsedFlags.MustRegexpSlice] stamp this onto
// every pattern they compile, which is how all four production lists are
// built. Patterns compiled any other way keep regexp2's default of
// math.MaxInt64, which it treats as no timeout at all, so a hand-built slice
// must set this itself before reaching [DecideOutbound].
//
// The ceiling is wall-clock. Match through [MatchPattern] rather than calling
// regexp2 directly: a match that never approaches the ceiling still aborts if
// the process loses the CPU at the wrong moment.
const PatternMatchTimeout = 250 * time.Millisecond

// MustRegexpSlice returns a slice of compiled regular expressions from a
// string-slice flag given by name. Empty strings are skipped.
// It panics if an error occurs.
//
// Every allow-list and deny-list in Gotenberg is read through this method, so
// it is also where allow-list patterns are audited. See [AuditAllowList].
//
// The returned expressions carry [PatternMatchTimeout] and are safe to match
// on concurrently: callers must not compile a private copy per match.
func (f *ParsedFlags) MustRegexpSlice(name string) []*regexp2.Regexp {
	vals := f.MustStringSlice(name)

	f.warnRiskyAllowList(name, vals)

	var regexps []*regexp2.Regexp
	for _, val := range vals {
		if val == "" {
			continue
		}

		re := regexp2.MustCompile(val, regexp2.None)
		re.MatchTimeout = PatternMatchTimeout

		regexps = append(regexps, re)
	}

	return regexps
}

// allowListFlagSuffix identifies the flags whose patterns grant an IP-check
// bypass. Deny-lists are never audited: they always apply, cannot be bypassed,
// and a loose deny-list is safe rather than dangerous.
const allowListFlagSuffix = "-allow-list"

// warnRiskyAllowList logs one warning per allow-list entry that matches more
// URLs than its author is likely to intend.
//
// It warns and never fails: operators depend on loose patterns today, and
// rejecting them at startup would break running deployments.
func (f *ParsedFlags) warnRiskyAllowList(name string, vals []string) {
	if !strings.HasSuffix(name, allowListFlagSuffix) {
		return
	}

	findings := AuditAllowList(vals)
	if len(findings) == 0 {
		return
	}

	// The logger is nil until the entry point initializes it, which happens
	// before any module is provisioned. Tests and embedders that call this
	// method directly get no logger, and must not panic for it.
	logger := log.Logger()
	if logger == nil {
		return
	}

	for _, finding := range findings {
		// Provision has no context.Context to propagate, so the trace-aware
		// logging convention is satisfied with a background context.
		logger.WarnContext(
			context.Background(),
			f.allowListWarning(name, finding),
			slog.String("flag", "--"+name),
			slog.String("env", EnvVarName(name)),
			slog.Int("entry", finding.Index+1),
			slog.String("reason", string(finding.Risk)),
		)
	}
}

// allowListWarning builds the operator-facing message for a finding. It names
// the flag and its environment variable, and, when they exist, the IP-check
// flags the entry silently disables.
func (f *ParsedFlags) allowListWarning(name string, finding AllowListFinding) string {
	var b strings.Builder

	// Print the pattern raw rather than quoted: %q escapes every backslash, so
	// the operator would not recognize the value they set.
	fmt.Fprintf(&b, "--%s (%s) entry %d '%s' ", name, EnvVarName(name), finding.Index+1, finding.Pattern)

	switch finding.Risk {
	case AllowListRiskUnanchored:
		b.WriteString("is not anchored with ^, so it matches anywhere in the URL and a URL such as http://attacker.example/?u=trusted.example.com passes. ")
	case AllowListRiskUnanchoredBranch:
		b.WriteString("has an alternation branch that is not anchored with ^, and that branch matches anywhere in the URL. ")
	case AllowListRiskCatchAll:
		b.WriteString("matches every URL. ")
	case AllowListRiskOpenHost:
		b.WriteString("does not terminate the host, so it also matches suffix hosts such as http://trusted.example.com.attacker.example/. ")
	}

	b.WriteString(f.bypassSentence(name))

	switch finding.Risk {
	case AllowListRiskUnanchored, AllowListRiskUnanchoredBranch:
		b.WriteString("Anchor every branch with ^ and end the host with /, :, or $.")
	case AllowListRiskCatchAll:
		b.WriteString("Restrict the entry to the hosts you trust, or unset the flag.")
	case AllowListRiskOpenHost:
		b.WriteString("End the host with /, :, $, or a group such as (:|/|$).")
	}

	return b.String()
}

// bypassSentence names the IP-check flags an allow-list match skips, when the
// module registers them.
func (f *ParsedFlags) bypassSentence(name string) string {
	prefix := strings.TrimSuffix(name, allowListFlagSuffix)

	private, public := prefix+"-deny-private-ips", prefix+"-deny-public-ips"
	if f.Lookup(private) == nil || f.Lookup(public) == nil {
		return "A URL that matches the allow-list skips the private and public IP checks. "
	}

	return fmt.Sprintf(
		"A URL that matches the allow-list skips --%s (%s) and --%s (%s). ",
		private, EnvVarName(private), public, EnvVarName(public),
	)
}

// EnvVarName returns the environment variable that overrides the flag given by
// name. The entry point derives the same name when it applies environment
// overrides, so operator-facing messages can name both without drifting.
func EnvVarName(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}
