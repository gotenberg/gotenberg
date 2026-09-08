package pdfengines

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

func TestForEachInputPath(t *testing.T) {
	for _, tc := range []struct {
		scenario   string
		inputPaths []string
		fn         func(inputPath string) error
		expectErr  string
	}{
		{
			scenario:   "no input path",
			inputPaths: nil,
			fn:         func(string) error { return errors.New("must not run") },
		},
		{
			scenario:   "single input path",
			inputPaths: []string{"a.pdf"},
			fn:         func(string) error { return nil },
		},
		{
			scenario:   "single input path with error",
			inputPaths: []string{"a.pdf"},
			fn:         func(p string) error { return fmt.Errorf("boom %s", p) },
			expectErr:  "boom a.pdf",
		},
		{
			scenario:   "many input paths",
			inputPaths: []string{"a.pdf", "b.pdf", "c.pdf", "d.pdf", "e.pdf"},
			fn:         func(string) error { return nil },
		},
		{
			scenario:   "error is the first in input order, not the first to arrive",
			inputPaths: []string{"a.pdf", "b.pdf", "c.pdf"},
			fn: func(p string) error {
				// "c.pdf" fails without delay so that it lands well before
				// "b.pdf"; the reported error must still be "b.pdf".
				if p == "b.pdf" {
					var counter int
					for i := range 5_000_000 {
						counter += i
					}
					return fmt.Errorf("slow failure %s (%d)", p, counter%1)
				}
				if p == "c.pdf" {
					return fmt.Errorf("fast failure %s", p)
				}
				return nil
			},
			expectErr: "slow failure b.pdf (0)",
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			err := forEachInputPath(new(api.Context), tc.inputPaths, tc.fn)

			if tc.expectErr == "" {
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error %q but got none", tc.expectErr)
			}

			if err.Error() != tc.expectErr {
				t.Fatalf("expected error %q but got %q", tc.expectErr, err.Error())
			}
		})
	}
}

func TestForEachInputPathRunsEveryPath(t *testing.T) {
	inputPaths := make([]string, 50)
	for i := range inputPaths {
		inputPaths[i] = fmt.Sprintf("%d.pdf", i)
	}

	var (
		mu   sync.Mutex
		seen = make(map[string]int)
	)

	err := forEachInputPath(new(api.Context), inputPaths, func(inputPath string) error {
		mu.Lock()
		defer mu.Unlock()
		seen[inputPath]++
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if len(seen) != len(inputPaths) {
		t.Fatalf("expected %d distinct paths but got %d", len(inputPaths), len(seen))
	}

	for path, count := range seen {
		if count != 1 {
			t.Fatalf("expected '%s' to run once but it ran %d times", path, count)
		}
	}
}

func TestForEachInputPathRespectsTheSlotCeiling(t *testing.T) {
	previousSlots, previousMax := engineExtraSlots, maxFileConcurrency
	defer func() { engineExtraSlots, maxFileConcurrency = previousSlots, previousMax }()

	// One request may run its reserved unit plus ceiling-1 borrowed ones.
	const ceiling = 3
	maxFileConcurrency = ceiling
	engineExtraSlots = make(chan struct{}, ceiling-1)

	inputPaths := make([]string, 40)
	for i := range inputPaths {
		inputPaths[i] = fmt.Sprintf("%d.pdf", i)
	}

	var inFlight, peak atomic.Int64

	err := forEachInputPath(new(api.Context), inputPaths, func(string) error {
		current := inFlight.Add(1)
		defer inFlight.Add(-1)

		for {
			observed := peak.Load()
			if current <= observed || peak.CompareAndSwap(observed, current) {
				break
			}
		}

		// Hold the slot long enough that the ceiling would be exceeded if it
		// were not enforced.
		var counter int
		for i := range 200_000 {
			counter += i
		}
		_ = counter

		return nil
	})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if peak.Load() > ceiling {
		t.Fatalf("expected at most %d concurrent runs but observed %d", ceiling, peak.Load())
	}
}

func TestForEachInputPathHonorsCancellation(t *testing.T) {
	previousSlots, previousMax := engineExtraSlots, maxFileConcurrency
	defer func() { engineExtraSlots, maxFileConcurrency = previousSlots, previousMax }()

	// Concurrent path, with the shared pool exhausted by another request, so
	// only this request's reserved unit is available.
	maxFileConcurrency = 3
	engineExtraSlots = make(chan struct{}, 2)
	engineExtraSlots <- struct{}{}
	engineExtraSlots <- struct{}{}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Neither source of capacity is available: the pool is exhausted by other
	// requests and this request's reserved unit is already in use by one of its
	// own files. A waiter must observe the cancelled context rather than block
	// forever. Driving acquireEngineSlot directly keeps that deterministic:
	// through forEachInputPath the reserved unit is reusable, so whether a
	// given file waits at all depends on how fast the file before it finishes.
	inUse := make(chan struct{}, 1)

	release, err := acquireEngineSlot(&api.Context{Context: cancelledCtx}, inUse)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled but got: %v", err)
	}

	if release != nil {
		t.Fatal("expected no release function when acquisition fails")
	}

	// A cancelled request stops taking on work even when capacity is free,
	// rather than deciding on the coin flip a ready select would give.
	inUse <- struct{}{}

	_, err = acquireEngineSlot(&api.Context{Context: cancelledCtx}, inUse)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled with the reserved unit free but got: %v", err)
	}

	// Live request, free reserved unit: acquired and handed back.
	release, err = acquireEngineSlot(&api.Context{Context: context.Background()}, inUse)
	if err != nil {
		t.Fatalf("expected the reserved unit to be acquired but got: %v", err)
	}

	release()

	if len(inUse) != 1 {
		t.Fatalf("expected the reserved unit to be returned but the channel holds %d", len(inUse))
	}
}

func TestForEachInputPathCompletesWithACancelledContext(t *testing.T) {
	previousSlots, previousMax := engineExtraSlots, maxFileConcurrency
	defer func() { engineExtraSlots, maxFileConcurrency = previousSlots, previousMax }()

	maxFileConcurrency = 3
	engineExtraSlots = make(chan struct{}, 2)
	engineExtraSlots <- struct{}{}
	engineExtraSlots <- struct{}{}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)
		_ = forEachInputPath(&api.Context{Context: cancelledCtx}, []string{"a.pdf", "b.pdf", "c.pdf"}, func(string) error {
			return nil
		})
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("forEachInputPath hung on a cancelled context with the shared pool exhausted")
	}
}

func TestForEachInputPathNeverThrottlesBelowOnePerRequest(t *testing.T) {
	previousSlots, previousMax := engineExtraSlots, maxFileConcurrency
	defer func() { engineExtraSlots, maxFileConcurrency = previousSlots, previousMax }()

	// A small ceiling against far more concurrent requests than it covers.
	// Before the shared pool existed each of these ran a binary of its own, so
	// the pool must not drop aggregate concurrency below one per request.
	const (
		ceiling  = 2
		requests = 8
	)

	maxFileConcurrency = ceiling
	engineExtraSlots = make(chan struct{}, ceiling-1)

	// Every runner announces itself and then blocks, so the count of arrivals
	// is the true simultaneous concurrency rather than whatever the scheduler
	// happened to overlap.
	arrived := make(chan struct{}, requests*3)
	release := make(chan struct{})

	var wg sync.WaitGroup
	for range requests {
		wg.Go(func() {
			_ = forEachInputPath(new(api.Context), []string{"a.pdf", "b.pdf", "c.pdf"}, func(string) error {
				arrived <- struct{}{}
				<-release
				return nil
			})
		})
	}

	// One runner per request must be able to start without waiting on the
	// shared pool. If the pool governed the total instead of the surplus, only
	// `ceiling` runners would ever arrive and this would time out.
	for i := range requests {
		select {
		case <-arrived:
		case <-time.After(10 * time.Second):
			t.Fatalf("only %d runners started concurrently, expected at least one per request (%d); the shared pool is throttling requests against each other", i, requests)
		}
	}

	close(release)
	wg.Wait()
}

func TestForEachInputPathIsSequentialAtTheDefaultCeiling(t *testing.T) {
	if defaultMaxConcurrency != 1 {
		t.Fatalf("this test pins the default as a no-op, but defaultMaxConcurrency is %d", defaultMaxConcurrency)
	}

	previousSlots, previousMax := engineExtraSlots, maxFileConcurrency
	defer func() { engineExtraSlots, maxFileConcurrency = previousSlots, previousMax }()

	maxFileConcurrency = defaultMaxConcurrency
	engineExtraSlots = make(chan struct{}, defaultMaxConcurrency-1)

	// At the default the helper must behave exactly like the sequential loops
	// it replaced: files in input order, and no file attempted once one has
	// failed.
	var order []string

	err := forEachInputPath(new(api.Context), []string{"a.pdf", "b.pdf", "c.pdf", "d.pdf"}, func(inputPath string) error {
		order = append(order, inputPath)
		if inputPath == "b.pdf" {
			return errors.New("boom")
		}
		return nil
	})

	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected error \"boom\" but got: %v", err)
	}

	if len(order) != 2 || order[0] != "a.pdf" || order[1] != "b.pdf" {
		t.Fatalf("expected the run to stop after b.pdf in input order but got %v", order)
	}
}

func TestForEachInputPathIndexed(t *testing.T) {
	inputPaths := []string{"a.pdf", "b.pdf", "c.pdf", "d.pdf"}
	collected := make([]string, len(inputPaths))

	err := forEachInputPathIndexed(new(api.Context), inputPaths, func(i int, inputPath string) error {
		collected[i] = inputPath
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	for i, inputPath := range inputPaths {
		if collected[i] != inputPath {
			t.Fatalf("expected index %d to hold '%s' but got '%s'", i, inputPath, collected[i])
		}
	}
}
