package chromium

import (
	"log/slog"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v9/pkg/gotenberg"
)

// The memory watch runs while a conversion is in flight, and kills the
// browser process when Chromium holds more than the limit.
func TestChromiumBrowser_WatchMemory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the memory limit reads /proc")
	}

	logger := slog.New(slog.DiscardHandler)

	start := func(t *testing.T) (*exec.Cmd, <-chan error) {
		t.Helper()

		// A stand-in for the browser process: any process holds more than
		// one byte of anonymous memory.
		cmd := exec.Command("sleep", "30")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill() })

		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()

		return cmd, exited
	}

	t.Run("idle, not watched", func(t *testing.T) {
		cmd, exited := start(t)
		b := &chromiumBrowser{arguments: browserArguments{maxMemory: 1}, pid: cmd.Process.Pid}

		b.watchMemory(logger)()

		select {
		case <-exited:
			t.Fatal("the watch killed the browser once no conversion was in flight")
		case <-time.After(5 * gotenberg.MemoryWatchInterval):
		}

		if b.memoryExceeded.Load() != 0 {
			t.Fatal("memoryExceeded set without a conversion in flight")
		}
	})

	t.Run("in flight, over the limit", func(t *testing.T) {
		cmd, exited := start(t)
		b := &chromiumBrowser{arguments: browserArguments{maxMemory: 1}, pid: cmd.Process.Pid}

		releaseFirst := b.watchMemory(logger)
		releaseSecond := b.watchMemory(logger)
		defer releaseSecond()
		releaseFirst()

		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			t.Fatal("the browser still runs over the memory limit")
		}

		if b.memoryExceeded.Load() == 0 {
			t.Fatal("memoryExceeded not set once the browser was killed")
		}
	})

	t.Run("no limit", func(t *testing.T) {
		b := &chromiumBrowser{pid: 1}
		b.watchMemory(logger)()

		if b.inFlight != 0 || b.stopWatching != nil {
			t.Fatal("watchMemory counted a conversion without a limit")
		}
	})
}

func TestChromiumErrorType_MemoryLimit(t *testing.T) {
	if got := chromiumErrorType(ErrChromiumMemoryLimitExceeded); got != "chromium_memory_limit" {
		t.Errorf("error type = %q, want chromium_memory_limit", got)
	}
}
