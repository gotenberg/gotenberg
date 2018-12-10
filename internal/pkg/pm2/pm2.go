package pm2

import (
	"fmt"
	"os/exec"

	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
)

// Process is a type that can launch or
// shutdown a process with PM2.
type Process interface {
	Launch() error
	Shutdown() error
	getArgs() []string
	getName() string
	getFullname() string
	isViable() bool
	warmup()
}

const maxRestartAttempts int = 5

var humanNames = map[string]string{
	"start":   "started",
	"restart": "restarted",
	"stop":    "stopped",
}

func launch(p Process) error {
	if err := run(p, "start"); err != nil {
		return err
	}
	p.warmup()
	if !p.isViable() {
		attempts := 0
		for attempts < maxRestartAttempts && !p.isViable() {
			run(p, "restart")
			p.warmup()
			attempts++
		}
		if !p.isViable() {
			return fmt.Errorf("failed to launch %s", p.getFullname())
		}
	}
	return nil
}

func shutdown(p Process) error {
	return run(p, "stop")
}

func run(p Process, cmdName string) error {
	cmdArgs := []string{
		cmdName,
		p.getName(),
	}
	if cmdName == "start" {
		cmdArgs = append(cmdArgs, "--interpreter none", "--")
		cmdArgs = append(cmdArgs, p.getArgs()...)
	}
	cmd := exec.Command(
		"pm2",
		cmdArgs...,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s %s with PM2: %v", cmdName, p.getFullname(), err)
	}
	notify.Println(fmt.Sprintf("%s %s with PM2", p.getFullname(), humanNames[cmdName]))
	return nil
}
