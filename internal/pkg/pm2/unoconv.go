package pm2

import (
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/logger"
	"github.com/thecodingmachine/gotenberg/internal/pkg/standarderror"
)

const unoconvWarmupTime = 5 * time.Second

type unoconv struct {
	manager *processManager
}

// NewUnoconv returns a unoconv listener
// process.
func NewUnoconv(logger *logger.Logger) Process {
	return &unoconv{
		manager: &processManager{logger: logger},
	}
}

func (p *unoconv) Fullname() string {
	return "unoconv listener"
}

func (p *unoconv) Start() error {
	const op = "pm2.unoconv.Start"
	if err := p.manager.start(p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

func (p *unoconv) Shutdown() error {
	const op = "pm2.unoconv.Shutdown"
	if err := p.manager.shutdown(p); err != nil {
		return &standarderror.Error{Op: op, Err: err}
	}
	return nil
}

func (p *unoconv) args() []string {
	return []string{
		"--listener",
		"--verbose",
	}
}

func (p *unoconv) name() string {
	return "unoconv"
}

func (p *unoconv) viable() bool {
	// TODO find a way to check if
	// the unoconv listener
	// is correctly started?
	return true
}

func (p *unoconv) warmup() {
	const op = "pm2.unoconv.warmup"
	p.manager.logger.DebugfOp(
		op,
		"allowing %v to startup",
		unoconvWarmupTime,
	)
	time.Sleep(unoconvWarmupTime)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(unoconv))
)
