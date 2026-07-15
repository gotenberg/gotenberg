package chromium

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

// TestChromiumBrowser_Start_rejectsOverlappingStart guards against the latch
// reported in https://github.com/gotenberg/gotenberg/issues/1599. When the
// supervisor abandons a Start goroutine on request-deadline expiry, that
// goroutine keeps running and holds startMu (and the pinning proxy it started)
// until it unwinds. A second Start must be refused rather than proceeding to
// start the pinning proxy a second time.
func TestChromiumBrowser_Start_rejectsOverlappingStart(t *testing.T) {
	b := &chromiumBrowser{initialCtx: context.Background()}

	// Simulate a Start still in flight.
	b.startMu.Lock()
	defer b.startMu.Unlock()

	err := b.Start(slog.New(slog.DiscardHandler))
	if err == nil {
		t.Fatal("expected an error when a start is already in progress, got nil")
	}
	if !strings.Contains(err.Error(), "already in progress") {
		t.Fatalf("expected an 'already in progress' error, got %q", err)
	}

	// The guard must return before touching any startup resource, so no user
	// profile directory is created and the browser stays not started.
	if b.userProfileDirPath != "" {
		t.Fatalf("expected no user profile directory to be created, got %q", b.userProfileDirPath)
	}
	if b.isStarted.Load() {
		t.Fatal("expected the browser to stay not started")
	}
}
