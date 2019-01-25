package printer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

var mu sync.Mutex

// Office facilitates Office documents to PDF conversion.
type Office struct {
	Context   context.Context
	FilePaths []string
	Landscape bool
}

// Print converts Office documents to PDF.
func (o *Office) Print(destination string) error {
	mu.Lock()
	defer mu.Unlock()
	fpaths := make([]string, len(o.FilePaths))
	dirPath := filepath.Dir(destination)
	for i, fpath := range o.FilePaths {
		baseFilename, err := rand.Get()
		if err != nil {
			return err
		}
		tmpDest := fmt.Sprintf("%s/%s.pdf", dirPath, baseFilename)
		cmdArgs := []string{
			"--format",
			"pdf",
		}
		if o.Landscape {
			cmdArgs = append(cmdArgs, "--printer", "PaperOrientation=landscape")
		}
		cmdArgs = append(cmdArgs, "--output", tmpDest, fpath)
		cmd := exec.CommandContext(
			o.Context,
			"unoconv",
			cmdArgs...,
		)
		_, err = cmd.Output()
		if o.Context.Err() == context.DeadlineExceeded {
			return errors.New("unoconv: command timed out")
		}
		if err != nil {
			return fmt.Errorf("unoconv: non-zero exit code: %v", err)
		}
		fpaths[i] = tmpDest
	}
	if len(fpaths) == 1 {
		return os.Rename(fpaths[0], destination)
	}
	return Merge(fpaths, destination)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(Office))
)
