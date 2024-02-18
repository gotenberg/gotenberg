package gotenberg

import (
	"context"
	"errors"
	"fmt"
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
	mutexChan       chan struct{}
	firstStart      atomic.Bool
	reqCounter      atomic.Int64
	reqQueueSize    atomic.Int64
	restartsCounter atomic.Int64
	isRestarting    atomic.Bool
}

// NewProcessSupervisor initializes a new [ProcessSupervisor].
func NewProcessSupervisor(logger *zap.Logger, process Process, maxReqLimit, maxQueueSize int64) ProcessSupervisor {
	b := &processSupervisor{
		logger:       logger,
		process:      process,
		mutexChan:    make(chan struct{}, 1),
		maxReqLimit:  maxReqLimit,
		maxQueueSize: maxQueueSize,
	}
	b.reqCounter.Store(0)
	b.reqQueueSize.Store(0)
	b.restartsCounter.Store(0)
	b.isRestarting.Store(false)

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
	if s.isRestarting.Load() {
		s.logger.Debug("process already restarting, skip restart")

		return ErrProcessAlreadyRestarting
	}

	s.logger.Debug("restart process")
	s.isRestarting.Store(true)
	defer s.isRestarting.Store(false)

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
	currentQueueSize := s.reqQueueSize.Load()
	if s.maxQueueSize > 0 && currentQueueSize >= s.maxQueueSize {
		return ErrMaximumQueueSizeExceeded
	}

	s.reqQueueSize.Add(1)

	for {
		err := func() error {
			select {
			case s.mutexChan <- struct{}{}:
				logger.Debug("process lock acquired")
				s.reqQueueSize.Add(-1)
				s.reqCounter.Add(1)

				defer func() {
					logger.Debug("process lock released")
					<-s.mutexChan
				}()

				if !s.firstStart.Load() {
					err := s.runWithDeadline(ctx, func() error {
						return s.Launch()
					})
					if err != nil {
						return fmt.Errorf("process first start: %w", err)
					}
				}

				if !s.Healthy() {
					s.logger.Debug("process is unhealthy, cannot handle task, restarting...")
					err := s.runWithDeadline(ctx, func() error {
						return s.restart()
					})
					if err != nil {
						return fmt.Errorf("process restart before task: %w", err)
					}
				}

				if s.maxReqLimit > 0 && s.reqCounter.Load() >= s.maxReqLimit {
					s.logger.Debug("max request limit reached, restarting...")
					err := s.runWithDeadline(ctx, func() error {
						return s.restart()
					})
					if err != nil {
						return fmt.Errorf("process restart before task: %w", err)
					}
				}

				// Note: no error wrapping because it leaks on Chromium console exceptions output.
				return s.runWithDeadline(ctx, task)
			case <-ctx.Done():
				logger.Debug("failed to acquire process lock before deadline")
				s.reqQueueSize.Add(-1)

				return fmt.Errorf("acquire process lock: %w", ctx.Err())
			}
		}()

		if errors.Is(err, ErrProcessAlreadyRestarting) {
			logger.Debug("process is already restarting, trying to acquire process lock again...")
			s.reqQueueSize.Add(1)
			continue
		}

		// Note: no error wrapping because it leaks on Chromium console exceptions output.
		return err
	}
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
