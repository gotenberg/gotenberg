package process

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xtime"
)

type jlistItem struct {
	Name   string `json:"name"`
	PM2Env struct {
		Status      string `json:"status"`
		RestartTime int64  `json:"restart_time"`
	} `json:"pm2_env"`
	Monit struct {
		Memory int64   `json:"memory"`
		CPU    float64 `json:"cpu"`
	} `json:"monit"`
}

type jlist []jlistItem

func (list jlist) toList() List {
	var result List
	for _, current := range list {
		item := ListItem{
			Name:    current.Name,
			Status:  current.PM2Env.Status,
			Restart: current.PM2Env.RestartTime,
			Memory:  current.Monit.Memory, // TODO humanize?
			CPU:     current.Monit.CPU,
		}
		result = append(result, item)
	}
	return result
}

func (list jlist) isOnline(p Process) bool {
	const onlineStatus string = "online"
	for _, item := range list {
		if item.Name == p.ID() {
			return item.PM2Env.Status == onlineStatus
		}
	}
	return false
}

func (list jlist) memory(p Process) (int64, error) {
	const op string = "process.jlist.memory"
	for _, item := range list {
		if item.Name == p.ID() {
			return item.Monit.Memory, nil
		}
	}
	return 0, xerror.New(
		op,
		fmt.Errorf("'%s' does not exist in the list of PM2 processes", p.ID()),
	)
}

type command string

const (
	startCommand   command = "start"
	restartCommand command = "restart"
	stopCommand    command = "stop"
	logsCommand    command = "logs"
	jlistCommand   command = "jlist"
)

const maximumRestartAttempts uint = 3

type pm2Manager struct {
	logger   xlog.Logger
	config   conf.Config
	pool     map[Key][]Process
	list     *jlist
	listLock *sync.Mutex
}

// NewPM2Manager returns a PM2 manager.
func NewPM2Manager(logger xlog.Logger, config conf.Config) Manager {
	const op string = "process.NewPM2Manager"
	m := &pm2Manager{
		logger:   logger,
		config:   config,
		pool:     make(map[Key][]Process),
		listLock: &sync.Mutex{},
	}
	if !config.DisableGoogleChrome() {
		processes := make([]Process, 2)
		availablePort := 9222
		// TODO from config
		for i := 0; i < 2; i++ {
			proc := chromeProcess{
				host: "127.0.0.1",
				port: availablePort,
			}
			proc.id = fmt.Sprintf("%s-%d", proc.binary(), proc.port)
			processes[i] = proc
			logger.DebugfOp(op, "added new process %v", proc)
			availablePort++
		}
		m.pool[ChromeKey] = processes
	}
	if !config.DisableUnoconv() {
		processes := make([]Process, 2)
		availablePort := 2002
		// TODO from config
		for i := 0; i < 2; i++ {
			proc := sofficeProcess{
				host: "127.0.0.1",
				port: availablePort,
			}
			proc.id = fmt.Sprintf("%s-%d", proc.binary(), proc.port)
			processes[i] = proc
			logger.DebugfOp(op, "added new process %v", proc)
			availablePort++
		}
		m.pool[SofficeKey] = processes
	}
	// update the manager processes list
	// only if there are processes.
	if !config.DisableGoogleChrome() || !config.DisableUnoconv() {
		go m.jlistTimer()
	}
	return m
}

func (m *pm2Manager) jlistTimer() {
	const op string = "process.pm2Manager.jlistTimer"
	duration := xtime.Duration(10)
	resolver := func() error {
		m.listLock.Lock()
		defer m.listLock.Unlock()
		out, err := exec.
			Command("pm2", string(jlistCommand)).
			Output()
		if err != nil {
			return err
		}
		data := &jlist{}
		if err := json.Unmarshal(out, data); err != nil {
			return err
		}
		m.list = data
		return nil
	}
	// update every x seconds the
	// list from the manager.
	for range time.Tick(duration) {
		if err := resolver(); err != nil {
			m.logger.ErrorOp(op, xerror.New(op, err))
		}
	}
}

func (m *pm2Manager) Start() error {
	const op string = "process.pm2Manager.Start"
	for _, processes := range m.pool {
		for _, proc := range processes {
			if err := m.start(proc); err != nil {
				return xerror.New(op, err)
			}
		}
	}
	return nil
}

func (m *pm2Manager) start(p Process) error {
	const op string = "process.pm2Manager.start"
	resolver := func() error {
		// first, we try to start the process.
		if err := m.run(startCommand, p); err != nil {
			return err
		}
		// we wait the process to be ready.
		m.warmup(p)
		// if the process failed to start correctly,
		// we have to restart it.
		if !m.IsViable(p) && maximumRestartAttempts > 0 {
			return m.Restart(p)
		}
		// the process is viable, let's log its
		// output.
		return m.logs(p)
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2Manager) List() List {
	m.listLock.Lock()
	defer m.listLock.Unlock()
	return m.list.toList()
}

func (m *pm2Manager) All() []Process {
	var result []Process
	for _, processes := range m.pool {
		result = append(result, processes...)
	}
	return result
}

func (m *pm2Manager) Processes(key Key) []Process {
	return m.pool[key]
}

func (m *pm2Manager) Restart(p Process) error {
	const op string = "process.pm2Manager.Restart"
	resolver := func() error {
		var attempts uint
		for attempts < maximumRestartAttempts {
			// we restart the process.
			if err := m.run(restartCommand, p); err != nil {
				return err
			}
			// we wait the process to be ready.
			m.warmup(p)
			attempts++
			// if the process is viable, we
			// leave.
			if m.IsViable(p) {
				return m.logs(p)
			}
		}
		return fmt.Errorf("failed to start '%s'", p.ID())
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2Manager) Stop(p Process) error {
	const op string = "process.pm2Manager.Stop"
	if err := m.run(stopCommand, p); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2Manager) IsViable(p Process) bool {
	if !p.viabilityFunc()(m.logger) {
		return false
	}
	m.listLock.Lock()
	defer m.listLock.Unlock()
	return m.list.isOnline(p)
}

func (m *pm2Manager) Memory(p Process) (int64, error) {
	const op string = "process.pm2Manager.Memory"
	m.listLock.Lock()
	defer m.listLock.Unlock()
	memory, err := m.list.memory(p)
	if err != nil {
		return 0, xerror.New(op, err)
	}
	return memory, nil
}

func (m *pm2Manager) warmup(p Process) {
	const op string = "process.pm2Manager.warmup"
	warmupTime := p.warmupTime()
	m.logger.DebugfOp(
		op,
		"waiting '%v' for allowing '%s' to warmup",
		warmupTime,
		p.ID(),
	)
	time.Sleep(warmupTime)
}

func (m *pm2Manager) logs(p Process) error {
	const op string = "process.pm2Manager.logs"
	if m.config.LogLevel() == xlog.DebugLevel {
		if err := m.run(logsCommand, p); err != nil {
			return xerror.New(op, err)
		}
	}
	return nil
}

func (m *pm2Manager) run(pm2Cmd command, p Process) error {
	const op string = "process.pm2Manager.run"
	resolver := func() error {
		args := []string{
			string(pm2Cmd),
			p.binary(),
		}
		if pm2Cmd == startCommand {
			args = append(args, fmt.Sprintf("--name=%s", p.ID()))
			args = append(args, "--interpreter=none", "--")
			args = append(args, p.args()...)
		}
		cmd, err := xexec.Command(m.logger, "pm2", args...)
		if err != nil {
			return err
		}
		xexec.LogBeforeExecute(m.logger, cmd)
		return cmd.Start()
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Manager(new(pm2Manager))
)
