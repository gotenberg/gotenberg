package ghostscript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
)

func init() {
	gotenberg.MustRegisterModule(Ghostscript{})
}

// Ghostscript abstracts the CLI tool Ghostscript and implements the gotenberg.PDFEngine
// interface.
type Ghostscript struct {
	binPath string
}

// Descriptor returns a Ghostscript's module descriptor.
func (Ghostscript) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "ghostscript",
		New: func() gotenberg.Module { return new(Ghostscript) },
	}
}

// Provision sets the modules properties. It returns an error if the
// environment variable GHOSTSCRIPT_BIN_PATH is not set.
func (engine *Ghostscript) Provision(_ *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("GHOSTSCRIPT_BIN_PATH")
	if !ok {
		return errors.New("GHOSTSCRIPT_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine Ghostscript) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("Ghostscript binary path does not exist: %w", err)
	}

	return nil
}

// Metrics returns the metrics.
func (engine Ghostscript) Metrics() ([]gotenberg.Metric, error) {
	return []gotenberg.Metric{
		{
			Name:        "ghostscript_active_instances_count",
			Description: "Current number of active Ghostscript instances.",
			Read: func() float64 {
				activeInstancesCountMu.RLock()
				defer activeInstancesCountMu.RUnlock()

				return activeInstancesCount
			},
		},
	}, nil
}

// Merge is not available for this PDF engine.
func (engine Ghostscript) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	return fmt.Errorf("merge PDFs with Ghostscript: %w", gotenberg.ErrPDFEngineMethodNotAvailable)
}

// Convert is not available for this PDF engine.
func (engine Ghostscript) Convert(_ context.Context, _ *zap.Logger, format, _, _ string) error {
	return fmt.Errorf("convert PDF to '%s' with Ghostscript: %w", format, gotenberg.ErrPDFEngineMethodNotAvailable)
}


// Compress the given pdf
func (engine Ghostscript) Compress(ctx context.Context, logger *zap.Logger, inputPath string, outputPath string) error {
	var args []string
	args = append(args, "-sDEVICE=pdfwrite")
	args = append(args, "-dCompatibilityLevel=1.4")
	args = append(args, "-dPDFSETTINGS=/screen")
	args = append(args, "-dNOPAUSE")
	args = append(args, "-dQUIET")
	args = append(args, "-dBATCH")
	args = append(args, "-sOutputFile=" + outputPath)
	args = append(args, inputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	activeInstancesCountMu.Lock()
	activeInstancesCount += 1
	activeInstancesCountMu.Unlock()

	_, err = cmd.Exec()

	activeInstancesCountMu.Lock()
	activeInstancesCount -= 1
	activeInstancesCountMu.Unlock()

	if err == nil {
		return nil
	}

	return fmt.Errorf("compress PDFs with Ghostscript: %w", err)
}

var (
	activeInstancesCount   float64
	activeInstancesCountMu sync.RWMutex
)

// Interface guards.
var (
	_ gotenberg.Module          = (*Ghostscript)(nil)
	_ gotenberg.Provisioner     = (*Ghostscript)(nil)
	_ gotenberg.Validator       = (*Ghostscript)(nil)
	_ gotenberg.MetricsProvider = (*Ghostscript)(nil)
	_ gotenberg.PDFEngine       = (*Ghostscript)(nil)
)
