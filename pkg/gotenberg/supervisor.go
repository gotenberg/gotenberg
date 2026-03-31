package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// ErrProcessAlreadyRestarting happens if the [ProcessSupervisor] is trying
// to restart an already restarting [Process].
var ErrProcessAlreadyRestarting = errors.New("process already restarting")

// ErrMaximumQueueSizeExceeded happens if Run() is called but the maximum queue
// size is already used.
var ErrMaximumQueueSizeExceeded = errors.New("maximum queue size exceeded")

// Process is an interface that represents an abstract process
// and provides methods for starting, stopping, and checking the health of the
// process.
//
// Implementations of this interface should handle the actual logic for
// starting, stopping, and ensuring the process's health.
type Process interface {
	// Start initiates the process and returns an error if the process cannot
	// be started.
	Start(logger *slog.Logger) error

	// Stop terminates the process and returns an error if the process cannot
	// be stopped.
	Stop(logger *slog.Logger) error

	// Healthy checks the health of the process. It returns true if the process
	// is healthy; otherwise, it returns false.
	Healthy(logger *slog.Logger) bool
}

// ProcessSupervisor provides methods to manage a [Process], including
// starting, stopping, and ensuring its health.
//
// Additionally, it allows for the execution of tasks while managing the
// process's state and provides functionality for limiting the number of
// requests that can be handled by the process, as well as managing a request
// queue.
type ProcessSupervisor interface {
	// Launch starts the managed [Process].
	Launch() error

	// Shutdown stops the managed [Process].
	Shutdown() error

	// Healthy checks and returns the health status of the managed [Process].
	//
	// A non-started process is considered healthy (startup is deferred until
	// the first request). Returns false if the process is currently restarting
	// or is reported unhealthy by the underlying [Process].
	Healthy() bool

	// Run executes a provided task while managing the state of the [Process].
	//
	// Run manages the request queue and may restart the process if it is not
	// healthy or if the number of handled requests exceeds the maximum limit.
	//
	// It returns an error if the task cannot be run or if the process state
	// cannot be managed properly.
	Run(ctx context.Context, logger *slog.Logger, task func() error) error

	// ReqQueueSize returns the current size of the request queue.
	ReqQueueSize() int64

	// RestartsCount returns the current number of restart.
	RestartsCount() int64

	// ActiveTasksCount returns the current number of active tasks.
	ActiveTasksCount() int64
}

type processSupervisor struct {
	logger         *slog.Logger
	process        Process
	maxReqLimit    int64
	maxQueueSize   int64
	maxConcurrency int64
	semaphore      chan struct{}
	firstStart     atomic.Bool
	firstStartOnce sync.Once
	// firstStartErr stores the error from the first Launch attempt executed
	// via firstStartOnce. Subsequent callers that enter the !firstStart block
	// need to observe this value after the Once has completed, without
	// re-executing the closure.
	firstStartErr       error
	reqCounter          atomic.Int64
	reqQueueSize        atomic.Int64
	restartsCounter     atomic.Int64
	isRestarting        atomic.Bool
	activeTasks         atomic.Int64
	restartMutex        sync.Mutex
	idleShutdownTimeout time.Duration
	lastActivity        atomic.Int64  // unix nano timestamp of last completed task
	idleMu              sync.Mutex    // protects idleStopChan
	idleStopChan        chan struct{} // signal to stop the idle ticker goroutine
}

// NewProcessSupervisor initializes a new [ProcessSupervisor].
func NewProcessSupervisor(logger *slog.Logger, process Process, maxReqLimit, maxQueueSize, maxConcurrency int64, idleShutdownTimeout time.Duration) ProcessSupervisor {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}

	b := &processSupervisor{
		logger:              logger,
		process:             process,
		semaphore:           make(chan struct{}, maxConcurrency),
		maxReqLimit:         maxReqLimit,
		maxQueueSize:        maxQueueSize,
		maxConcurrency:      maxConcurrency,
		idleShutdownTimeout: idleShutdownTimeout,
	}
	b.reqCounter.Store(0)
	b.reqQueueSize.Store(0)
	b.restartsCounter.Store(0)
	b.isRestarting.Store(false)
	b.activeTasks.Store(0)

	return b
}

func (s *processSupervisor) Launch() error {
	s.logger.DebugContext(context.Background(), "start process")
	err := s.process.Start(s.logger)
	if err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	s.firstStart.Store(true)

	if s.idleShutdownTimeout > 0 {
		s.lastActivity.Store(time.Now().UnixNano())
		s.startIdleTicker()
	}

	s.logger.DebugContext(context.Background(), "process successfully started")

	return nil
}

func (s *processSupervisor) Shutdown() error {
	s.logger.DebugContext(context.Background(), "shutdown process")

	s.stopIdleTicker()

	err := s.process.Stop(s.logger)
	if err != nil {
		return fmt.Errorf("shutdown process: %w", err)
	}

	s.logger.DebugContext(context.Background(), "process successfully shutdown")

	return nil
}

func (s *processSupervisor) restart() error {
	s.logger.DebugContext(context.Background(), "restart process")

	err := s.Shutdown()
	if err != nil {
		// Not necessarily critical — chances are the process is already stopped,
		// but worth flagging in case it indicates a real issue.
		s.logger.WarnContext(context.Background(), fmt.Sprintf("stop process before restart: %s", err))
	}

	err = s.Launch()
	if err != nil {
		return fmt.Errorf("restart process: %w", err)
	}

	s.reqCounter.Store(0)
	s.restartsCounter.Add(1)
	s.logger.DebugContext(context.Background(), "process successfully restarted")

	return nil
}

func (s *processSupervisor) Healthy() bool {
	if !s.firstStart.Load() {
		// A non-started process is considered healthy: Gotenberg defers
		// process startup until the first request to keep resource usage low.
		// Reporting unhealthy here would cause container orchestrators to
		// restart the pod before any request arrives.
		return true
	}

	if s.isRestarting.Load() {
		// A restarting process is not yet healthy — this gives load balancers
		// honest information so they can avoid routing traffic to this node.
		return false
	}

	return s.process.Healthy(s.logger)
}

func (s *processSupervisor) Run(ctx context.Context, logger *slog.Logger, task func() error) error {
	// Atomically check and increment the queue size to avoid the TOCTOU race
	// originally reported in https://github.com/gotenberg/gotenberg/issues/951.
	for {
		current := s.reqQueueSize.Load()
		if s.maxQueueSize > 0 && current >= s.maxQueueSize {
			return ErrMaximumQueueSizeExceeded
		}
		if s.reqQueueSize.CompareAndSwap(current, current+1) {
			break
		}
	}

	// Decrement when Run() returns, regardless of which path is taken
	// (context timeout, task completion, error, etc.). This ensures the
	// request is counted as "in the queue" for the entire duration of Run(),
	// preventing new requests from entering while one is being processed.
	// See https://github.com/gotenberg/gotenberg/issues/1502.
	defer s.reqQueueSize.Add(-1)

	for {
		err := func() error {
			if err := s.acquireSlot(ctx, logger); err != nil {
				return err
			}

			s.reqCounter.Add(1)
			s.activeTasks.Add(1)
			semaphoreOwned := true

			defer func() {
				s.activeTasks.Add(-1)
				if s.idleShutdownTimeout > 0 {
					s.lastActivity.Store(time.Now().UnixNano())
				}
				if semaphoreOwned {
					logger.DebugContext(ctx, "process lock released")
					<-s.semaphore
				}
			}()

			if err := s.ensureStarted(ctx); err != nil {
				return err
			}

			if err := s.ensureHealthy(ctx); err != nil {
				return err
			}

			err := s.runWithDeadline(ctx, task)

			if s.maybeRestartAfterTask(logger) {
				semaphoreOwned = false
			}

			// Note: no error wrapping because it leaks on Chromium console exceptions output.
			return err
		}()

		if errors.Is(err, ErrProcessAlreadyRestarting) {
			logger.DebugContext(ctx, "process is already restarting, trying to acquire process lock again...")
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// Note: no error wrapping because it leaks on Chromium console exceptions output.
		return err
	}
}

// startIdleTicker starts a background goroutine that periodically checks
// whether the process has been idle long enough to shut down.
func (s *processSupervisor) startIdleTicker() {
	stopChan := make(chan struct{})

	s.idleMu.Lock()
	s.idleStopChan = stopChan
	s.idleMu.Unlock()

	go func() {
		ticker := time.NewTicker(s.idleShutdownTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.maybeIdleShutdown()
			case <-stopChan:
				return
			}
		}
	}()
}

// stopIdleTicker signals the idle ticker goroutine to exit, if one is running.
func (s *processSupervisor) stopIdleTicker() {
	s.idleMu.Lock()
	defer s.idleMu.Unlock()

	if s.idleStopChan != nil {
		close(s.idleStopChan)
		s.idleStopChan = nil
	}
}

// maybeIdleShutdown stops the process if it has been idle for longer than
// the configured timeout. It is safe to call concurrently with Run and
// restart.
func (s *processSupervisor) maybeIdleShutdown() {
	if !s.firstStart.Load() || s.isRestarting.Load() {
		return
	}

	if s.activeTasks.Load() > 0 || s.reqQueueSize.Load() > 0 {
		return
	}

	lastNano := s.lastActivity.Load()
	if lastNano == 0 || time.Since(time.Unix(0, lastNano)) < s.idleShutdownTimeout {
		return
	}

	if !s.restartMutex.TryLock() {
		return
	}
	defer s.restartMutex.Unlock()

	// Double-check after acquiring the lock.
	if s.activeTasks.Load() > 0 || s.reqQueueSize.Load() > 0 {
		return
	}

	s.logger.DebugContext(context.Background(), "idle shutdown timeout reached, stopping process")

	// Stop the ticker — it will be restarted on the next Launch().
	s.stopIdleTicker()

	err := s.process.Stop(s.logger)
	if err != nil {
		s.logger.WarnContext(context.Background(), fmt.Sprintf("idle shutdown: %s", err))
		return
	}

	// Reset state so ensureStarted() re-launches on next request.
	s.firstStart.Store(false)
	s.firstStartOnce = sync.Once{}
	s.firstStartErr = nil
	s.reqCounter.Store(0)

	s.logger.DebugContext(context.Background(), "process stopped due to idle timeout")
}

// acquireSlot attempts to acquire a semaphore slot, yielding it back if a
// restart drain is in progress.
func (s *processSupervisor) acquireSlot(ctx context.Context, logger *slog.Logger) error {
	select {
	case s.semaphore <- struct{}{}:
		// If a restart drain is in progress, release the slot
		// immediately so the drain can acquire it instead.
		if s.isRestarting.Load() {
			<-s.semaphore
			return ErrProcessAlreadyRestarting
		}

		logger.DebugContext(ctx, "process lock acquired")

		return nil
	case <-ctx.Done():
		logger.DebugContext(ctx, "failed to acquire process lock before deadline")

		return fmt.Errorf("acquire process lock: %w", ctx.Err())
	}
}

// ensureStarted performs a one-time lazy launch of the process on its first
// use. Subsequent calls are no-ops.
func (s *processSupervisor) ensureStarted(ctx context.Context) error {
	if s.firstStart.Load() {
		return nil
	}

	s.firstStartOnce.Do(func() {
		s.firstStartErr = s.runWithDeadline(ctx, func() error {
			return s.Launch()
		})
	})

	if s.firstStartErr != nil {
		return fmt.Errorf("process first start: %w", s.firstStartErr)
	}

	return nil
}

// ensureHealthy checks the underlying process health and triggers a
// synchronous restart if the process is unhealthy. Skips the check if a
// restart is already in progress.
func (s *processSupervisor) ensureHealthy(ctx context.Context) error {
	if s.isRestarting.Load() || s.process.Healthy(s.logger) {
		return nil
	}

	s.logger.DebugContext(context.Background(), "process is unhealthy, cannot handle task, restarting...")

	if err := s.doRestart(ctx); err != nil {
		return fmt.Errorf("process restart before task: %w", err)
	}

	return nil
}

// maybeRestartAfterTask checks if the maximum request limit has been reached
// and, if so, triggers an asynchronous restart. If a restart is initiated, it
// takes ownership of the caller's semaphore slot (the caller must not release
// it). Returns true if ownership was taken.
func (s *processSupervisor) maybeRestartAfterTask(logger *slog.Logger) bool {
	if s.maxReqLimit <= 0 || s.reqCounter.Load() < s.maxReqLimit {
		return false
	}

	if !s.restartMutex.TryLock() {
		return false
	}

	s.logger.DebugContext(context.Background(), "max request limit reached, restarting eagerly...")

	go func() {
		restartErr := s.doRestartLocked(context.Background())
		s.restartMutex.Unlock()
		if restartErr != nil {
			s.logger.ErrorContext(context.Background(), fmt.Sprintf("process restart after task: %v", restartErr))
		}
		logger.DebugContext(context.Background(), "process lock released")
		<-s.semaphore
	}()

	return true
}

// doRestart coordinates a process restart, draining all active concurrent
// tasks before stopping and restarting the process.
func (s *processSupervisor) doRestart(ctx context.Context) error {
	s.restartMutex.Lock()
	defer s.restartMutex.Unlock()

	return s.doRestartLocked(ctx)
}

// doRestartLocked performs the restart drain logic. The caller must hold restartMutex.
func (s *processSupervisor) doRestartLocked(ctx context.Context) error {
	s.isRestarting.Store(true)
	defer s.isRestarting.Store(false)

	// Drain all other active semaphore slots so no other tasks are running during the restart.
	slotsToAcquire := s.maxConcurrency - 1
	acquired := make([]struct{}, 0, slotsToAcquire)

	for range slotsToAcquire {
		select {
		case s.semaphore <- struct{}{}:
			acquired = append(acquired, struct{}{})
		case <-ctx.Done():
			for range acquired {
				<-s.semaphore
			}
			return fmt.Errorf("drain active tasks before restart: %w", ctx.Err())
		}
	}

	err := s.runWithDeadline(ctx, func() error {
		return s.restart()
	})

	for range acquired {
		<-s.semaphore
	}

	return err
}

func (s *processSupervisor) runWithDeadline(ctx context.Context, task func() error) error {
	runChan := make(chan error, 1)
	go func() {
		runChan <- task()
	}()

	for {
		select {
		case err := <-runChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *processSupervisor) ReqQueueSize() int64 {
	return s.reqQueueSize.Load()
}

func (s *processSupervisor) RestartsCount() int64 {
	return s.restartsCounter.Load()
}

func (s *processSupervisor) ActiveTasksCount() int64 {
	return s.activeTasks.Load()
}

// Interface guards.
var (
	_ ProcessSupervisor = (*processSupervisor)(nil)
)
