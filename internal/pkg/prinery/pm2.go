package prinery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thecodingmachine/gotenberg/internal/pkg/conf"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xexec"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type command string

const (
	startCommand   command = "start"
	restartCommand command = "restart"
	stopCommand    command = "stop"
	logsCommand    command = "logs"
	jlistCommand   command = "jlist"
)

const maximumRestartAttempts uint = 3

type pm2 struct {
	logger      xlog.Logger
	config      conf.Config
	chromeUnit  *unit
	sofficeUnit *unit
	processes   []process
}

func NewPM2Prinery(logger xlog.Logger, config conf.Config) (Prinery, error) {
	const op string = "prinery.NewPM2Prinery"
	resolver := func() (*pm2, error) {
		m := &pm2{
			logger: logger,
			config: config,
		}
		chromeProcesses := newChromeProcesses(config.GoogleChromeInstances())
		if len(chromeProcesses) > 0 {
			unit, err := newUnit(logger, processesToSpecs(chromeProcesses))
			if err != nil {
				return nil, err
			}
			m.chromeUnit = unit
		}
		sofficeProcesses := newSofficesProcesses(config.SofficeInstances())
		if len(sofficeProcesses) > 0 {
			unit, err := newUnit(logger, processesToSpecs(sofficeProcesses))
			if err != nil {
				return nil, err
			}
			m.sofficeUnit = unit
		}
		m.processes = append(chromeProcesses, sofficeProcesses...)
		return m, nil
	}
	p, err := resolver()
	if err != nil {
		return nil, xerror.New(op, err)
	}
	return p, nil
}

func (m *pm2) Start(emergency chan error) error {
	const op string = "prinery.pm2.Start"
	// start processes.
	// those lines "works" but processes
	// fail to start...
	/*var wg sync.WaitGroup
	result := make(chan error, len(m.processes))
	for _, proc := range m.processes {
		wg.Add(1)
		go func(proc process, result chan error) {
			result <- m.start(proc)
			wg.Done()
		}(proc, result)
	}
	wg.Wait()
	close(result)
	for err := range result {
		if err != nil {
			return xerror.New(op, err)
		}
	}*/
	for _, proc := range m.processes {
		if err := m.start(proc); err != nil {
			return xerror.New(op, err)
		}
	}
	// start units.
	if m.chromeUnit != nil {
		go m.chromeUnit.start(emergency)
	}
	if m.sofficeUnit != nil {
		go m.sofficeUnit.start(emergency)
	}
	return nil
}

func (m *pm2) HTML(
	ctx context.Context,
	logger xlog.Logger,
	dest, fpath string,
	opts ChromePrintOptions,
) error {
	const op string = "prinery.pm2.HTML"
	resolver := func() error {
		if m.chromeUnit == nil {
			return errors.New("cannot handle HTML print request as there is no chromeUnit")
		}
		URL := fmt.Sprintf("file://%s", fpath)
		p := chromePrinter{
			logger: logger,
			url:    URL,
			opts:   opts,
		}
		req := request{
			ctx:     ctx,
			logger:  logger,
			printer: p,
			dest:    dest,
			result:  make(chan error),
		}
		m.chromeUnit.dispatch(req)
		err := <-req.result
		return err
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2) URL(
	ctx context.Context,
	logger xlog.Logger, dest, URL string,
	opts ChromePrintOptions,
) error {
	const op string = "prinery.pm2.URL"
	resolver := func() error {
		if m.chromeUnit == nil {
			return errors.New("cannot handle URL print request as there is no chromeUnit")
		}
		p := chromePrinter{
			logger: logger,
			url:    URL,
			opts:   opts,
		}
		req := request{
			ctx:     ctx,
			logger:  logger,
			printer: p,
			dest:    dest,
			result:  make(chan error),
		}
		m.chromeUnit.dispatch(req)
		err := <-req.result
		return err
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2) Markdown(
	ctx context.Context,
	logger xlog.Logger,
	dest, fpath string,
	opts ChromePrintOptions,
) error {
	const op string = "prinery.pm2.Markdown"
	resolver := func() error {
		if m.chromeUnit == nil {
			return errors.New("cannot handle Markdown print request as there is no chromeUnit")
		}
		p, err := newMarkdownPrinter(logger, fpath, opts)
		if err != nil {
			return err
		}
		req := request{
			ctx:     ctx,
			logger:  logger,
			printer: p,
			dest:    dest,
			result:  make(chan error),
		}
		m.chromeUnit.dispatch(req)
		err = <-req.result
		return err
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2) Office(
	ctx context.Context,
	logger xlog.Logger,
	dest string,
	fpaths []string,
	opts UnoconvPrintOptions,
) error {
	const op string = "prinery.pm2.Office"
	resolver := func() error {
		if m.sofficeUnit == nil {
			return errors.New("cannot handle Office print request as there is no sofficeUnit")
		}
		p := unoconvPrinter{
			logger: logger,
			fpaths: fpaths,
			opts:   opts,
		}
		req := request{
			ctx:     ctx,
			logger:  logger,
			printer: p,
			dest:    dest,
			result:  make(chan error),
		}
		m.sofficeUnit.dispatch(req)
		err := <-req.result
		return err
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2) Merge(
	ctx context.Context,
	logger xlog.Logger,
	dest string,
	fpaths []string,
) error {
	const op string = "prinery.pm2.Merge"
	p := mergePrinter{
		logger: logger,
		fpaths: fpaths,
	}
	if err := p.print(ctx, nil, dest); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2) start(p process) error {
	const op string = "prinery.pm2.start"
	resolver := func() error {
		// first, we try to start the process.
		if err := m.run(startCommand, p); err != nil {
			return err
		}
		// we wait the process to be ready.
		m.warmup(p)
		// if the process failed to start correctly,
		// we have to restart it.
		if !m.isViable(p) && maximumRestartAttempts > 0 {
			return m.restart(p)
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

func (m *pm2) restart(p process) error {
	const op string = "prinery.pm2.restart"
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
			if m.isViable(p) {
				return m.logs(p)
			}
		}
		return fmt.Errorf("failed to start '%s'", p.spec().id())
	}
	if err := resolver(); err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (m *pm2) isViable(p process) bool {
	if !p.viabilityFunc()(m.logger) {
		return false
	}
	/*m.listLock.Lock()
	defer m.listLock.Unlock()
	return m.list.isOnline(p)*/
	return true
}

func (m *pm2) warmup(p process) {
	const op string = "prinery.pm2.warmup"
	warmupTime := p.warmupTime()
	m.logger.DebugfOp(
		op,
		"waiting '%v' for allowing '%s' to warmup",
		warmupTime,
		p.spec().id(),
	)
	time.Sleep(warmupTime)
}

func (m *pm2) logs(p process) error {
	const op string = "prinery.pm2.logs"
	if m.config.LogLevel() == xlog.DebugLevel {
		if err := m.run(logsCommand, p); err != nil {
			return xerror.New(op, err)
		}
	}
	return nil
}

func (m *pm2) run(pm2Cmd command, p process) error {
	const op string = "prinery.pm2.run"
	resolver := func() error {
		args := []string{
			string(pm2Cmd),
		}
		if pm2Cmd == startCommand {
			args = append(args, p.binary())
			args = append(args, fmt.Sprintf("--name=%s", p.spec().id()))
			args = append(args, "--interpreter=none", "--")
			args = append(args, p.args()...)
		} else {
			args = append(args, p.spec().id())
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
	_ = Prinery(new(pm2))
)
