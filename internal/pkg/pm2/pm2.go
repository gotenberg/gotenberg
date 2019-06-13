package pm2

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/notify"
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
	verbose        bool
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
	m.notifyf("executing command '%v'", strings.Join(cmd.Args, " "))
	if m.verbose {
		chromeStdErr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed getting Chrome stderr: %v", err)
		}
		chromeStdOut, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed getting Chrome stdout: %v", err)
		}
		readFromPipe := func(name string, reader io.ReadCloser) {
			r := bufio.NewReader(reader)
			defer reader.Close()
			for {
				line, _, err := r.ReadLine()
				if err != nil {
					if err != io.EOF {
						m.notifyf("error reading from %v for process %v", name, p.name())
					}
					break
				}
				if len(line) != 0 {
					m.notifyf("%v %v: %s", p.name(), name, string(line))
				}
			}
		}
		go readFromPipe("stdout", chromeStdOut)
		go readFromPipe("stderr", chromeStdErr)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s %s with PM2: %v", cmdName, p.Fullname(), err)
	}
	return nil
}

func (m *processManager) notifyf(format string, args ...interface{}) {
	if m.verbose {
		notify.Printf(fmt.Sprintf("%v: %s", time.Now().Format(time.RFC3339), format), args...)
	}
}
