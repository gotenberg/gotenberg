package gotenberg

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// GarbageCollect deletes the entries directly under rootPath whose name
// matches one of the [filepath.Match] patterns and whose modification time is
// before expirationTime. It does not descend into directories.
func GarbageCollect(ctx context.Context, logger *slog.Logger, rootPath string, patterns []string, expirationTime time.Time) error {
	logger = logger.With(slog.String("logger", "gc"))

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return fmt.Errorf("read '%s': %w", rootPath, err)
	}

	for _, entry := range entries {
		if !matchesAny(entry.Name(), patterns) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			// The entry may have been removed since the directory was read.
			continue
		}

		if !info.ModTime().Before(expirationTime) {
			continue
		}

		path := filepath.Join(rootPath, entry.Name())

		err = os.RemoveAll(path) //nolint:gosec // G122: rootPath is a trusted internal working directory
		if err != nil {
			return fmt.Errorf("garbage collect '%s': %w", path, err)
		}

		logger.DebugContext(ctx, fmt.Sprintf("'%s' removed", path))
	}

	return nil
}

// matchesAny reports whether name matches one of the patterns.
func matchesAny(name string, patterns []string) bool {
	for _, pattern := range patterns {
		ok, err := filepath.Match(pattern, name)
		if err == nil && ok {
			return true
		}
	}

	return false
}
