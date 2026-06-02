package gotenberg

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// Cmd wraps an [exec.Cmd].
type Cmd struct {
	ctx     context.Context
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
		ctx:     nil,
		logger:  logger.With(slog.String("logger", strings.ReplaceAll(binPath, "/", ""))),
		process: cmd,
	}
}

// CommandContext creates a [Cmd] with a context. It configures the internal
// [exec.Cmd] of [Cmd] so that we may kill its unix process and all its
// children without creating orphans.
//
// See https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773.
func CommandContext(ctx context.Context, logger *slog.Logger, binPath string, args ...string) (*Cmd, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return &Cmd{
		ctx:     ctx,
		logger:  logger.With(slog.String("logger", strings.ReplaceAll(binPath, "/", ""))),
		process: cmd,
	}, nil
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
//
// When the context carries an active trace span, Exec records a
// "process.exec" client span around the execution. It is the single
// instrumentation point for every short-lived external binary (soffice, pdftk,
// qpdf, exiftool, pdfcpu). The span is skipped when there is no active parent,
// so process starts performed off the request path do not emit orphan roots.
func (cmd *Cmd) Exec() (int, error) {
	if cmd.ctx == nil {
		return 10, errors.New("nil context")
	}

	var span trace.Span
	if trace.SpanContextFromContext(cmd.ctx).IsValid() {
		_, span = Tracer().Start(cmd.ctx, "process.exec",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(semconv.ProcessExecutableName(filepath.Base(cmd.process.Path))),
		)
		defer span.End()
	}

	code, err := cmd.exec()

	if span != nil {
		span.SetAttributes(attribute.Int("process.exit.code", code))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(semconv.ErrorTypeKey.String(execErrorType(cmd.ctx)))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}

	return code, err
}

// exec runs the command and returns its exit code and error.
func (cmd *Cmd) exec() (int, error) {
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
			cmd.logger.ErrorContext(context.Background(), errProc.Error())
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
			cmd.logger.ErrorContext(context.Background(), errProc.Error())
		}

		return 62, fmt.Errorf("context done: %w", cmd.ctx.Err())
	}
}

// execErrorType maps an execution failure to a bounded semconv error.type
// value.
func execErrorType(ctx context.Context) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "context_deadline_exceeded"
	case errors.Is(ctx.Err(), context.Canceled):
		return "context_canceled"
	default:
		return "process_error"
	}
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

	// logCommandOutput creates logs entries according to a reader
	// (either stdout or stderr).
	logCommandOutput := func(logger *slog.Logger, reader io.ReadCloser) {
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
