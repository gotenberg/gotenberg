package process

import (
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type Process interface {
	ID() string
	Host() string
	Port() int
	binary() string
	args() []string
	warmupTime() time.Duration
	viabilityFunc() func(logger xlog.Logger) bool
}

type ListItem struct {
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	Restart int64   `json:"restart"`
	Memory  int64   `json:"memory"`
	CPU     float64 `json:"cpu"`
}

type List []ListItem

type Key string

type Manager interface {
	Start() error
	List() List
	All() []Process
	Processes(key Key) []Process
	Restart(proc Process) error
	Stop(proc Process) error
	IsViable(proc Process) bool
	Memory(proc Process) (int64, error)
}
