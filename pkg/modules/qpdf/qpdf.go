package qpdf

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
	gotenberg.MustRegisterModule(QPDF{})
}

// QPDF abstracts the CLI tool QPDF and implements the gotenberg.QPDF
// interface.
type QPDF struct {
	binPath string
}

// Descriptor returns a QPDF's module descriptor.
func (QPDF) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "qpdf",
		New: func() gotenberg.Module { return new(QPDF) },
	}
}

// Provision sets the modules properties. It returns an error if the
// environment variable QPDF_BIN_PATH is not set.
func (engine *QPDF) Provision(_ *gotenberg.Context) error {
	binPath, ok := os.LookupEnv("QPDF_BIN_PATH")
	if !ok {
		return errors.New("QPDF_BIN_PATH environment variable is not set")
	}

	engine.binPath = binPath

	return nil
}

// Validate validates the module properties.
func (engine QPDF) Validate() error {
	_, err := os.Stat(engine.binPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("QPDF binary path does not exist: %w", err)
	}

	return nil
}

// Metrics returns the metrics.
func (engine QPDF) Metrics() ([]gotenberg.Metric, error) {
	return []gotenberg.Metric{
		{
			Name:        "qpdf_active_instances_count",
			Description: "Current number of active QPDF instances.",
			Read: func() float64 {
				activeInstancesCountMu.RLock()
				defer activeInstancesCountMu.RUnlock()

				return activeInstancesCount
			},
		},
	}, nil
}

// Merge merges the given PDFs into a unique PDF.
func (engine QPDF) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var args []string
	args = append(args, "--empty")
	args = append(args, "--pages")
	args = append(args, inputPaths...)
	args = append(args, "--", outputPath)

	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	activeInstancesCountMu.Lock()
	activeInstancesCount += 1
	activeInstancesCountMu.Unlock()

	err = cmd.Exec()

	activeInstancesCountMu.Lock()
	activeInstancesCount -= 1
	activeInstancesCountMu.Unlock()

	if err == nil {
		return nil
	}

	return fmt.Errorf("merge PDFs with QPDF: %w", err)
}

// Convert is not available for this PDF engine.
func (engine QPDF) Convert(_ context.Context, _ *zap.Logger, format, _, _ string) error {
	return fmt.Errorf("convert PDF to '%s' with QPDF: %w", format, gotenberg.ErrPDFEngineMethodNotAvailable)
}

var (
	activeInstancesCount   float64
	activeInstancesCountMu sync.RWMutex
)

var (
	_ gotenberg.Module          = (*QPDF)(nil)
	_ gotenberg.Provisioner     = (*QPDF)(nil)
	_ gotenberg.Validator       = (*QPDF)(nil)
	_ gotenberg.MetricsProvider = (*QPDF)(nil)
	_ gotenberg.PDFEngine       = (*QPDF)(nil)
)
