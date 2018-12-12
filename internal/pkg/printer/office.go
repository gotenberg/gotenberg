package printer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

var mu sync.Mutex

// Office facilitates Office documents to PDF conversion.
type Office struct {
	Context     context.Context
	FilePaths   []string
	PaperWidth  float64
	PaperHeight float64
	Landscape   bool
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
		paperSize, err := unoconvPaperSize(o.PaperWidth, o.PaperHeight)
		if err != nil {
			return err
		}
		cmdArgs := []string{
			"--format",
			"pdf",
			"--printer",
			paperSize,
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

func unoconvPaperSize(paperWidth, paperHeight float64) (string, error) {
	width, err := strconv.Atoi(fmt.Sprintf("%.0f", paperWidth*25.4))
	if err != nil {
		return "", fmt.Errorf("%0.f: converting width to millimiter: %v", paperWidth, err)
	}
	height, err := strconv.Atoi(fmt.Sprintf("%.0f", paperHeight*25.4))
	if err != nil {
		return "", fmt.Errorf("%0.f: converting height to millimiter: %v", paperHeight, err)
	}
	return fmt.Sprintf("PaperSize=%dx%d", width, height), nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(Office))
)
