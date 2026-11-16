package gotenberg

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// fakeProc writes a proc filesystem: each process has a parent and, unless
// it is a zombie, an RssAnon in kB.
func fakeProc(t *testing.T, processes map[int]fakeProcess) {
	t.Helper()

	dir := t.TempDir()
	for pid, p := range processes {
		pidDir := filepath.Join(dir, strconv.Itoa(pid))
		if err := os.MkdirAll(pidDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// A command name with spaces and parentheses, as Chromium's are.
		stat := strconv.Itoa(pid) + " (chrome (renderer) x) S " + strconv.Itoa(p.parent) + " 1 1 0 -1"
		if err := os.WriteFile(filepath.Join(pidDir, "stat"), []byte(stat), 0o644); err != nil {
			t.Fatal(err)
		}

		status := "Name:\tchrome\nState:\tS (sleeping)\n"
		if !p.zombie {
			status += "VmRSS:\t   999999 kB\nRssAnon:\t   " + strconv.Itoa(p.kb) + " kB\nRssFile:\t   777 kB\n"
		}
		if err := os.WriteFile(filepath.Join(pidDir, "status"), []byte(status), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Not a process.
	if err := os.MkdirAll(filepath.Join(dir, "sys"), 0o755); err != nil {
		t.Fatal(err)
	}

	previous := procDir
	procDir = dir
	t.Cleanup(func() { procDir = previous })
}

type fakeProcess = struct {
	parent int
	kb     int
	zombie bool
}

func TestProcessTreeMemory(t *testing.T) {
	fakeProc(t, map[int]fakeProcess{
		1:  {parent: 0, kb: 5},
		10: {parent: 1, kb: 100},
		11: {parent: 10, kb: 200},
		12: {parent: 11, kb: 300},
		13: {parent: 10, zombie: true},
		20: {parent: 1, kb: 4000},
	})

	used, err := ProcessTreeMemory(10)
	if err != nil {
		t.Fatal(err)
	}

	if want := uint64(600 * 1024); used != want {
		t.Errorf("ProcessTreeMemory(10) = %d, want %d: the process and its descendants, not its siblings", used, want)
	}

	_, err = ProcessTreeMemory(99)
	if err == nil {
		t.Error("ProcessTreeMemory of a process that is gone succeeded")
	}
}

func TestWatchMemory(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	t.Run("over the limit", func(t *testing.T) {
		fakeProc(t, map[int]fakeProcess{10: {parent: 1, kb: 2048}})

		got := make(chan uint64, 2)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		WatchMemory(ctx, logger, 10, 1<<20, func(used uint64) { got <- used })

		select {
		case used := <-got:
			if used != 2048*1024 {
				t.Errorf("exceeded(%d), want %d", used, 2048*1024)
			}
		default:
			t.Fatal("exceeded was not called")
		}
	})

	t.Run("under the limit until the process exits", func(t *testing.T) {
		fakeProc(t, map[int]fakeProcess{10: {parent: 1, kb: 10}})

		gone := filepath.Join(procDir, "10")
		removed := make(chan struct{})
		go func() {
			defer close(removed)
			time.Sleep(3 * MemoryWatchInterval)
			_ = os.RemoveAll(gone)
		}()
		defer func() { <-removed }()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		WatchMemory(ctx, logger, 10, 1<<20, func(uint64) { t.Error("exceeded was called under the limit") })

		if ctx.Err() != nil {
			t.Fatal("WatchMemory outlived the process")
		}
	})

	t.Run("no limit", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		WatchMemory(ctx, logger, 10, 0, func(uint64) { t.Error("exceeded was called without a limit") })

		if ctx.Err() != nil {
			t.Fatal("WatchMemory waited without a limit")
		}
	})
}

func TestFormatMemory(t *testing.T) {
	if got := FormatMemory(900 << 20); got != "900 MiB" {
		t.Errorf("FormatMemory = %q, want 900 MiB", got)
	}
}
