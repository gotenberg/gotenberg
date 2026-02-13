package gotenberg

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
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
	Start(logger *zap.Logger) error

	// Stop terminates the process and returns an error if the process cannot
	// be stopped.
	Stop(logger *zap.Logger) error

	// Healthy checks the health of the process. It returns true if the process
	// is healthy; otherwise, it returns false.
	Healthy(logger *zap.Logger) bool
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
	// If the process has not been started or is restarting, it is considered
	// healthy and true is returned. Otherwise, it returns the health status of
	// the actual process.
	Healthy() bool

	// Run executes a provided task while managing the state of the [Process].
	//
	// Run manages the request queue and may restart the process if it is not
	// healthy or if the number of handled requests exceeds the maximum limit.
	//
	// It returns an error if the task cannot be run or if the process state
	// cannot be managed properly.
	Run(ctx context.Context, logger *zap.Logger, task func() error) error

	// ReqQueueSize returns the current size of the request queue.
	ReqQueueSize() int64

	// RestartsCount returns the current number of restart.
	RestartsCount() int64
}

type processSupervisor struct {
	logger          *zap.Logger
	process         Process
	maxReqLimit     int64
	maxQueueSize    int64
	maxConcurrency  int64
	semaphore       chan struct{}
	firstStart      atomic.Bool
	firstStartOnce  sync.Once
	firstStartErr   error
	reqCounter      atomic.Int64
	reqQueueSize    atomic.Int64
	restartsCounter atomic.Int64
	isRestarting    atomic.Bool
	activeTasks     atomic.Int64
	restartMutex    sync.Mutex
}

// NewProcessSupervisor initializes a new [ProcessSupervisor].
func NewProcessSupervisor(logger *zap.Logger, process Process, maxReqLimit, maxQueueSize, maxConcurrency int64) ProcessSupervisor {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}

	b := &processSupervisor{
		logger:         logger,
		process:        process,
		semaphore:      make(chan struct{}, maxConcurrency),
		maxReqLimit:    maxReqLimit,
		maxQueueSize:   maxQueueSize,
		maxConcurrency: maxConcurrency,
	}
	b.reqCounter.Store(0)
	b.reqQueueSize.Store(0)
	b.restartsCounter.Store(0)
	b.isRestarting.Store(false)
	b.activeTasks.Store(0)

	return b
}

func (s *processSupervisor) Launch() error {
	s.logger.Debug("start process")
	err := s.process.Start(s.logger)
	if err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	s.firstStart.Store(true)
	s.logger.Debug("process successfully started")

	return nil
}

func (s *processSupervisor) Shutdown() error {
	s.logger.Debug("shutdown process")
	err := s.process.Stop(s.logger)
	if err != nil {
		return fmt.Errorf("shutdown process: %w", err)
	}

	s.logger.Debug("process successfully shutdown")

	return nil
}

func (s *processSupervisor) restart() error {
	s.logger.Debug("restart process")

	err := s.Shutdown()
	if err != nil {
		// No big deal? Chances are it's already stopped.
		s.logger.Debug(fmt.Sprintf("stop process before restart: %s", err))
	}

	err = s.Launch()
	if err != nil {
		return fmt.Errorf("restart process: %w", err)
	}

	s.reqCounter.Store(0)
	s.restartsCounter.Add(1)
	s.logger.Debug("process successfully restarted")

	return nil
}

func (s *processSupervisor) Healthy() bool {
	if !s.firstStart.Load() {
		// A non-started process is always healthy.
		return true
	}

	if s.isRestarting.Load() {
		// A restarting process is always healthy.
		return true
	}

	return s.process.Healthy(s.logger)
}

func (s *processSupervisor) Run(ctx context.Context, logger *zap.Logger, task func() error) error {
	// A user reported a potential issue:
	//
	// "Although the counting operation is atomic, nothing prevent 2 concurrent
	// goroutines to retrieve the same 'currentQueueSize' and to compare its
	// value against the max limit. Then, resulting queue size would be 1 above
	// the allowed limit."
	//
	// However, he was unable to actually trigger this issue, even when sending
	// a lot of requests.
	//
	// For now, the best option is to consider this issue to be unlikely to
	// happen, and keep the code as it is because it is more readable this way.
	//
	// See https://github.com/gotenberg/gotenberg/issues/951.
	currentQueueSize := s.reqQueueSize.Load()
	if s.maxQueueSize > 0 && currentQueueSize >= s.maxQueueSize {
		return ErrMaximumQueueSizeExceeded
	}

	s.reqQueueSize.Add(1)

	for {
		err := func() error {
			select {
			case s.semaphore <- struct{}{}:
				logger.Debug("process lock acquired")

				// If a restart drain is in progress, release the slot
				// immediately so the drain can acquire it instead.
				if s.isRestarting.Load() {
					<-s.semaphore
					return ErrProcessAlreadyRestarting
				}

				s.reqQueueSize.Add(-1)
				s.reqCounter.Add(1)
				s.activeTasks.Add(1)
				releaseSemaphore := true

				defer func() {
					s.activeTasks.Add(-1)
					if releaseSemaphore {
						logger.Debug("process lock released")
						<-s.semaphore
					}
				}()

				if !s.firstStart.Load() {
					s.firstStartOnce.Do(func() {
						s.firstStartErr = s.runWithDeadline(ctx, func() error {
							return s.Launch()
						})
					})
					if s.firstStartErr != nil {
						return fmt.Errorf("process first start: %w", s.firstStartErr)
					}
				}

				if !s.Healthy() {
					s.logger.Debug("process is unhealthy, cannot handle task, restarting...")
					err := s.doRestart(ctx)
					if err != nil {
						return fmt.Errorf("process restart before task: %w", err)
					}
				}

				err := s.runWithDeadline(ctx, task)

				if s.maxReqLimit > 0 && s.reqCounter.Load() >= s.maxReqLimit {
					// Only one goroutine should trigger the restart.
					if s.restartMutex.TryLock() {
						s.logger.Debug("max request limit reached, restarting eagerly...")
						releaseSemaphore = false

						go func() {
							restartErr := s.doRestartLocked(context.Background())
							s.restartMutex.Unlock()
							if restartErr != nil {
								s.logger.Error(fmt.Sprintf("process restart after task: %v", restartErr))
							}
							logger.Debug("process lock released")
							<-s.semaphore
						}()
					}
				}

				// Note: no error wrapping because it leaks on Chromium console exceptions output.
				return err
			case <-ctx.Done():
				logger.Debug("failed to acquire process lock before deadline")
				s.reqQueueSize.Add(-1)

				return fmt.Errorf("acquire process lock: %w", ctx.Err())
			}
		}()

		if errors.Is(err, ErrProcessAlreadyRestarting) {
			logger.Debug("process is already restarting, trying to acquire process lock again...")
			continue
		}

		// Note: no error wrapping because it leaks on Chromium console exceptions output.
		return err
	}
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

// Interface guards.
var (
	_ ProcessSupervisor = (*processSupervisor)(nil)
)
