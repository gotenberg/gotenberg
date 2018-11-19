package printer

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

var mu sync.Mutex

// Office facilitates Office document to PDF conversion.
type Office struct {
	Context  context.Context
	FilePath string
}

// Print converts Office document to PDF.
func (o *Office) Print(destination string) error {
	if !fileExists(o.FilePath) {
		return fmt.Errorf("%s: file does not exist", o.FilePath)
	}
	mu.Lock()
	defer mu.Unlock()
	cmd := exec.CommandContext(
		o.Context,
		"unoconv",
		"--stdout",
		"--no-launch",
		"--format pdf",
		destination,
	)
	_, err := cmd.Output()
	if o.Context.Err() == context.DeadlineExceeded {
		return fmt.Errorf("%s: command timed out", o.FilePath)
	}
	if err != nil {
		return fmt.Errorf("%s: non-zero exit code: %v", o.FilePath, err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(Office))
)
