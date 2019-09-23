package print

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thecodingmachine/gotenberg/internal/pkg/process"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
)

type officePrint struct {
	logger xlog.Logger
	fpaths []string
	opts   OfficePrintOptions
}

// OfficePrintOptions helps customizing the
// LibreOffice Print result.
type OfficePrintOptions struct {
	Landscape bool
}

// DefaultOfficePrinterOptions returns the default
// LibreOffice Print options.
func DefaultOfficePrinterOptions() OfficePrintOptions {
	return OfficePrintOptions{
		Landscape: false,
	}
}

// NewOfficePrint returns a Print for
// converting Office documents to PDF.
func NewOfficePrint(logger xlog.Logger, fpaths []string, opts OfficePrintOptions) Print {
	return officePrint{
		logger: logger,
		fpaths: fpaths,
		opts:   opts,
	}
}

func (p officePrint) Print(ctx context.Context, dest string, proc process.Process) error {
	const op string = "print.officePrint.Print"
	resolver := func() error {
		fpaths := make([]string, len(p.fpaths))
		dirPath := filepath.Dir(dest)
		for i, fpath := range p.fpaths {
			baseFilename := xrand.Get()
			tmpDest := fmt.Sprintf("%s/%d%s.pdf", dirPath, i, baseFilename)
			p.logger.DebugfOp(op, "converting '%s' to PDF...", fpath)
			if err := unoconv(
				ctx,
				p.logger,
				proc,
				p.opts,
				fpath,
				tmpDest,
			); err != nil {
				return err
			}
			p.logger.DebugfOp(op, "'%s.pdf' created", baseFilename)
			fpaths[i] = tmpDest
		}
		if len(fpaths) == 1 {
			p.logger.DebugOp(op, "only one PDF created, nothing to merge")
			if err := os.Rename(fpaths[0], dest); err != nil {
				return err
			}
			return nil
		}
		merger := NewMergePrint(p.logger, fpaths)
		return merger.Print(ctx, dest, nil)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func unoconv(
	ctx context.Context,
	logger xlog.Logger,
	proc process.Process,
	opts OfficePrintOptions,
	fpath, dest string,
) error {
	const op string = "print.unoconv"
	resolver := func() error {
		args := []string{
			"--server",
			proc.Host(),
			"--port",
			fmt.Sprintf("%d", proc.Port()),
			"--format",
			"pdf",
		}
		if opts.Landscape {
			args = append(args, "--printer", "PaperOrientation=landscape")
		}
		args = append(args, "--output", dest, fpath)
		cmd, err := xexec.CommandContext(
			ctx,
			logger,
			"unoconv",
			args...,
		)
		if err != nil {
			return err
		}
		xexec.LogBeforeExecute(logger, cmd)
		return cmd.Run()
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Print(new(officePrint))
)
