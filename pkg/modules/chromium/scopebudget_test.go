package chromium

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
)

func TestScopeMatchBudget(t *testing.T) {
	t.Run("allows matching while credit remains", func(t *testing.T) {
		b := newScopeMatchBudget(time.Second)
		if !b.tryAcquire() {
			t.Fatal("tryAcquire() = false on a fresh budget, want true")
		}
	})

	t.Run("denies matching once exhausted", func(t *testing.T) {
		b := newScopeMatchBudget(time.Second)
		b.consume(time.Second)
		if b.tryAcquire() {
			t.Error("tryAcquire() = true after the budget was spent, want false")
		}
	})

	t.Run("saturates at zero instead of wrapping into credit", func(t *testing.T) {
		b := newScopeMatchBudget(time.Second)
		b.consume(time.Hour)
		if got := b.remaining.Load(); got != 0 {
			t.Errorf("remaining = %d, want 0", got)
		}
		if b.tryAcquire() {
			t.Error("tryAcquire() = true after an overlong match, want false")
		}
	})

	t.Run("a spent budget stays spent", func(t *testing.T) {
		b := newScopeMatchBudget(time.Second)
		b.consume(time.Second)
		b.consume(time.Millisecond)
		if got := b.remaining.Load(); got != 0 {
			t.Errorf("remaining = %d, want 0", got)
		}
	})

	t.Run("is safe for concurrent use", func(t *testing.T) {
		const goroutines = 64
		// Each goroutine spends 1ms against a budget of half that many
		// milliseconds, so the total spend overshoots it.
		b := newScopeMatchBudget(time.Duration(goroutines/2) * time.Millisecond)

		var wg sync.WaitGroup
		for range goroutines {
			wg.Go(func() {
				b.tryAcquire()
				b.consume(time.Millisecond)
			})
		}
		wg.Wait()

		if got := b.remaining.Load(); got != 0 {
			t.Errorf("remaining = %d, want 0", got)
		}
	})
}

// TestScopeMatchBudget_BoundsCatastrophicBacktracking is the regression test for
// the amplification: many scoped headers matched against a hostile URL must cost
// the budget, not a multiple of it.
// See https://github.com/gotenberg/gotenberg/issues/1588.
func TestScopeMatchBudget_BoundsCatastrophicBacktracking(t *testing.T) {
	const (
		headers = 16
		budget  = 200 * time.Millisecond
	)

	// Nested quantifier with no possible match: classic catastrophic
	// backtracking.
	pattern := compileScopePattern(t, `(a+)+b`)
	url := "http://example.com/" + strings.Repeat("a", 40)

	b := newScopeMatchBudget(budget)

	start := time.Now()
	var matched int
	for range headers {
		if !b.tryAcquire() {
			break
		}
		matchStart := time.Now()
		_, _ = pattern.MatchString(url)
		b.consume(time.Since(matchStart))
		matched++
	}
	elapsed := time.Since(start)

	if matched == headers {
		t.Errorf("all %d headers were matched, want the budget to stop matching early", headers)
	}

	// Each match is separately capped at extraHttpHeaderScopeMatchTimeout, so
	// the worst case is the budget plus one final match that started with the
	// last of the credit. Generous slack keeps this stable on a loaded CI box.
	ceiling := budget + extraHttpHeaderScopeMatchTimeout + time.Second
	if elapsed > ceiling {
		t.Errorf("matching took %s, want at most %s", elapsed, ceiling)
	}
}

func compileScopePattern(t *testing.T, pattern string) *regexp2.Regexp {
	t.Helper()

	p, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		t.Fatalf("compile %q: %v", pattern, err)
	}
	p.MatchTimeout = extraHttpHeaderScopeMatchTimeout

	return p
}
