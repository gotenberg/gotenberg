package gotenberg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// GarbageCollect scans the root path and deletes files or directories with
// names containing specific substrings.
func GarbageCollect(logger *zap.Logger, rootPath string, includeSubstr []string) error {
	logger = logger.Named("gc")

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
			if strings.Contains(info.Name(), substr) || path == substr {
				err := os.RemoveAll(path)
				if err != nil {
					return fmt.Errorf("garbage collect '%s': %w", path, err)
				}

				logger.Debug(fmt.Sprintf("'%s' removed", path))

				return skipDirOrNil(info)
			}
		}

		return skipDirOrNil(info)
	})
}
