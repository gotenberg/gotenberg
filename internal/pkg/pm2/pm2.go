package pm2

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"

	log "github.com/thecodingmachine/gotenberg/internal/pkg/logger"
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
	logger         *log.StandardLogger
}

func (m *processManager) start(p Process) error {
	if err := m.pm2(p, "start"); err != nil {
		return err
	}
	p.warmup()
	if !p.viable() {
		attempts := 0
		for attempts < 5 && !p.viable() {
			if err := m.pm2(p, "restart"); err != nil {
				m.heuristicState = errorState
				return err
			}
			p.warmup()
			attempts++
		}
		if !p.viable() {
			m.heuristicState = errorState
			return fmt.Errorf("failed to launch %s", p.Fullname())
		}
	}
	m.heuristicState = runningState
	return nil
}

func (m *processManager) shutdown(p Process) error {
	if m.heuristicState != runningState {
		return nil
	}
	if err := m.pm2(p, "stop"); err != nil {
		m.heuristicState = errorState
		return err
	}
	m.heuristicState = stoppedState
	return nil
}

func (m *processManager) pm2(p Process, cmdName string) error {
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
	m.logger.Debugf("executing command: %v", strings.Join(cmd.Args, " "))
	processStdOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed getting stdout from %s: %s", p.Fullname(), err)
	}
	processStdErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed getting stderr from %s: %s", p.Fullname(), err)
	}
	readFromPipe := func(outputType string, reader io.ReadCloser) {
		r := bufio.NewReader(reader)
		defer reader.Close() // nolint: errcheck
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				if err != io.EOF {
					m.logger.Errorf("error reading from %s for process %s", outputType, p.Fullname())
				}
				break
			}
			if len(line) != 0 {
				m.logger.Debugf("%s %s: %s", p.Fullname(), outputType, string(line))
			}
		}
	}
	go readFromPipe("stdout", processStdOut)
	go readFromPipe("stderr", processStdErr)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s %s with PM2: %v", cmdName, p.Fullname(), err)
	}
	return nil
}
