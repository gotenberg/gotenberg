package pm2

import (
	"fmt"
	"os/exec"
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
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s %s with PM2: %v", cmdName, p.Fullname(), err)
	}
	return nil
}
