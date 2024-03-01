package gotenberg

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
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
			logger := zap.NewNop()

			process := &ProcessMock{
				StartMock: func(logger *zap.Logger) error {
					return tc.startError
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0).(*processSupervisor)
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
			logger := zap.NewNop()

			process := &ProcessMock{
				StopMock: func(logger *zap.Logger) error {
					return tc.stopError
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0)
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
		scenario            string
		initiallyRestarting bool
		startError          error
		stopError           error
		expectError         bool
		expectedError       error
	}{
		{
			scenario:            "already restarting",
			initiallyRestarting: true,
			expectError:         true,
			expectedError:       ErrProcessAlreadyRestarting,
		},
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
			logger := zap.NewNop()

			process := &ProcessMock{
				StartMock: func(logger *zap.Logger) error {
					return tc.startError
				},
				StopMock: func(logger *zap.Logger) error {
					return tc.stopError
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0).(*processSupervisor)
			if tc.initiallyRestarting {
				ps.isRestarting.Store(true)
			}

			err := ps.restart()

			if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if tc.expectedError != nil && !errors.Is(err, tc.expectedError) {
				t.Fatalf("expected error %v but got: %v", tc.expectedError, err)
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
			scenario:         "non-started process is always healthy",
			initiallyStarted: false,
			expectHealthy:    true,
		},
		{
			scenario:            "restarting process is always healthy",
			initiallyStarted:    true,
			initiallyRestarting: true,
			expectHealthy:       true,
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
			logger := zap.NewNop()

			process := &ProcessMock{
				HealthyMock: func(logger *zap.Logger) bool {
					return tc.processHealthy
				},
			}

			ps := NewProcessSupervisor(logger, process, 5, 0).(*processSupervisor)
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
			scenario:             "cannot restart after reaching max request limit",
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
			logger := zap.NewNop()

			var startCalls, healthyCalls, stopCalls atomic.Int64
			startCalls.Store(0)
			healthyCalls.Store(0)
			stopCalls.Store(0)

			process := &ProcessMock{
				StartMock: func(logger *zap.Logger) error {
					startCalls.Add(1)
					return tc.startError
				},
				StopMock: func(logger *zap.Logger) error {
					stopCalls.Add(1)
					return nil
				},
				HealthyMock: func(logger *zap.Logger) bool {
					healthyCalls.Add(1)
					return tc.processHealthy
				},
			}

			ps := NewProcessSupervisor(logger, process, tc.maxReqLimit, tc.maxQueueSize).(*processSupervisor)
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
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := ps.Run(ctx, logger, task)
					if err != nil {
						errorChan <- err
					}
				}()
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
			ps := NewProcessSupervisor(zap.NewNop(), new(ProcessMock), 0, 0).(*processSupervisor)

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
	logger := zap.NewNop()
	process := &ProcessMock{
		StartMock: func(logger *zap.Logger) error {
			return nil
		},
		HealthyMock: func(logger *zap.Logger) bool {
			return true
		},
	}
	ps := NewProcessSupervisor(logger, process, 0, 0).(*processSupervisor)

	// Simulating a lock.
	ps.mutexChan <- struct{}{}

	if ps.ReqQueueSize() != 0 {
		t.Fatalf("expected queue size to be 0 but got %d", ps.ReqQueueSize())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	errorChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ps.Run(ctx, logger, func() error {
				return nil
			})
			if err != nil {
				errorChan <- err
			}
		}()
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
			logger := zap.NewNop()

			process := &ProcessMock{
				StartMock: func(logger *zap.Logger) error {
					return tc.startError
				},
				StopMock: func(logger *zap.Logger) error {
					return tc.stopError
				},
			}

			ps := NewProcessSupervisor(logger, process, 0, 0).(*processSupervisor)
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
