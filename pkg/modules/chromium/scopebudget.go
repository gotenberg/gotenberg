package chromium

import (
	"sync/atomic"
	"time"
)

// scopeMatchBudgetPerConversion caps the total time a single conversion may
// spend matching scoped extra HTTP header patterns.
//
// The per-pattern MatchTimeout bounds one match, not their number: Chromium
// pauses every sub-resource request, and each paused request is matched against
// every scoped header. Without a shared budget the total is the product of the
// two, both of which the client controls.
// See https://github.com/gotenberg/gotenberg/issues/1588.
const scopeMatchBudgetPerConversion = 5 * time.Second

// scopeMatchBudget is a time allowance shared by every scope match of a
// conversion. It is safe for concurrent use: paused requests are handled on
// their own goroutines.
type scopeMatchBudget struct {
	remaining atomic.Int64
}

// newScopeMatchBudget returns a [scopeMatchBudget] allowing d of matching.
func newScopeMatchBudget(d time.Duration) *scopeMatchBudget {
	b := new(scopeMatchBudget)
	b.remaining.Store(int64(d))
	return b
}

// tryAcquire reports whether the budget still allows a match.
func (b *scopeMatchBudget) tryAcquire() bool {
	return b.remaining.Load() > 0
}

// consume subtracts the time a match took. It saturates at zero so that a long
// match cannot wrap the counter back into credit.
func (b *scopeMatchBudget) consume(d time.Duration) {
	for {
		current := b.remaining.Load()
		if current <= 0 {
			return
		}

		next := max(current-int64(d), 0)

		if b.remaining.CompareAndSwap(current, next) {
			return
		}
	}
}
