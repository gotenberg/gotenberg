package gotenberg

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// procDir is the proc filesystem the memory of a process tree is read from.
// Tests point it at a fixture.
var procDir = "/proc"

// MemoryWatchInterval is how often [WatchMemory] reads the memory of the
// process tree it watches. A process that allocates 100 MB per second, as a
// runaway conversion does, goes about 10 MB over the limit before the read
// that catches it.
const MemoryWatchInterval = 100 * time.Millisecond

// ProcessTreeMemory returns the anonymous resident memory, in bytes, of the
// process pid and of every process it descends from: the RssAnon lines of
// their /proc/<pid>/status. Anonymous memory is what a process allocates, so a
// runaway allocation shows there, and it leaves out the shared libraries each
// process of a tree maps, which resident memory would count once per process.
// It fails when pid is gone, or when the proc filesystem is missing, as off
// Linux.
func ProcessTreeMemory(pid int) (uint64, error) {
	root, err := anonymousMemory(pid)
	if err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(procDir)
	if err != nil {
		return 0, fmt.Errorf("list processes: %w", err)
	}

	children := make(map[int][]int)
	for _, entry := range entries {
		child, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		parent, err := parentPid(child)
		if err != nil {
			// The process exited while the list was read.
			continue
		}

		children[parent] = append(children[parent], child)
	}

	total := root
	queue := children[pid]
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]

		used, err := anonymousMemory(next)
		if err == nil {
			total += used
		}

		queue = append(queue, children[next]...)
	}

	return total, nil
}

// anonymousMemory returns the RssAnon of the process pid, in bytes.
func anonymousMemory(pid int) (uint64, error) {
	status, err := os.ReadFile(filepath.Join(procDir, strconv.Itoa(pid), "status"))
	if err != nil {
		return 0, fmt.Errorf("read the status of process %d: %w", pid, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(status))
	for scanner.Scan() {
		value, found := bytes.CutPrefix(scanner.Bytes(), []byte("RssAnon:"))
		if !found {
			continue
		}

		// The value reads "  123456 kB".
		fields := bytes.Fields(value)
		if len(fields) != 2 || string(fields[1]) != "kB" {
			return 0, fmt.Errorf("read the status of process %d: unexpected RssAnon %q", pid, value)
		}

		kb, err := strconv.ParseUint(string(fields[0]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("read the status of process %d: %w", pid, err)
		}

		return kb * 1024, nil
	}

	// A zombie has no memory, and no RssAnon line.
	return 0, nil
}

// parentPid returns the parent of the process pid, from its /proc/<pid>/stat.
func parentPid(pid int) (int, error) {
	stat, err := os.ReadFile(filepath.Join(procDir, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, err
	}

	// The command name, in parentheses, may hold spaces and parentheses: the
	// fields that follow start after its last closing parenthesis. They are
	// the state, then the parent.
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 {
		return 0, errors.New("malformed stat")
	}

	fields := bytes.Fields(stat[end+1:])
	if len(fields) < 2 {
		return 0, errors.New("malformed stat")
	}

	return strconv.Atoi(string(fields[1]))
}

// WatchMemory calls exceeded once, with the memory used, when the process tree
// of pid holds more than limit bytes of anonymous resident memory (see
// [ProcessTreeMemory]). It reads it at once, then every [MemoryWatchInterval],
// until ctx is done, exceeded runs, or the tree can no longer be read, once pid
// exits. It returns at once when limit is 0.
func WatchMemory(ctx context.Context, logger *slog.Logger, pid int, limit uint64, exceeded func(used uint64)) {
	if limit == 0 {
		return
	}

	ticker := time.NewTicker(MemoryWatchInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}

		used, err := ProcessTreeMemory(pid)
		if err != nil {
			logger.DebugContext(ctx, fmt.Sprintf("stop watching the memory of process %d: %s", pid, err))
			return
		}

		if used > limit {
			exceeded(used)
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// FormatMemory writes n bytes in mebibytes, as an operator reads a memory
// limit, such as "900 MiB".
func FormatMemory(n uint64) string {
	return fmt.Sprintf("%.0f MiB", float64(n)/(1<<20))
}
