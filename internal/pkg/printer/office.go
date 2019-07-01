package printer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/labstack/gommon/random"
)

type office struct {
	fpaths []string
	opts   *OfficeOptions
}

// OfficeOptions helps customizing the
// Office printer behaviour.
type OfficeOptions struct {
	WaitTimeout float64
	Landscape   bool
}

// NewOffice returns an Office printer.
func NewOffice(fpaths []string, opts *OfficeOptions) Printer {
	return &office{
		fpaths: fpaths,
		opts:   opts,
	}
}

func (p *office) Print(destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.opts.WaitTimeout)*time.Second)
	defer cancel()
	fpaths := make([]string, len(p.fpaths))
	dirPath := filepath.Dir(destination)
	for i, fpath := range p.fpaths {
		baseFilename := random.String(32)
		tmpDest := fmt.Sprintf("%s/%d%s.pdf", dirPath, i, baseFilename)
		if err := unoconv(ctx, fpath, tmpDest, p.opts); err != nil {
			return err
		}
		fpaths[i] = tmpDest
	}
	if len(fpaths) == 1 {
		return os.Rename(fpaths[0], destination)
	}
	m := &merge{
		ctx:    ctx,
		fpaths: fpaths,
	}
	return m.Print(destination)
}

// nolint: gochecknoglobals
var mu sync.Mutex

func unoconv(ctx context.Context, fpath, destination string, opts *OfficeOptions) error {
	mu.Lock()
	defer mu.Unlock()
	cmdArgs := []string{
		"--format",
		"pdf",
	}
	if opts.Landscape {
		cmdArgs = append(cmdArgs, "--printer", "PaperOrientation=landscape")
	}
	cmdArgs = append(cmdArgs, "--output", destination, fpath)
	cmd := exec.CommandContext(
		ctx,
		"unoconv",
		cmdArgs...,
	)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("unoconv: %v", err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(office))
)
