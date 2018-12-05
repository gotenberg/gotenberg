package printer

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

var mu sync.Mutex

// Office facilitates Office document to PDF conversion.
type Office struct {
	Context   context.Context
	FilePaths []string
}

// Print converts Office document to PDF.
func (o *Office) Print(destination string) error {
	mu.Lock()
	defer mu.Unlock()
	dirPath := filepath.Dir(destination)
	for _, fpath := range o.FilePaths {
		tmpFilename, err := rand.Get()
		if err != nil {
			return err
		}
		tmpDest := fmt.Sprintf("%s/%s.pdf", dirPath, tmpFilename)
		cmd := exec.CommandContext(
			o.Context,
			"unoconv",
			"--format",
			"pdf",
			"--output",
			tmpDest,
			fpath,
		)
		_, err = cmd.Output()
		if o.Context.Err() == context.DeadlineExceeded {
			return errors.New("unoconv: command timed out")
		}
		if err != nil {
			return fmt.Errorf("unoconv: non-zero exit code: %v", err)
		}
	}
	return Merge(dirPath, destination)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(Office))
)
