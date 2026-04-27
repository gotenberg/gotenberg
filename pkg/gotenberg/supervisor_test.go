package gotenberg

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProcessSupervisor_Launch(t *testing.T) {
	for _, tc := range []struct {
		scenario      string
		startError    error
		expectError   bool
		firstStartSet bool
	}{
		{
			scenario:      "successful launch",
			startError:    nil,
			expectError:   false,
			firstStartSet: true,
		},
		{
			scenario:      "failed launch",
			startError:    errors.New("start error"),
			expectError:   true,
			firstStartSet: false,
		},
		{
			scenario:      "process already started",
			startError:    nil,
			expectError:   false,
			firstStartSet: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			process := &ProcessMock{
				StartMock: func(logger *slog.Logger) error {
					return tc.startError
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0, 1, 0).(*processSupervisor)
			if tc.firstStartSet {
				ps.firstStart.Store(true)
			}

			err := ps.Launch()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.firstStartSet && !ps.firstStart.Load() {
				t.Error("expected firstStart to be set but it was not")
			}
		})
	}
}

func TestProcessSupervisor_Shutdown(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		stopError   error
		expectError bool
	}{
		{
			scenario:    "successful shutdown",
			stopError:   nil,
			expectError: false,
		},
		{
			scenario:    "failed shutdown",
			stopError:   errors.New("stop error"),
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			process := &ProcessMock{
				StopMock: func(logger *slog.Logger) error {
					return tc.stopError
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0, 1, 0)
			err := ps.Shutdown()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestProcessSupervisor_restart(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		startError  error
		stopError   error
		expectError bool
	}{
		{
			scenario:    "successful restart",
			startError:  nil,
			stopError:   nil,
			expectError: false,
		},
		{
			scenario:    "failed to stop during restart",
			startError:  nil,
			stopError:   errors.New("stop error"),
			expectError: false,
		},
		{
			scenario:    "failed to start during restart",
			startError:  errors.New("start error"),
			stopError:   nil,
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			process := &ProcessMock{
				StartMock: func(logger *slog.Logger) error {
					return tc.startError
				},
				StopMock: func(logger *slog.Logger) error {
					return tc.stopError
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0, 1, 0).(*processSupervisor)

			err := ps.restart()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestProcessSupervisor_Healthy(t *testing.T) {
	for _, tc := range []struct {
		scenario            string
		initiallyStarted    bool
		initiallyRestarting bool
		processHealthy      bool
		expectHealthy       bool
	}{
		{
			scenario:         "non-started process is healthy",
			initiallyStarted: false,
			expectHealthy:    true,
		},
		{
			scenario:            "restarting process is not healthy",
			initiallyStarted:    true,
			initiallyRestarting: true,
			expectHealthy:       false,
		},
		{
			scenario:         "process reports as healthy",
			initiallyStarted: true,
			processHealthy:   true,
			expectHealthy:    true,
		},
		{
			scenario:         "process reports as unhealthy",
			initiallyStarted: true,
			processHealthy:   false,
			expectHealthy:    false,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			process := &ProcessMock{
				HealthyMock: func(logger *slog.Logger) bool {
					return tc.processHealthy
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0, 1, 0).(*processSupervisor)
			if tc.initiallyStarted {
				ps.firstStart.Store(true)
			}
			if tc.initiallyRestarting {
				ps.isRestarting.Store(true)
			}

			healthy := ps.Healthy()

			if healthy != tc.expectHealthy {
				t.Fatalf("expected healthy to be %v but got %v", tc.expectHealthy, healthy)
			}
		})
	}
}

func TestProcessSupervisor_Run(t *testing.T) {
	for _, tc := range []struct {
		scenario             string
		initiallyStarted     bool
		isRestarting         bool
		startError           error
		processHealthy       bool
		maxReqLimit          int64
		tasksToRun           int
		taskError            error
		expectError          bool
		skipCallsCheck       bool
		expectedStartCalls   int64
		expectedHealthyCalls int64
		expectedStopCalls    int64
		currentQueueSize     int64
		maxQueueSize         int64
	}{
		{
			scenario:             "successfully run task on non-started process",
			initiallyStarted:     false,
			isRestarting:         false,
			processHealthy:       true,
			maxReqLimit:          2,
			tasksToRun:           1,
			expectError:          false,
			expectedStartCalls:   1,
			expectedHealthyCalls: 1,
			expectedStopCalls:    0,
		},
		{
			scenario:             "cannot launch non-started process",
			initiallyStarted:     false,
			isRestarting:         false,
			startError:           errors.New("launch error"),
			processHealthy:       true,
			maxReqLimit:          2,
			tasksToRun:           1,
			expectError:          true,
			expectedStartCalls:   1,
			expectedHealthyCalls: 0,
			expectedStopCalls:    0,
		},
		{
			scenario:             "run task with unhealthy process causing restart",
			initiallyStarted:     true,
			isRestarting:         false,
			processHealthy:       false,
			maxReqLimit:          2,
			tasksToRun:           1,
			expectError:          false,
			expectedStartCalls:   1,
			expectedHealthyCalls: 1,
			expectedStopCalls:    1,
		},
		{
			scenario:             "cannot restart unhealthy process",
			startError:           errors.New("start error"),
			initiallyStarted:     true,
			isRestarting:         false,
			processHealthy:       false,
			maxReqLimit:          2,
			tasksToRun:           1,
			expectError:          true,
			expectedStartCalls:   1,
			expectedHealthyCalls: 1,
			expectedStopCalls:    1,
		},
		{
			scenario:         "ErrProcessAlreadyRestarting",
			initiallyStarted: true,
			isRestarting:     true,
			processHealthy:   false,
			maxReqLimit:      1,
			tasksToRun:       1,
			expectError:      true,
			skipCallsCheck:   true,
		},
		{
			scenario:             "run tasks reaching max request limit causing restart",
			initiallyStarted:     true,
			isRestarting:         false,
			processHealthy:       true,
			maxReqLimit:          2,
			tasksToRun:           3,
			expectError:          false,
			expectedStartCalls:   1,
			expectedHealthyCalls: 3,
			expectedStopCalls:    1,
		},
		{
			scenario:             "auto-restart after reaching max request limit",
			startError:           errors.New("start error"),
			initiallyStarted:     true,
			isRestarting:         false,
			processHealthy:       true,
			maxReqLimit:          2,
			tasksToRun:           2,
			expectError:          true,
			expectedStartCalls:   1,
			expectedHealthyCalls: 2,
			expectedStopCalls:    1,
		},
		{
			scenario:             "task error",
			initiallyStarted:     true,
			isRestarting:         false,
			processHealthy:       true,
			maxReqLimit:          0,
			tasksToRun:           1,
			taskError:            errors.New("task error"),
			expectError:          true,
			expectedStartCalls:   0,
			expectedHealthyCalls: 1,
			expectedStopCalls:    0,
		},
		{
			scenario:             "queue size exceeded",
			initiallyStarted:     false,
			isRestarting:         false,
			processHealthy:       true,
			maxReqLimit:          2,
			tasksToRun:           1,
			expectError:          true,
			expectedStartCalls:   0,
			expectedHealthyCalls: 0,
			expectedStopCalls:    0,
			currentQueueSize:     1,
			maxQueueSize:         1,
		},
		{
			scenario:             "queue size not exceeded",
			initiallyStarted:     false,
			isRestarting:         false,
			processHealthy:       true,
			maxReqLimit:          2,
			tasksToRun:           1,
			expectError:          true,
			expectedStartCalls:   1,
			expectedHealthyCalls: 1,
			expectedStopCalls:    0,
			currentQueueSize:     1,
			maxQueueSize:         2,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			var startCalls, healthyCalls, stopCalls atomic.Int64
			startCalls.Store(0)
			healthyCalls.Store(0)
			stopCalls.Store(0)

			process := &ProcessMock{
				StartMock: func(logger *slog.Logger) error {
					startCalls.Add(1)
					return tc.startError
				},
				StopMock: func(logger *slog.Logger) error {
					stopCalls.Add(1)
					return nil
				},
				HealthyMock: func(logger *slog.Logger) bool {
					healthyCalls.Add(1)
					return tc.processHealthy
				},
			}

			ps := NewProcessSupervisor(logger, process, tc.maxReqLimit, tc.maxQueueSize, 1, 0).(*processSupervisor)
			if tc.initiallyStarted {
				ps.firstStart.Store(true)
			}
			if tc.isRestarting {
				ps.isRestarting.Store(true)
			}
			if tc.currentQueueSize > 0 {
				ps.reqQueueSize.Store(tc.currentQueueSize)
			}

			task := func() error {
				return tc.taskError
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			var wg sync.WaitGroup
			errorChan := make(chan error, tc.tasksToRun)

			for i := 0; i < tc.tasksToRun; i++ {
				wg.Go(func() {
					err := ps.Run(ctx, logger, task)
					if err != nil {
						errorChan <- err
					}
				})
			}

			wg.Wait()
			close(errorChan)

			for err := range errorChan {
				if tc.expectError && err == nil {
					t.Fatal("expected an error but got none")
				}

				if !tc.expectError && err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			}

			if tc.skipCallsCheck {
				return
			}

			// Making sure restarts are finished.
			ps.semaphore <- struct{}{}
			<-ps.semaphore

			if startCalls.Load() != tc.expectedStartCalls {
				t.Errorf("expected %d process.Start calls, got %d", tc.expectedStartCalls, startCalls.Load())
			}

			if healthyCalls.Load() != tc.expectedHealthyCalls {
				t.Errorf("expected %d process.Healthy calls, got %d", tc.expectedHealthyCalls, healthyCalls.Load())
			}

			if stopCalls.Load() != tc.expectedStopCalls {
				t.Errorf("expected %d process.Stop calls, got %d", tc.expectedStopCalls, stopCalls.Load())
			}
		})
	}
}

func TestProcessSupervisor_runWithDeadline(t *testing.T) {
	for _, tc := range []struct {
		scenario    string
		ctxDone     bool
		expectError bool
	}{
		{
			scenario:    "task finished",
			ctxDone:     false,
			expectError: false,
		},
		{
			scenario:    "context expired",
			ctxDone:     true,
			expectError: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			ps := NewProcessSupervisor(slog.New(slog.DiscardHandler), new(ProcessMock), 0, 0, 1, 0).(*processSupervisor)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			if tc.ctxDone {
				cancel()
			} else {
				defer cancel()
			}

			err := ps.runWithDeadline(ctx, func() error {
				return nil
			})

			if tc.expectError && err == nil {
				t.Fatal("expected an error but got none")
			}

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}

func TestProcessSupervisor_ReqQueueSize(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool {
			return true
		},
	}
	ps := NewProcessSupervisor(logger, process, 0, 0, 1, 0).(*processSupervisor)

	// Simulating a lock.
	ps.semaphore <- struct{}{}

	if ps.ReqQueueSize() != 0 {
		t.Fatalf("expected queue size to be 0 but got %d", ps.ReqQueueSize())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errorChan := make(chan error, 10)

	for range 10 {
		wg.Go(func() {
			err := ps.Run(ctx, logger, func() error {
				return nil
			})
			if err != nil {
				errorChan <- err
			}
		})
	}

	// We have to wait a little bit so that the request queue size may change.
	time.Sleep(10 * time.Millisecond)

	if ps.ReqQueueSize() != 10 {
		t.Fatalf("expected queue size to be 10 but got %d", ps.ReqQueueSize())
	}

	wg.Wait()
	close(errorChan)

	for err := range errorChan {
		if err == nil {
			t.Error("expected a lock error but got none")
		}
	}

	if ps.ReqQueueSize() != 0 {
		t.Errorf("expected queue size to be 0 but got %d", ps.ReqQueueSize())
	}
}

func TestProcessSupervisor_QueueSizeCAS(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool {
			return true
		},
	}

	maxQueueSize := int64(50)
	// maxConcurrency=1 so all goroutines block on the semaphore, exercising queue logic.
	ps := NewProcessSupervisor(logger, process, 0, maxQueueSize, 1, 0).(*processSupervisor)

	// Simulating a lock so that all goroutines queue up.
	ps.semaphore <- struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	goroutines := 100
	var wg sync.WaitGroup
	var exceeded atomic.Int64

	for range goroutines {
		wg.Go(func() {
			err := ps.Run(ctx, logger, func() error {
				return nil
			})
			if err != nil {
				if errors.Is(err, ErrMaximumQueueSizeExceeded) {
					exceeded.Add(1)
				}
			}
		})
	}

	// Wait a bit for goroutines to queue up.
	time.Sleep(50 * time.Millisecond)

	currentQueue := ps.ReqQueueSize()
	if currentQueue > maxQueueSize {
		t.Fatalf("queue size %d exceeded max %d", currentQueue, maxQueueSize)
	}

	cancel()
	wg.Wait()

	if exceeded.Load() < int64(goroutines)-maxQueueSize {
		t.Errorf("expected at least %d rejections, got %d", goroutines-int(maxQueueSize), exceeded.Load())
	}
}

func TestProcessSupervisor_QueueSizeIncludesActiveTasks(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool {
			return true
		},
	}

	// maxQueueSize=1, maxConcurrency=1: only one request at a time.
	ps := NewProcessSupervisor(logger, process, 0, 1, 1, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	taskStarted := make(chan struct{})
	taskDone := make(chan struct{})

	// Start a long-running task that holds the slot.
	var wg sync.WaitGroup
	wg.Go(func() {
		err := ps.Run(ctx, logger, func() error {
			close(taskStarted)
			<-taskDone
			return nil
		})
		if err != nil {
			t.Errorf("first task: unexpected error: %v", err)
		}
	})

	// Wait for the first task to be running.
	<-taskStarted

	// A second request should be rejected immediately because the queue
	// slot is still held by the active task.
	err := ps.Run(ctx, logger, func() error {
		return nil
	})
	if !errors.Is(err, ErrMaximumQueueSizeExceeded) {
		t.Fatalf("expected ErrMaximumQueueSizeExceeded but got: %v", err)
	}

	close(taskDone)
	wg.Wait()
}

func TestProcessSupervisor_RestartsCount(t *testing.T) {
	for _, tc := range []struct {
		scenario              string
		initialRestartsCount  int64
		restartAttempts       int
		startError            error
		stopError             error
		expectedRestartsCount int64
	}{
		{
			scenario:              "no restarts, counter remains 0",
			initialRestartsCount:  0,
			restartAttempts:       0,
			expectedRestartsCount: 0,
		},
		{
			scenario:              "successful restart increases counter",
			initialRestartsCount:  0,
			restartAttempts:       1,
			startError:            nil,
			stopError:             nil,
			expectedRestartsCount: 1,
		},
		{
			scenario:              "failed to stop during restart, no impact",
			initialRestartsCount:  0,
			restartAttempts:       1,
			startError:            nil,
			stopError:             errors.New("stop error"),
			expectedRestartsCount: 1,
		},
		{
			scenario:              "multiple successful restarts",
			initialRestartsCount:  0,
			restartAttempts:       3,
			startError:            nil,
			stopError:             nil,
			expectedRestartsCount: 3,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)

			process := &ProcessMock{
				StartMock: func(logger *slog.Logger) error {
					return tc.startError
				},
				StopMock: func(logger *slog.Logger) error {
					return tc.stopError
				},
			}

			ps := NewProcessSupervisor(logger, process, 0, 0, 1, 0).(*processSupervisor)
			ps.restartsCounter.Store(tc.initialRestartsCount)

			for i := 0; i < tc.restartAttempts; i++ {
				_ = ps.restart()
			}

			actualRestartsCount := ps.RestartsCount()
			if actualRestartsCount != tc.expectedRestartsCount {
				t.Fatalf("expected restarts count to be %d, but got  %d", tc.expectedRestartsCount, actualRestartsCount)
			}
		})
	}
}

func TestProcessSupervisor_ConcurrentRun(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	var startCalls atomic.Int64
	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			startCalls.Add(1)
			return nil
		},
		StopMock: func(logger *slog.Logger) error {
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool {
			return true
		},
	}

	maxConcurrency := int64(3)
	ps := NewProcessSupervisor(logger, process, 0, 0, maxConcurrency, 0).(*processSupervisor)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var running atomic.Int64
	var maxRunning atomic.Int64

	var wg sync.WaitGroup
	tasks := 6

	for range tasks {
		wg.Go(func() {
			err := ps.Run(ctx, logger, func() error {
				cur := running.Add(1)
				for {
					old := maxRunning.Load()
					if cur <= old || maxRunning.CompareAndSwap(old, cur) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				running.Add(-1)
				return nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	wg.Wait()

	observed := maxRunning.Load()
	if observed > maxConcurrency {
		t.Fatalf("expected at most %d concurrent tasks, but observed %d", maxConcurrency, observed)
	}
	if observed < 2 {
		t.Fatalf("expected concurrent execution (at least 2 tasks running simultaneously), but observed max %d", observed)
	}

	if startCalls.Load() != 1 {
		t.Errorf("expected 1 start call, got %d", startCalls.Load())
	}
}

func TestProcessSupervisor_RestartDrainsAllSlots(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			return nil
		},
		StopMock: func(logger *slog.Logger) error {
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool {
			return true
		},
	}

	maxConcurrency := int64(3)
	ps := NewProcessSupervisor(logger, process, 3, 0, maxConcurrency, 0).(*processSupervisor)
	ps.firstStart.Store(true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	tasks := 3

	for range tasks {
		wg.Go(func() {
			err := ps.Run(ctx, logger, func() error {
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	wg.Wait()

	// Wait for the async restart goroutine to complete.
	deadline := time.After(5 * time.Second)
	for ps.RestartsCount() < 1 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for restart to complete")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if ps.RestartsCount() != 1 {
		t.Fatalf("expected 1 restart, got %d", ps.RestartsCount())
	}
}

func TestProcessSupervisor_IdleShutdown(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	var stopCalls atomic.Int64
	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			return nil
		},
		StopMock: func(logger *slog.Logger) error {
			stopCalls.Add(1)
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool {
			return true
		},
	}

	idleTimeout := 50 * time.Millisecond
	ps := NewProcessSupervisor(logger, process, 0, 0, 1, idleTimeout).(*processSupervisor)

	ctx := context.Background()
	err := ps.Run(ctx, logger, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for idle shutdown to fire.
	deadline := time.After(2 * time.Second)
	for ps.firstStart.Load() {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for idle shutdown")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if stopCalls.Load() < 1 {
		t.Fatal("expected process to be stopped via idle shutdown")
	}

	// Verify re-launch on next request.
	err = ps.Run(ctx, logger, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error on re-launch: %v", err)
	}

	if !ps.firstStart.Load() {
		t.Fatal("expected process to be re-launched after idle shutdown")
	}
}

func TestProcessSupervisor_IdleShutdownSkippedWhenActive(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	var stopCalls atomic.Int64
	taskRunning := make(chan struct{})
	taskDone := make(chan struct{})

	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			return nil
		},
		StopMock: func(logger *slog.Logger) error {
			stopCalls.Add(1)
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool {
			return true
		},
	}

	idleTimeout := 50 * time.Millisecond
	ps := NewProcessSupervisor(logger, process, 0, 0, 1, idleTimeout)

	ctx := context.Background()
	go func() {
		_ = ps.Run(ctx, logger, func() error {
			close(taskRunning)
			<-taskDone
			return nil
		})
	}()

	<-taskRunning

	// Wait longer than the idle timeout while a task is active.
	time.Sleep(idleTimeout * 3)

	if stopCalls.Load() > 0 {
		t.Fatal("idle shutdown should not fire while a task is active")
	}

	close(taskDone)
}

// TestProcessSupervisor_FirstStart_RetriesAfterFailure pins the fix for
// issue #1538: if Launch fails on the first request (e.g. an aggressive
// --libreoffice-start-timeout), the supervisor must let the *next*
// request retry the launch instead of replaying the cached error to
// every subsequent caller until the container is restarted.
func TestProcessSupervisor_FirstStart_RetriesAfterFailure(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	var startCalls atomic.Int32
	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			n := startCalls.Add(1)
			if n == 1 {
				return errors.New("first start: timeout")
			}
			return nil
		},
		HealthyMock: func(logger *slog.Logger) bool { return true },
	}

	ps := NewProcessSupervisor(logger, process, 0, 0, 1, 0).(*processSupervisor)

	// First request: Launch fails, Run returns the error, firstStart
	// must remain false so the cached failure does not persist.
	if err := ps.ensureStarted(context.Background()); err == nil {
		t.Fatal("first ensureStarted: expected error from failed launch, got nil")
	}
	if ps.firstStart.Load() {
		t.Fatal("firstStart must remain false after a failed Launch")
	}

	// Second request: Launch succeeds, ensureStarted returns nil and
	// firstStart flips to true. Without the fix, this call would have
	// replayed the cached firstStartErr from the prior sync.Once.
	if err := ps.ensureStarted(context.Background()); err != nil {
		t.Fatalf("second ensureStarted (succeeding launch): unexpected error: %v", err)
	}
	if !ps.firstStart.Load() {
		t.Fatal("firstStart must be true after a successful Launch")
	}
	if got := startCalls.Load(); got != 2 {
		t.Fatalf("Start mock invocations = %d, want 2 (one failed, one retried)", got)
	}

	// Third request: short-circuit — no further Start calls.
	if err := ps.ensureStarted(context.Background()); err != nil {
		t.Fatalf("third ensureStarted (already started): unexpected error: %v", err)
	}
	if got := startCalls.Load(); got != 2 {
		t.Fatalf("Start mock invocations after already-started = %d, want 2", got)
	}
}

// TestProcessSupervisor_FirstStart_NoRetryAfterSuccess pins the
// "happy path stays cached" half of the contract — once Launch has
// succeeded, ensureStarted is a pure no-op for every subsequent
// caller. Combined with the retry test above, this proves the cache
// is keyed on success rather than on "Launch was attempted at all"
// (the #1538 regression).
func TestProcessSupervisor_FirstStart_NoRetryAfterSuccess(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	var startCalls atomic.Int32
	process := &ProcessMock{
		StartMock: func(logger *slog.Logger) error {
			startCalls.Add(1)
			return nil
		},
	}

	ps := NewProcessSupervisor(logger, process, 0, 0, 1, 0).(*processSupervisor)

	for i := 0; i < 5; i++ {
		if err := ps.ensureStarted(context.Background()); err != nil {
			t.Fatalf("ensureStarted #%d: %v", i, err)
		}
	}
	if got := startCalls.Load(); got != 1 {
		t.Fatalf("Start invocations = %d, want 1 (cached after success)", got)
	}
}
