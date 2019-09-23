package prinery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xrand"
)

type sofficeProcess struct {
	unoPort uint
}

func newSofficesProcesses(nInstances int64) []process {
	processes := make([]process, nInstances)
	var i int64
	var currentPort uint = 2002
	for i = 0; i < nInstances; i++ {
		processes[i] = sofficeProcess{
			unoPort: currentPort,
		}
		currentPort++
	}
	return processes
}

func (p sofficeProcess) id() string {
	return fmt.Sprintf("%s-%d", p.binary(), p.port())
}

func (p sofficeProcess) host() string {
	return "127.0.0.1"
}

func (p sofficeProcess) port() uint {
	return p.unoPort
}

func (p sofficeProcess) spec() processSpec {
	return p
}

func (p sofficeProcess) binary() string {
	return "soffice"
}

func (p sofficeProcess) args() []string {
	return []string{
		// see https://ask.libreoffice.org/en/question/42975/how-can-i-run-multiple-instances-of-sofficebin-at-a-time/.
		fmt.Sprintf("-env:UserInstallation=file:///tmp/%d", p.port()),
		"--headless",
		"--invisible",
		"--nocrashreport",
		"--nodefault",
		"--nofirststartwizard",
		"--nologo",
		"--norestore",
		fmt.Sprintf("--accept=socket,host=%s,port=%d,tcpNoDelay=1;urp;StarOffice.ComponentContext", p.host(), p.port()),
	}
}

func (p sofficeProcess) warmupTime() time.Duration {
	return 3 * time.Second
}

func (p sofficeProcess) viabilityFunc() func(logger xlog.Logger) bool {
	const op string = "prinery.sofficeProcess.viabilityFunc"
	return func(logger xlog.Logger) bool {
		// TODO find a way to check.
		return true
	}
}

type unoconvPrinter struct {
	logger xlog.Logger
	fpaths []string
	opts   UnoconvPrintOptions
}

func (p unoconvPrinter) print(ctx context.Context, spec processSpec, dest string) error {
	const op string = "prinery.unoconvPrinter.print"
	resolver := func() error {
		fpaths := make([]string, len(p.fpaths))
		dirPath := filepath.Dir(dest)
		for i, fpath := range p.fpaths {
			baseFilename := xrand.Get()
			tmpDest := fmt.Sprintf("%s/%d%s.pdf", dirPath, i, baseFilename)
			p.logger.DebugfOp(op, "converting '%s' to PDF...", fpath)
			if err := p.unoconv(ctx, spec, fpath, tmpDest); err != nil {
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
		merger := mergePrinter{
			logger: p.logger,
			fpaths: fpaths,
		}
		return merger.print(ctx, nil, dest)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p unoconvPrinter) unoconv(ctx context.Context, spec processSpec, fpath, dest string) error {
	const op string = "prinery.unoconvPrinter.unoconv"
	resolver := func() error {
		args := []string{
			"--server",
			spec.host(),
			"--port",
			fmt.Sprintf("%d", spec.port()),
			"--format",
			"pdf",
		}
		if p.opts.Landscape {
			args = append(args, "--printer", "PaperOrientation=landscape")
		}
		args = append(args, "--output", dest, fpath)
		cmd, err := xexec.CommandContext(
			ctx,
			p.logger,
			"unoconv",
			args...,
		)
		if err != nil {
			return err
		}
		xexec.LogBeforeExecute(p.logger, cmd)
		return cmd.Run()
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = processSpec(new(sofficeProcess))
	_ = process(new(sofficeProcess))
	_ = printer(new(unoconvPrinter))
)
