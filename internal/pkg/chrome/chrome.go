package chrome

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/mafredri/cdp/devtool"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
)

func Start(ctx context.Context, logger xlog.Logger) (*os.Process, error) {
	const op string = "chrome.Start"
	logger.DebugOp(op, "starting new Google Chrome process on port 9222...")
	resolver := func() (*os.Process, error) {
		binary := "google-chrome-stable"
		args := []string{
			"--no-sandbox",
			"--headless",
			// see https://github.com/GoogleChrome/puppeteer/issues/2410.
			"--font-render-hinting=medium",
			"--remote-debugging-port=9222",
			"--disable-gpu",
			"--disable-translate",
			"--disable-extensions",
			"--disable-background-networking",
			"--safebrowsing-disable-auto-update",
			"--disable-sync",
			"--disable-default-apps",
			"--hide-scrollbars",
			"--metrics-recording-only",
			"--mute-audio",
			"--no-first-run",
		}
		cmd, err := xexec.CommandContext(ctx, logger, binary, args...)
		if err != nil {
			return nil, err
		}
		// we try to start the process.
		xexec.LogBeforeExecute(logger, cmd)
		if err := cmd.Start(); err != nil {
			return cmd.Process, err
		}
		// we wait the process to be ready.
		warmup(logger)
		// if the process failed to start correctly,
		// we have to restart it.
		if !isViable(logger) {
			return restart(ctx, logger, cmd)
		}
		return cmd.Process, nil
	}
	proc, err := resolver()
	if err != nil {
		if errKill := Kill(logger, proc); errKill != nil {
			logger.ErrorOp(op, errKill)
		}
		return nil, xerror.New(op, err)
	}
	return proc, nil
}

func Kill(logger xlog.Logger, proc *os.Process) error {
	const op string = "chrome.Kill"
	resolver := func() error {
		logger.DebugOp(op, "removing Google Chrome process using port 9222...")
		if proc == nil {
			logger.DebugOp(op, "no Google Chrome process using port 9222 found, skipping")
			return nil
		}
		return proc.Kill()
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func restart(ctx context.Context, logger xlog.Logger, cmd *exec.Cmd) (*os.Process, error) {
	const op string = "chrome.restart"
	resolver := func() (*os.Process, error) {
		// we try to restart the process.
		xexec.LogBeforeExecute(logger, cmd)
		if err := cmd.Start(); err != nil {
			return cmd.Process, err
		}
		// we wait the process to be ready.
		warmup(logger)
		// if the process failed to restart correctly,
		// we have to restart it again.
		if !isViable(logger) {
			return restart(ctx, logger, cmd)
		}
		return cmd.Process, nil
	}
	proc, err := resolver()
	if err != nil {
		return proc, xerror.New(op, err)
	}
	return proc, err
}

func isViable(logger xlog.Logger) bool {
	const op string = "chrome.isViable"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	endpoint := "http://localhost:9222"
	logger.DebugfOp(
		op,
		"checking Google Chrome process viability via endpoint '%s/json/version'",
		endpoint,
	)
	v, err := devtool.New(endpoint).Version(ctx)
	if err != nil {
		logger.ErrorfOp(
			op,
			"Google Chrome is not viable as endpoint returned '%v'",
			err,
		)
		return false
	}
	logger.DebugfOp(
		op,
		"Google Chrome is viable as endpoint returned '%v'",
		v,
	)
	return true
}

func warmup(logger xlog.Logger) {
	const op string = "chrome.warmup"
	warmupTime := xtime.Duration(10)
	logger.DebugfOp(
		op,
		"waiting '%v' for allowing Google Chrome to warmup",
		warmupTime,
	)
	time.Sleep(warmupTime)
}
