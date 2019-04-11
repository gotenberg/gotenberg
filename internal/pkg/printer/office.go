package printer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
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
		baseFilename, err := rand.Get()
		if err != nil {
			return err
		}
		tmpDest := fmt.Sprintf("%s/%d%s.pdf", dirPath, i, baseFilename)
		if err := p.print(ctx, fpath, tmpDest); err != nil {
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

func (p *office) print(ctx context.Context, fpath, destination string) error {
	cmdArgs := []string{
		"--format",
		"pdf",
	}
	if p.opts.Landscape {
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
