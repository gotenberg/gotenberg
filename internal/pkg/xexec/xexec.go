package xexec

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

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

// LogBeforeExecute logs a command before its execution.
func LogBeforeExecute(logger xlog.Logger, cmd *exec.Cmd) {
	const op string = "xexec.LogBeforeExecute"
	logger.DebugfOp(op, "executing command: %s", strings.Join(cmd.Args, " "))
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
			if err != io.EOF {
				logger.ErrorOp(op, err)
			}
			break
		}
		if len(line) != 0 {
			logger.DebugOp(op, string(line))
		}
	}
}
