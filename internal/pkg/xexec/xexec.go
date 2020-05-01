package xexec

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

/*
Command is a wrapper around exec.Command.

If given xlog.Logger has a xlog.DebugLevel,
also logs the output from the command.
*/
func Command(logger xlog.Logger, binary string, args ...string) (*exec.Cmd, error) {
	const op string = "xexec.Command"
	cmd := exec.Command(binary, args...)
	if err := pipe(logger, cmd); err != nil {
		return nil, xerror.New(op, err)
	}
	return cmd, nil
}

/*
CommandContext is a wrapper around exec.CommandContext.

If given xlog.Logger has a xlog.DebugLevel,
also logs the output from the command.
*/
func CommandContext(ctx context.Context, logger xlog.Logger, binary string, args ...string) (*exec.Cmd, error) {
	const op string = "xexec.CommandContext"
	cmd := exec.CommandContext(ctx, binary, args...)
	if err := pipe(logger, cmd); err != nil {
		return nil, xerror.New(op, err)
	}
	return cmd, nil
}

/*
Run runs a command.

If command finishes or fails to finish
before context.Context deadline, kill the
corresponding process in a way which does
not leak orphan processes.
*/
func Run(ctx context.Context, logger xlog.Logger, binary string, args ...string) error {
	const op string = "xexec.Run"
	resolver := func() error {
		cmd, err := Command(
			logger,
			binary,
			args...,
		)
		if err != nil {
			return err
		}
		LogBeforeExecute(logger, cmd)
		// see https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773.
		kill := func() {
			err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			if err == nil {
				return
			}
			if !strings.Contains(err.Error(), "no such process") {
				logger.ErrorOp(op, err)
			}
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			return err
		}
		result := make(chan error, 1)
		go func() {
			result <- cmd.Wait()
		}()
		select {
		case err := <-result:
			logger.DebugOpf(op, "command '%s' finished", strings.Join(cmd.Args, " "))
			kill()
			return err
		case <-ctx.Done():
			logger.DebugOpf(op, "command '%s' failed to finish before context.Context deadline", strings.Join(cmd.Args, " "))
			kill()
			return ctx.Err()
		}
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// LogBeforeExecute logs a command before its execution.
func LogBeforeExecute(logger xlog.Logger, cmd *exec.Cmd) {
	const op string = "xexec.LogBeforeExecute"
	logger.DebugOpf(op, "executing command: %s", strings.Join(cmd.Args, " "))
}

func pipe(logger xlog.Logger, cmd *exec.Cmd) error {
	const op string = "xexec.pipe"
	if logger.Level() != xlog.DebugLevel {
		return nil
	}
	// if xlog.DebugLevel, log the output
	// from the command.
	resolver := func() error {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		go logCommandOutput(logger, stdout, "stdout", cmd)
		go logCommandOutput(logger, stderr, "stderr", cmd)
		return nil
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func logCommandOutput(logger xlog.Logger, reader io.ReadCloser, outputType string, cmd *exec.Cmd) {
	var buf bytes.Buffer
	buf.WriteString(outputType)
	for _, arg := range cmd.Args {
		buf.WriteString(fmt.Sprintf(".%s", arg))
	}
	op := buf.String()
	r := bufio.NewReader(reader)
	defer reader.Close() // nolint: errcheck
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
				logger.ErrorOp(op, err)
			}
			break
		}
		if len(line) != 0 {
			logger.DebugOp(op, string(line))
		}
	}
}
