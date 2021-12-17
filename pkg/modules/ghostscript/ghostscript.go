package ghostscript

import (
	"context"
	"errors"
	"fmt"
	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"go.uber.org/zap"
	"os"
	"sync"
)

func init() {
	gotenberg.MustRegisterModule(Ghostscript{})
}

// Ghostscript abstracts the CLI tool Ghostscript and implements the gotenberg.PDFEngine
// interface.
type Ghostscript struct {
	binPath string
}

// Descriptor returns a QPDF's module descriptor.
func (Ghostscript) Descriptor() gotenberg.ModuleDescriptor {
	return gotenberg.ModuleDescriptor{
		ID:  "ghostscript",
		New: func() gotenberg.Module { return new(Ghostscript) },
	}
}

// Provision sets the modules properties. It returns an error if the
// environment variable QPDF_BIN_PATH is not set.
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
		return fmt.Errorf("ghostscript binary path does not exist: %w", err)
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

// Merge merges PDFs with Ghostscript.
func (engine Ghostscript) Merge(ctx context.Context, logger *zap.Logger, inputPaths []string, outputPath string) error {
	var args []string
	args = append(args, "-dBATCH")
	args = append(args, "-dNOPAUSE")
	args = append(args, "-sDEVICE=pdfwrite")
	args = append(args, fmt.Sprintf("-sOutputFile=%s", outputPath))
	args = append(args, inputPaths...)

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

	return fmt.Errorf("merge PDFs with Ghostscript: %w", err)
}

// Convert converts PDF with this engine
func (engine Ghostscript) Convert(ctx context.Context, logger *zap.Logger, format, inputPath, outputPath string) error {
	var pdfALevel string

	switch format {
	case gotenberg.FormatPDFA1b:
		pdfALevel = "1"
	case gotenberg.FormatPDFA2b:
		pdfALevel = "2"
	case gotenberg.FormatPDFA3b:
		pdfALevel = "3"
	default:
		return fmt.Errorf("convert PDF to '%s' with ghostsript: %w", format, gotenberg.ErrPDFFormatNotAvailable)
	}

	var args []string
	args = append(args, fmt.Sprintf("-dPDFA=%s", pdfALevel))
	args = append(args, "-dBATCH")
	args = append(args, "-dNOPAUSE")
	args = append(args, "-sColorConversionStrategy=UseDeviceIndependentColor")
	args = append(args, "-sDEVICE=pdfwrite")
	args = append(args, "-dPDFACompatibilityPolicy=1")
	args = append(args, fmt.Sprintf("-sOutputFile=%s", outputPath))
	args = append(args, inputPath)

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
	return fmt.Errorf("convert PDF to '%s' with Ghostscript: %w", format, err)
}

var (
	activeInstancesCount   float64
	activeInstancesCountMu sync.RWMutex
)

var (
	_ gotenberg.Module          = (*Ghostscript)(nil)
	_ gotenberg.Provisioner     = (*Ghostscript)(nil)
	_ gotenberg.Validator       = (*Ghostscript)(nil)
	_ gotenberg.MetricsProvider = (*Ghostscript)(nil)
	_ gotenberg.PDFEngine       = (*Ghostscript)(nil)
)
