package uno

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

type listener interface {
	start(logger *zap.Logger) error
	stop(logger *zap.Logger) error
	lock(ctx context.Context, logger *zap.Logger) error
	unlock(logger *zap.Logger) error
	port() int
	queue() int
	healthy() bool
}

type libreOfficeListener struct {
	binPath      string
	startTimeout time.Duration
	threshold    int

	socketPort         int
	userProfileDirPath string
	cmd                gotenberg.Cmd
	cfgMu              sync.RWMutex

	usage         int
	queueLength   int
	queueLengthMu sync.RWMutex
	lockChan      chan struct{}
	logger        *zap.Logger
}

func newLibreOfficeListener(logger *zap.Logger, binPath string, startTimeout time.Duration, threshold int) listener {
	return &libreOfficeListener{
		binPath:      binPath,
		startTimeout: startTimeout,
		threshold:    threshold,
		lockChan:     make(chan struct{}, 1),
		logger:       logger.Named("listener"),
	}
}

func (listener *libreOfficeListener) start(logger *zap.Logger) error {
	port, err := freePort(logger)
	if err != nil {
		return fmt.Errorf("get free port: %w", err)
	}

	// Good to know: when the supervisor manages the LibreOffice listener,
	// the garbage collector might delete the next directory while it is
	// still running. It does seem to cause any issue though.
	userProfileDirPath := gotenberg.NewDirPath()

	args := []string{
		"--headless",
		"--invisible",
		"--nocrashreport",
		"--nodefault",
		"--nologo",
		"--nofirststartwizard",
		"--norestore",
		fmt.Sprintf("-env:UserInstallation=file://%s", userProfileDirPath),
		fmt.Sprintf("--accept=socket,host=127.0.0.1,port=%d,tcpNoDelay=1;urp;StarOffice.ComponentContext", port),
	}

	ctx, cancel := context.WithTimeout(context.Background(), listener.startTimeout)
	defer cancel()

	cmd, err := gotenberg.CommandContext(ctx, logger, listener.binPath, args...)
	if err != nil {
		return fmt.Errorf("create LibreOffice listener command: %w", err)
	}

	// For whatever reason, LibreOffice requires a first start before being
	// able to run as a daemon.
	exitCode, err := cmd.Exec()
	if err != nil && exitCode != 81 {
		return fmt.Errorf("execute LibreOffice listener: %w", err)
	}

	logger.Debug("got exit code 81, e.g., LibreOffice listener first start")

	// Second start (daemon).
	cmd = gotenberg.Command(logger, listener.binPath, args...)

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start LibreOffice listener: %w", err)
	}

	// As the LibreOffice socket may take some time to be available, we have to
	// ensure that it is indeed accepting connections.
	logger.Debug("waiting for the LibreOffice listener socket to be available...")

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("waiting for the LibreOffice listener socket to be available: %w", ctx.Err())
		}

		_, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Duration(1)*time.Second)
		if err == nil {
			break
		}
	}

	logger.Debug("LibreOffice listener socket available")

	listener.cfgMu.Lock()
	listener.socketPort = port
	listener.userProfileDirPath = userProfileDirPath
	listener.cmd = cmd
	listener.cfgMu.Unlock()

	return nil
}

func (listener *libreOfficeListener) stop(logger *zap.Logger) error {
	listener.cfgMu.RLock()

	defer func() {
		defer listener.cfgMu.RUnlock()

		err := os.RemoveAll(listener.userProfileDirPath)
		if err != nil {
			logger.Error(fmt.Sprintf("remove LibreOffice listener user profile directory: %v", err))
		}
	}()

	err := listener.cmd.Kill()
	if err != nil {
		return fmt.Errorf("kill LibreOffice listener process: %w", err)
	}

	// Let's wait to make sure the process is no more.
	err = listener.cmd.Wait()
	if err != nil {
		logger.Debug(fmt.Sprintf("wait for the LibreOffice listener: %v", err))
	}

	return nil
}

func (listener *libreOfficeListener) lock(ctx context.Context, logger *zap.Logger) error {
	listener.queueLengthMu.Lock()
	listener.queueLength += 1
	listener.queueLengthMu.Unlock()

	select {
	case listener.lockChan <- struct{}{}:
		logger.Debug("LibreOffice listener lock acquired")

		listener.queueLengthMu.Lock()
		listener.queueLength -= 1
		listener.queueLengthMu.Unlock()

		return nil
	case <-ctx.Done():
		logger.Debug("failed to acquire LibreOffice listener lock before deadline")

		listener.queueLengthMu.Lock()
		listener.queueLength -= 1
		listener.queueLengthMu.Unlock()

		return fmt.Errorf("acquire LibreOffice listener lock: %w", ctx.Err())
	}
}

func (listener *libreOfficeListener) unlock(logger *zap.Logger) error {
	restart := func() error {
		err := listener.stop(logger)
		if err != nil {
			return fmt.Errorf("stop LibreOffice listener: %w", err)
		}

		err = listener.start(logger)
		if err != nil {
			return fmt.Errorf("start LibreOffice listener: %w", err)
		}

		listener.usage = 0

		return nil
	}

	defer func() {
		<-listener.lockChan
		logger.Debug("LibreOffice listener lock released")
	}()

	if !listener.healthy() {
		logger.Debug("LibreOffice listener is unhealthy, restarting it...")

		err := restart()
		if err == nil {
			return nil
		}

		return fmt.Errorf("restart LibreOffice listener: %w", err)
	}

	listener.usage += 1
	if listener.usage < listener.threshold {
		return nil
	}

	logger.Debug("LibreOffice listener threshold reached, restarting it...")

	err := restart()
	if err == nil {
		return nil
	}

	return fmt.Errorf("restart LibreOffice listener: %w", err)
}

func (listener *libreOfficeListener) port() int {
	listener.cfgMu.RLock()
	defer listener.cfgMu.RUnlock()

	return listener.socketPort
}

func (listener *libreOfficeListener) queue() int {
	listener.queueLengthMu.RLock()
	defer listener.queueLengthMu.RUnlock()

	return listener.queueLength
}

func (listener *libreOfficeListener) healthy() bool {
	_, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", listener.port()), time.Duration(1)*time.Second)

	return err == nil
}

// Interface guards.
var (
	_ listener = (*libreOfficeListener)(nil)
)
