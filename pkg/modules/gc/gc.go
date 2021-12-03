package gc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(GarbageCollector{})
}

// GarbageCollector is a module for removing files and directories that have
// expired. It allows us to make sure that the application does not leak files
// or directories when running.
type GarbageCollector struct {
	rootPath      string
	graceDuration time.Duration
	excludeSubstr []string

	ticker *time.Ticker
	done   chan bool

	logger *zap.Logger
}

// GarbageCollectorGraceDurationModifier is a module interface which allows to
// update the expiration time of files and directories parsed by the garbage
// collector. For instance, if the grace duration is 30s, the garbage collector
// will remove paths that have a modification time older than 30s. If there are
// many GarbageCollectorGraceDurationModifier, only the longest grace duration
// is selected.
type GarbageCollectorGraceDurationModifier interface {
	GraceDuration() time.Duration
}

// GarbageCollectorExcludeSubstrModifier is a module interface which adds the
// given substrings to the exclude list of the garbage collector. If a path
// contains one of those substrings, the garbage collector ignores it.
type GarbageCollectorExcludeSubstrModifier interface {
	ExcludeSubstr() []string
}

// Descriptor returns a GarbageCollector's module descriptor.
func (gc GarbageCollector) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "gc",
		New: func() gotenberg.Module { return new(GarbageCollector) },
	}
}

// Provision sets the module properties.
func (gc *GarbageCollector) Provision(ctx *gotenberg.Context) error {
	gc.rootPath = gotenberg.TmpPath()

	graceDurationModifiers, err := ctx.Modules(new(GarbageCollectorGraceDurationModifier))
	if err != nil {
		return fmt.Errorf("get grace duration modifiers: %w", err)
	}

	for _, graceDurationModifier := range graceDurationModifiers {
		modifier := graceDurationModifier.(GarbageCollectorGraceDurationModifier)

		if gc.graceDuration < modifier.GraceDuration() {
			gc.graceDuration = modifier.GraceDuration()
		}
	}

	excludeSubstrModifiers, err := ctx.Modules(new(GarbageCollectorExcludeSubstrModifier))
	if err != nil {
		return fmt.Errorf("get exclude substr modifiers: %w", err)
	}

	gc.excludeSubstr = strings.Split(os.Getenv("GC_EXCLUDE_SUBSTR"), ",")

	for _, excludeSubstrModifier := range excludeSubstrModifiers {
		modifier := excludeSubstrModifier.(GarbageCollectorExcludeSubstrModifier)

		gc.excludeSubstr = append(gc.excludeSubstr, modifier.ExcludeSubstr()...)
	}

	loggerProvider, err := ctx.Module(new(gotenberg.LoggerProvider))
	if err != nil {
		return fmt.Errorf("get logger provider: %w", err)
	}

	logger, err := loggerProvider.(gotenberg.LoggerProvider).Logger(gc)
	if err != nil {
		return fmt.Errorf("get logger: %w", err)
	}

	gc.logger = logger

	return nil
}

// Start starts the garbage collector.
func (gc *GarbageCollector) Start() error {
	gc.ticker = time.NewTicker(gc.graceDuration + time.Duration(1)*time.Second)
	gc.done = make(chan bool, 1)

	go func() {
		for {
			func() {
				gcMu.RLock()
				defer gcMu.RUnlock()

				select {
				case <-gc.done:
					return
				case <-gc.ticker.C:
					gc.collect(false)
				}
			}()
		}
	}()

	return nil
}

// collect parses the root path of the garbage collector and removes files or
// directories that have expired. It ignores the expiration date if the "force"
// argument is set to true.
func (gc GarbageCollector) collect(force bool) {
	expirationTime := time.Now().Add(-gc.graceDuration)

	// To make sure that the next Walk method stays on
	// the root level of the considered path, we have to
	// return a filepath.SkipDir error if the current path
	// is a directory.
	skipDirOrNil := func(info os.FileInfo) error {
		if info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}

	removePath := func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			gc.logger.Error(fmt.Sprintf("remove '%s': %s", path, err))
		}

		gc.logger.Debug(fmt.Sprintf("'%s' removed", path))
	}

	err := filepath.Walk(gc.rootPath, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			// For whatever reasons, the Walk method failed
			// to process the current path.
			return pathErr
		}

		if path == gc.rootPath {
			return nil
		}

		for _, substr := range gc.excludeSubstr {
			if strings.Contains(info.Name(), substr) {
				return skipDirOrNil(info)
			}
		}

		if force {
			removePath(path)

			return skipDirOrNil(info)
		}

		if info.ModTime().Before(expirationTime) {
			removePath(path)
		}

		return skipDirOrNil(info)
	})

	if err != nil {
		gc.logger.Error(err.Error())
	}
}

// StartupMessage returns an empty string.
func (gc GarbageCollector) StartupMessage() string {
	return ""
}

// Stop stops the garbage collector.
func (gc *GarbageCollector) Stop(ctx context.Context) error {
	_, ok := ctx.Deadline()
	if !ok {
		return errors.New("no context dead line")
	}

	// Block until the context is done so that other module may gracefully stop
	// before we do a shutdown cleanup.
	gc.logger.Debug("wait for the end of grace duration")

	<-ctx.Done()

	gc.ticker.Stop()
	gc.done <- true

	gc.logger.Debug("shutdown cleanup...")
	gc.collect(true)

	return nil
}

var gcMu sync.RWMutex

// Interface guards.
var (
	_ gotenberg.Module      = (*GarbageCollector)(nil)
	_ gotenberg.Provisioner = (*GarbageCollector)(nil)
	_ gotenberg.App         = (*GarbageCollector)(nil)
)
