package pm2

import (
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type unoconvProcess struct {
	logger xlog.Logger
}

// NewUnoconvProcess returns a unoconv listener
// process.
func NewUnoconvProcess(logger xlog.Logger) Process {
	return unoconvProcess{
		logger: logger,
	}
}

func (p unoconvProcess) Fullname() string {
	return "unoconv listener"
}

func (p unoconvProcess) Start() error {
	const op string = "pm2.unoconvProcess.Start"
	if err := start(p.logger, p); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p unoconvProcess) IsViable() bool {
	// TODO find a way to check if
	// the unoconv listener
	// is correctly started?
	return true
}

func (p unoconvProcess) Stop() error {
	const op string = "pm2.unoconvProcess.Stop"
	if err := stop(p.logger, p); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p unoconvProcess) args() []string {
	return []string{
		"--listener",
		"--verbose",
	}
}

func (p unoconvProcess) binary() string {
	return "unoconv"
}

func (p unoconvProcess) warmup() {
	const (
		op         string        = "pm2.unoconvProcess.warmup"
		warmupTime time.Duration = 3 * time.Second
	)
	p.logger.DebugfOp(
		op,
		"waiting '%v' for allowing '%s' to warmup",
		warmupTime,
		p.Fullname(),
	)
	time.Sleep(warmupTime)
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = Process(new(unoconvProcess))
)
