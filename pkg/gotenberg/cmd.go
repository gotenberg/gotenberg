package gotenberg

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

// Cmd wraps an [exec.Cmd].
type Cmd struct {
	ctx     context.Context
	logger  *zap.Logger
	process *exec.Cmd
}

// Command creates a [Cmd] without a context. It configures the internal
// [exec.Cmd] of [Cmd] so that we may kill its unix process and all its
// children without creating orphans.
//
// See https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773.
func Command(logger *zap.Logger, binPath string, args ...string) *Cmd {
	cmd := exec.Command(binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return &Cmd{
		ctx:     nil,
		logger:  logger.Named(strings.ReplaceAll(binPath, "/", "")),
		process: cmd,
	}
}

// CommandContext creates a [Cmd] with a context. It configures the internal
// [exec.Cmd] of [Cmd] so that we may kill its unix process and all its
// children without creating orphans.
//
// See https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773.
func CommandContext(ctx context.Context, logger *zap.Logger, binPath string, args ...string) (*Cmd, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return &Cmd{
		ctx:     ctx,
		logger:  logger.Named(strings.ReplaceAll(binPath, "/", "")),
		process: cmd,
	}, nil
}

// Start starts the command but does not wait for its completion.
func (cmd *Cmd) Start() error {
	err := cmd.pipeOutput()
	if err != nil {
		return fmt.Errorf("pipe unix process output: %w", err)
	}

	cmd.logger.Debug(fmt.Sprintf("start unix process: %s", strings.Join(cmd.process.Args, " ")))

	err = cmd.process.Start()
	if err != nil {
		return fmt.Errorf("start unix process: %w", err)
	}

	return nil
}

// Wait waits for the command to complete. It should be called when using the
// Start method so that the command does not leak zombies.
func (cmd *Cmd) Wait() error {
	err := cmd.process.Wait()
	if err != nil {
		return fmt.Errorf("wait for unix process: %w", err)
	}

	return nil
}

// Exec executes the command and waits for its completion or until the context
// is done. In any case, it kills the unix process and all its children.
func (cmd *Cmd) Exec() (int, error) {
	if cmd.ctx == nil {
		return 10, errors.New("nil context")
	}

	err := cmd.Start()
	if err != nil {
		if cmd.process.ProcessState == nil {
			return 131, fmt.Errorf("start command: %w", err)
		}

		return cmd.process.ProcessState.ExitCode(), fmt.Errorf("start command: %w", err)
	}

	errChan := make(chan error, 1)

	go func() {
		errChan <- cmd.Wait()
	}()

	select {
	case err = <-errChan:
		errProc := cmd.Kill()
		if errProc != nil {
			cmd.logger.Error(errProc.Error())
		}

		if err == nil {
			return 0, nil
		}

		if cmd.process.ProcessState == nil {
			return 131, fmt.Errorf("unix process error: %w", err)
		}

		return cmd.process.ProcessState.ExitCode(), fmt.Errorf("unix process error: %w", err)
	case <-cmd.ctx.Done():
		errProc := cmd.Kill()
		if errProc != nil {
			cmd.logger.Error(errProc.Error())
		}

		return 62, fmt.Errorf("context done: %w", cmd.ctx.Err())
	}
}

// pipeOutput creates logs entries according to the process stdout and stderr.
// It does nothing if the logging level is not debug.
func (cmd *Cmd) pipeOutput() error {
	checkedEntry := cmd.logger.Check(zap.DebugLevel, "check for debug level before piping unix process output")
	if checkedEntry == nil {
		return nil
	}

	stdout, err := cmd.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe unix process stdout: %w", err)
	}

	stderr, err := cmd.process.StderrPipe()
	if err != nil {
		return fmt.Errorf("unix process sdterr: %w", err)
	}

	// logCommandOutput creates logs entries according to a reader
	// (either stdout or stderr).
	logCommandOutput := func(logger *zap.Logger, reader io.ReadCloser) {
		r := bufio.NewReader(reader)
		defer func(reader io.ReadCloser) {
			err := reader.Close()
			if err != nil && !strings.Contains(err.Error(), "file already closed") {
				logger.Error(fmt.Sprintf("close reader: %s", err))
			}
		}(reader)

		for {
			line, _, err := r.ReadLine()
			if err != nil {
				if err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
					logger.Error(fmt.Sprintf("pipe unix process output error: %s", err))
				}

				break
			}

			if len(line) != 0 {
				logger.Debug(string(line))
			}
		}
	}

	go logCommandOutput(cmd.logger.Named("stdout"), stdout)
	go logCommandOutput(cmd.logger.Named("stderr"), stderr)

	return nil
}

// Kill kills the unix process and all its children without creating orphans.
//
// See https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773.
func (cmd *Cmd) Kill() error {
	if cmd.process == nil {
		// We cannot use the logger here, because for whatever reason using it
		// result to a panic.
		// cmd.logger.Debug("no process, skip killing")
		return nil
	}

	err := syscall.Kill(-cmd.process.Process.Pid, syscall.SIGKILL)
	if err == nil {
		cmd.logger.Debug("unix process killed")
		return nil
	}

	// If the process does not exist anymore, the error is irrelevant.
	if strings.Contains(err.Error(), "no such process") {
		cmd.logger.Debug("unix process already killed")
		return nil
	}

	return fmt.Errorf("kill unix process: %w", err)
}
