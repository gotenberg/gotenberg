package gotenberg

import (
	"github.com/dlclark/regexp2"
)

// patternMatchAttempts caps how many times [MatchPattern] runs one pattern
// against one string. It is what keeps a pattern that is genuinely out of
// budget from retrying forever: three attempts bound its cost at three
// [PatternMatchTimeout], which is still two orders of magnitude below the
// --api-timeout (env API_TIMEOUT) the ceiling exists to protect.
const patternMatchAttempts = 3

// MatchPattern reports whether s matches pattern. It bounds the match by the
// pattern's MatchTimeout without the false timeouts that the bound alone
// produces.
//
// regexp2 does not time a match against [time.Now]. It derives the deadline
// from a process-global clock that a background goroutine advances every
// 100ms, and it tests that deadline on the very first step of the match.
// Anything that stops the whole process, a cgroup CPU-quota throttle or a long
// stop-the-world pause, also stops that goroutine, which then advances the
// clock by the full pause in a single write. A match holding a deadline from
// before that jump aborts whatever work it had done: a 366ns match against a
// short URL reports "match timeout after 250ms". The abort lands on whichever
// match straddles the jump rather than on a match that was slow, which is why
// it fires on an idle instance and against Gotenberg's own file:///tmp/ URLs.
// See https://github.com/gotenberg/gotenberg/issues/1659.
//
// Retrying separates the two cases. Catastrophic backtracking is
// deterministic: the same pattern against the same string exhausts the same
// budget on every attempt, so a genuine runaway still aborts, and costs at
// most patternMatchAttempts ceilings to prove it. A clock-induced abort needs
// the process to lose the CPU inside one specific match, which the next
// attempt does not reproduce.
//
// Elapsed time cannot make that call instead. A match frozen mid-flight
// reports the freeze as its own cost, 806ms against a 250ms ceiling in one
// measured run, so it is indistinguishable by wall clock from a match that
// really did spend its budget. Go exposes no per-goroutine CPU time, and
// process CPU time counts every other request in flight.
func MatchPattern(pattern *regexp2.Regexp, s string) (bool, error) {
	var err error

	for range patternMatchAttempts {
		var ok bool

		ok, err = pattern.MatchString(s)
		if err == nil {
			return ok, nil
		}
	}

	return false, err
}
