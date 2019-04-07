package pm2

import (
	"fmt"
	"os/exec"

	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

const (
	// RunningState describes a running PM2 process.
	RunningState = iota
	// StoppedState describes a stopped PM2 process.
	StoppedState
	// ErrorState describes a PM2 process in error.
	ErrorState
)

// Process is a type that can start or
// shutdown a process with PM2.
type Process interface {
	Start() error
	Shutdown() error
	State() int32
	state(state int32)
	args() []string
	name() string
	fullname() string
	viable() bool
	warmup()
}

const maxRestartAttempts int = 5

func startProcess(p Process) error {
	if err := startPM2Command(p, "start"); err != nil {
		return err
	}
	p.warmup()
	if !p.viable() {
		attempts := 0
		for attempts < maxRestartAttempts && !p.viable() {
			if err := startPM2Command(p, "restart"); err != nil {
				p.state(ErrorState)
				return err
			}
			p.warmup()
			attempts++
		}
		if !p.viable() {
			p.state(ErrorState)
			return fmt.Errorf("failed to launch %s", p.fullname())
		}
	}
	p.state(RunningState)
	return nil
}

var notifyCommandNames = map[string]string{
	"stop":    "stopped",
	"restart": "restarted",
	"start":   "started",
}

func startPM2Command(p Process, cmdName string) error {
	cmdArgs := []string{
		cmdName,
		p.name(),
	}
	if cmdName == "start" {
		cmdArgs = append(cmdArgs, "--interpreter none", "--")
		cmdArgs = append(cmdArgs, p.args()...)
	}
	cmd := exec.Command(
		"pm2",
		cmdArgs...,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s %s with PM2: %v", cmdName, p.fullname(), err)
	}
	notify.Println(fmt.Sprintf("%s %s with PM2", p.fullname(), notifyCommandNames[cmdName]))
	return nil
}

func shutdownProcess(p Process) error {
	if err := startPM2Command(p, "stop"); err != nil {
		p.state(ErrorState)
		return err
	}
	p.state(StoppedState)
	return nil
}
