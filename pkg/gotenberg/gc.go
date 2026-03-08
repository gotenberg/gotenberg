package gotenberg

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GarbageCollect scans the root path and deletes files or directories with
// names containing specific substrings and before a given expiration time.
func GarbageCollect(ctx context.Context, logger *slog.Logger, rootPath string, includeSubstr []string, expirationTime time.Time) error {
	logger = logger.With(slog.String("logger", "gc"))

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

	return filepath.Walk(rootPath, func(path string, info os.FileInfo, pathErr error) error {
		if pathErr != nil {
			return pathErr
		}

		if path == rootPath {
			return nil
		}

		for _, substr := range includeSubstr {
			if (strings.Contains(info.Name(), substr) || path == substr) && info.ModTime().Before(expirationTime) {
				err := os.RemoveAll(path)
				if err != nil {
					return fmt.Errorf("garbage collect '%s': %w", path, err)
				}

				logger.DebugContext(ctx, fmt.Sprintf("'%s' removed", path))

				return skipDirOrNil(info)
			}
		}

		return skipDirOrNil(info)
	})
}
