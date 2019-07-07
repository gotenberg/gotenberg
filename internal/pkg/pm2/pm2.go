package pm2

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/thecodingmachine/gotenberg/internal/pkg/logger"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

const (
	stoppedState = iota
	runningState
	errorState
)

// Process is a type that can start or
// shutdown a process with PM2.
type Process interface {
	Fullname() string
	Start() error
	Shutdown() error
	args() []string
	name() string
	viable() bool
	warmup()
}

type processManager struct {
	heuristicState int32
	logger         *logger.Logger
}

func (m *processManager) start(p Process) error {
	const op = "pm2.start"
	if err := m.pm2(p, "start"); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	p.warmup()
	if !p.viable() {
		attempts := 0
		for attempts < 5 && !p.viable() {
			if err := m.pm2(p, "restart"); err != nil {
				m.heuristicState = errorState
				return &standarderror.Error{Op: op, Err: err}
			}
			p.warmup()
			attempts++
		}
		if !p.viable() {
			m.heuristicState = errorState
			return &standarderror.Error{
				Op:      op,
				Message: fmt.Sprintf("failed to launch %s", p.Fullname()),
			}
		}
	}
	m.heuristicState = runningState
	return nil
}

func (m *processManager) shutdown(p Process) error {
	const op = "pm2.shutdown"
	if m.heuristicState != runningState {
		return nil
	}
	if err := m.pm2(p, "stop"); err != nil {
		m.heuristicState = errorState
		return &standarderror.Error{Op: op, Err: err}
	}
	m.heuristicState = stoppedState
	return nil
}

func (m *processManager) pm2(p Process, cmdName string) error {
	const op = "pm2.pm2"
	cmdArgs := []string{
		cmdName,
		p.name(),
	}
	if cmdName == "start" {
		cmdArgs = append(cmdArgs, "--interpreter=none", "--")
		cmdArgs = append(cmdArgs, p.args()...)
	}
	cmd := exec.Command(
		"pm2",
		cmdArgs...,
	)
	m.logger.DebugfOp(op, "executing command: %s", strings.Join(cmd.Args, " "))
	processStdOut, err := cmd.StdoutPipe()
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	processStdErr, err := cmd.StderrPipe()
	if err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	readFromPipe := func(outputType string, reader io.ReadCloser) {
		readFromPipeOp := fmt.Sprintf("pm2.%s.%s", p.name(), outputType)
		r := bufio.NewReader(reader)
		defer reader.Close() // nolint: errcheck
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				if err != io.EOF {
					m.logger.ErrorOp(readFromPipeOp, err)
				}
				break
			}
			if len(line) != 0 {
				m.logger.DebugfOp(readFromPipeOp, string(line))
			}
		}
	}
	go readFromPipe("stdout", processStdOut)
	go readFromPipe("stderr", processStdErr)
	if err := cmd.Start(); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}
