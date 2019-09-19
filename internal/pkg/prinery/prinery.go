package prinery

import (
	"context"
	"fmt"

	"github.com/thecodingmachine/gotenberg/internal/pkg/print"
	"github.com/thecodingmachine/gotenberg/internal/pkg/process"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type request struct {
	ctx    context.Context
	logger xlog.Logger
	print  print.Print
	dest   string
	result chan error
}

type worker struct {
	work chan request
	proc process.Process
}

func (w *worker) do(done chan *worker) {
	for {
		req := <-w.work
		req.result <- req.print.Print(req.ctx, req.dest, w.proc)
		done <- w
	}
}

type Prinery struct {
	logger  xlog.Logger
	manager process.Manager
	work    chan request
	pool    chan *worker
	done    chan *worker
}

func New(logger xlog.Logger, manager process.Manager, key process.Key) (*Prinery, error) {
	const op string = "prinery.New"
	processes := manager.Processes(key)
	nWorkers := len(processes)
	if nWorkers == 0 {
		err := fmt.Errorf("no processes found for key '%s'", string(key))
		return nil, xerror.New(op, err)
	}
	logger.DebugfOp(op, "found '%d' processes for key '%s'", nWorkers, string(key))
	work := make(chan request, 1)
	pool := make(chan *worker, nWorkers)
	done := make(chan *worker, nWorkers)
	for _, p := range processes {
		w := &worker{
			work: work,
			proc: p,
		}
		pool <- w
		go w.do(done)
	}
	return &Prinery{
		logger:  logger,
		manager: manager,
		work:    work,
		pool:    pool,
		done:    done,
	}, nil
}

func (p *Prinery) PrintRequest(ctx context.Context, logger xlog.Logger, prnt print.Print, dest string) error {
	const op string = "prinery.Prinery.PrintRequest"
	req := request{
		ctx:    ctx,
		logger: logger,
		print:  prnt,
		dest:   dest,
		result: make(chan error),
	}
	p.dispatch(req)
	err := <-req.result
	if err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (p *Prinery) Start() {
	for {
		select {
		case req := <-p.work:
			p.dispatch(req)
		case w := <-p.done:
			p.completed(w)
		}
	}
}

func (p *Prinery) dispatch(req request) {
	const op string = "prinery.Prinery.dispatch"
	select {
	case w := <-p.pool:
		w.work <- req
	case <-req.ctx.Done():
		req.result <- xerror.New(op, req.ctx.Err())
	}
}

func (p *Prinery) completed(w *worker) {
	const op string = "prinery.Prinerty.completed"
	go func() {
		// check process viability.
		isViable := p.manager.IsViable(w.proc)
		// check process memory usage.
		// TODO handle error.
		memory, _ := p.manager.Memory(w.proc)
		p.logger.DebugfOp(op, "%s: isViable = %t, memory = %d", w.proc.ID(), isViable, memory)
		// TODO manage viability and memory usage.
		// pushing back the worker.
		p.pool <- w
	}()

}
