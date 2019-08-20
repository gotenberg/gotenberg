package pm2

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

// Process is a type that can start or
// stop a process with PM2.
type Process interface {
	Fullname() string
	Start() error
	IsViable() bool
	Stop() error
	args() []string
	binary() string
	warmup()
}

type pm2Command string

const (
	startCommand   pm2Command = "start"
	restartCommand pm2Command = "restart"
	stopCommand    pm2Command = "stop"
	logsCommand    pm2Command = "logs"
	jlistCommand   pm2Command = "jlist"
)

/*
JListItem is a struct
used for unmarshaling
ONE item of the result
of the command "pm2 jlist".
*/
type JListItem struct {
	Name   string `json:"name"`
	PM2Env struct {
		Status string `json:"status"`
	} `json:"pm2_env"`
	Monit struct {
		Memory int64   `json:"memory"`
		CPU    float64 `json:"cpu"`
	} `json:"monit"`
}

/*
JList is a struct
used for unmarshaling
the result of the command
"pm2 jlist".
*/
type JList []JListItem

func (list JList) isOnline(p Process) bool {
	const onlineStatus string = "online"
	for _, item := range list {
		if item.Name == p.binary() {
			return item.PM2Env.Status == onlineStatus
		}
	}
	return false
}

// List returns the
// processes details.
func List() (*JList, error) {
	const op = "pm2.List"
	resolver := func() (*JList, error) {
		out, err := exec.
			Command("pm2", string(jlistCommand)).
			Output()
		if err != nil {
			return nil, err
		}
		data := &JList{}
		if err := json.Unmarshal(out, data); err != nil {
			return nil, err
		}
		return data, nil
	}
	list, err := resolver()
	if err != nil {
		return nil, xerror.New(op, err)
	}
	return list, nil
}

func start(logger xlog.Logger, process Process) error {
	const (
		op              string = "pm2.start"
		maximumAttempts int    = 3
	)
	resolver := func() error {
		// first, we try to start the process.
		if err := run(logger, startCommand, process); err != nil {
			return err
		}
		// we wait the process to be ready.
		process.warmup()
		// if  the process failed to start correctly,
		// we have to restart it.
		if !process.IsViable() {
			attempts := 0
			for attempts < maximumAttempts && !process.IsViable() {
				if err := run(logger, restartCommand, process); err != nil {
					return err
				}
				process.warmup()
				attempts++
			}
			if !process.IsViable() {
				return fmt.Errorf("failed to start '%s'", process.Fullname())
			}
		}
		// the process is viable, let's log its
		// output.
		if err := run(logger, logsCommand, process); err != nil {
			return err
		}
		return nil
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func stop(logger xlog.Logger, process Process) error {
	const op string = "pm2.stop"
	if err := run(logger, stopCommand, process); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func run(logger xlog.Logger, pm2Cmd pm2Command, process Process) error {
	const op string = "pm2.run"
	resolver := func() error {
		args := []string{
			string(pm2Cmd),
			process.binary(),
		}
		if pm2Cmd == startCommand {
			args = append(args, "--interpreter=none", "--")
			args = append(args, process.args()...)
		}
		cmd, err := xexec.Command(logger, "pm2", args...)
		if err != nil {
			return err
		}
		xexec.LogBeforeExecute(logger, cmd)
		return cmd.Start()
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}
