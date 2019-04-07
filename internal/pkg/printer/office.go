package printer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/rand"
)

var mu sync.Mutex

type office struct {
	fpaths []string
	opts   *OfficeOptions
}

// OfficeOptions helps customizing the
// Office printer behaviour.
type OfficeOptions struct {
	Landscape   bool
	WaitTimeout float64
}

// NewOffice returns an Office printer.
func NewOffice(fpaths []string, opts *OfficeOptions) (Printer, error) {
	if len(fpaths) == 0 {
		return nil, fmt.Errorf("office requires at least one document to convert: got %d", len(fpaths))
	}
	sort.Strings(fpaths)
	// if no custom timeout, set default timeout to 30 seconds.
	if opts.WaitTimeout == 0.0 {
		opts.WaitTimeout = 30.0
	}
	return &office{
		fpaths: fpaths,
		opts:   opts,
	}, nil
}

func (o *office) Print(destination string) error {
	mu.Lock()
	defer mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.opts.WaitTimeout)*time.Second)
	defer cancel()
	fpaths := make([]string, len(o.fpaths))
	dirPath := filepath.Dir(destination)
	for i, fpath := range o.fpaths {
		baseFilename, err := rand.Get()
		if err != nil {
			return err
		}
		tmpDest := fmt.Sprintf("%s/%d_%s.pdf", dirPath, i, baseFilename)
		if err := o.print(ctx, fpath, tmpDest); err != nil {
			return err
		}
		fpaths[i] = tmpDest
	}
	if len(fpaths) == 1 {
		return os.Rename(fpaths[0], destination)
	}
	return o.merge(ctx, fpaths, destination)
}

func (o *office) print(ctx context.Context, fpath, destination string) error {
	cmdArgs := []string{
		"--format",
		"pdf",
	}
	if o.opts.Landscape {
		cmdArgs = append(cmdArgs, "--printer", "PaperOrientation=landscape")
	}
	cmdArgs = append(cmdArgs, "--output", destination, fpath)
	cmd := exec.CommandContext(
		ctx,
		"unoconv",
		cmdArgs...,
	)
	_, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return errors.New("unoconv: command timed out")
	}
	if err != nil {
		return fmt.Errorf("unoconv: non-zero exit code: %v", err)
	}
	return nil
}

func (o *office) merge(ctx context.Context, fpaths []string, destination string) error {
	p := &merge{
		ctx:    ctx,
		fpaths: fpaths,
	}
	return p.Print(destination)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Printer(new(office))
)
