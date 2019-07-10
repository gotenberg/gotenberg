package printer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/labstack/gommon/random"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/timeout"
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
	const op string = "printer.office.Print"
	ctx, cancel := timeout.Context(p.opts.WaitTimeout)
	defer cancel()
	fpaths := make([]string, len(p.fpaths))
	resolver := func() error {
		dirPath := filepath.Dir(destination)
		for i, fpath := range p.fpaths {
			baseFilename := random.String(32)
			tmpDest := fmt.Sprintf("%s/%d%s.pdf", dirPath, i, baseFilename)
			if err := unoconv(ctx, fpath, tmpDest, p.opts); err != nil {
				return &standarderror.Error{Op: op, Err: err}
			}
			fpaths[i] = tmpDest
		}
		return nil
	}
	if err := resolver(); err != nil {
		return timeout.Err(ctx, err)
	}
	if len(fpaths) == 1 {
		if err := os.Rename(fpaths[0], destination); err != nil {
			return &standarderror.Error{Op: op, Err: err}
		}
		return nil
	}
	m := &merge{
		ctx:    ctx,
		fpaths: fpaths,
	}
	if err := m.Print(destination); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

// nolint: gochecknoglobals
var mu sync.Mutex

func unoconv(ctx context.Context, fpath, destination string, opts *OfficeOptions) error {
	const op string = "printer.unoconv"
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
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(office))
)
