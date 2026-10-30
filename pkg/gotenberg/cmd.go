package gotenberg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"syscall"
)

// Cmd wraps an [exec.Cmd].
type Cmd struct {
	logger  *slog.Logger
	process *exec.Cmd
}

// Command creates a [Cmd] without a context. It configures the internal
// [exec.Cmd] of [Cmd] so that we may kill its unix process and all its
// children without creating orphans.
//
// See https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773.
func Command(logger *slog.Logger, binPath string, args ...string) *Cmd {
	cmd := exec.Command(binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return &Cmd{
		logger:  logger.With(slog.String("logger", strings.ReplaceAll(binPath, "/", ""))),
		process: cmd,
	}
}

// SetEnv replaces the environment variables passed to the underlying
// process. When SetEnv is not called, the process inherits the parent's
// environment.
func (cmd *Cmd) SetEnv(env []string) {
	cmd.process.Env = env
}

// Start starts the command but does not wait for its completion.
func (cmd *Cmd) Start() error {
	err := cmd.pipeOutput()
	if err != nil {
		return fmt.Errorf("pipe unix process output: %w", err)
	}

	cmd.logger.DebugContext(context.Background(), fmt.Sprintf("start unix process: %s", strings.Join(cmd.process.Args, " ")))

	err = cmd.process.Start()
	if err != nil {
		return fmt.Errorf("start unix process: %w", err)
	}

	return nil
}

// StartWithPipes starts the unix process and returns its standard input and
// output, for a process that speaks a protocol over them. Its standard error
// is logged at debug level. Call Wait so that the process does not leak a
// zombie.
func (cmd *Cmd) StartWithPipes() (io.WriteCloser, io.ReadCloser, error) {
	stdin, err := cmd.process.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pipe unix process stdin: %w", err)
	}

	stdout, err := cmd.process.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pipe unix process stdout: %w", err)
	}

	if cmd.logger.Enabled(context.Background(), slog.LevelDebug) {
		stderr, err := cmd.process.StderrPipe()
		if err != nil {
			return nil, nil, fmt.Errorf("pipe unix process stderr: %w", err)
		}

		go logCommandOutput(cmd.logger.With(slog.String("logger", "stderr")), stderr)
	}

	cmd.logger.DebugContext(context.Background(), fmt.Sprintf("start unix process: %s", strings.Join(cmd.process.Args, " ")))

	err = cmd.process.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("start unix process: %w", err)
	}

	return stdin, stdout, nil
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

// pipeOutput creates logs entries according to the process stdout and stderr.
// It does nothing if the logging level is not debug.
func (cmd *Cmd) pipeOutput() error {
	if !cmd.logger.Enabled(context.Background(), slog.LevelDebug) {
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

	go logCommandOutput(cmd.logger.With(slog.String("logger", "stdout")), stdout)
	go logCommandOutput(cmd.logger.With(slog.String("logger", "stderr")), stderr)

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
		cmd.logger.DebugContext(context.Background(), "unix process killed")
		return nil
	}

	// If the process does not exist anymore, the error is irrelevant.
	if strings.Contains(err.Error(), "no such process") {
		cmd.logger.DebugContext(context.Background(), "unix process already killed")
		return nil
	}

	return fmt.Errorf("kill unix process: %w", err)
}

// logCommandOutput creates log entries from the lines of a process output.
func logCommandOutput(logger *slog.Logger, reader io.ReadCloser) {
	r := bufio.NewReader(reader)
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil && !strings.Contains(err.Error(), "file already closed") {
			logger.ErrorContext(context.Background(), fmt.Sprintf("close reader: %s", err))
		}
	}(reader)

	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "file already closed") {
				logger.ErrorContext(context.Background(), fmt.Sprintf("pipe unix process output error: %s", err))
			}

			break
		}

		if len(line) != 0 {
			logger.DebugContext(context.Background(), string(line))
		}
	}
}
