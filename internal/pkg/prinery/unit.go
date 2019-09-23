package prinery

import (
	"context"
	"errors"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
	"github.com/thecodingmachine/gotenberg/internal/pkg/xlog"
)

type request struct {
	ctx     context.Context
	logger  xlog.Logger
	printer printer
	dest    string
	result  chan error
}

type worker struct {
	work chan request
	spec processSpec
}

func (w *worker) do(done chan *worker) {
	for {
		req := <-w.work
		req.result <- req.printer.print(req.ctx, w.spec, req.dest)
		done <- w
	}
}

type unit struct {
	logger  xlog.Logger
	workers []*worker
	work    chan request
	pool    chan *worker
	done    chan *worker
}

func newUnit(logger xlog.Logger, specs []processSpec) (*unit, error) {
	const op string = "prinery.newUnit"
	resolver := func() (*unit, error) {
		nWorkers := len(specs)
		if nWorkers == 0 {
			return nil, errors.New("no workers to create")
		}
		logger.DebugfOp(op, "creating '%d' workers...", nWorkers)
		workers := make([]*worker, nWorkers)
		work := make(chan request, 1)
		pool := make(chan *worker, nWorkers)
		done := make(chan *worker, nWorkers)
		for i, spec := range specs {
			w := &worker{
				work: work,
				spec: spec,
			}
			logger.DebugfOp(op, "worker '%s' created", w.spec.id())
			workers[i] = w
		}
		return &unit{
			logger:  logger,
			workers: workers,
			work:    work,
			pool:    pool,
			done:    done,
		}, nil
	}
	u, err := resolver()
	if err != nil {
		return nil, xerror.New(op, err)
	}
	return u, nil
}

func (u *unit) start(emergency chan error) {
	const op string = "prinery.unit.start"
	for _, w := range u.workers {
		u.logger.DebugfOp(op, "starting worker '%s'...", w.spec.id())
		go w.do(u.done)
		u.pool <- w
	}
	for {
		select {
		case req := <-u.work:
			u.dispatch(req)
		case w := <-u.done:
			u.completed(w, emergency)
		}
	}
}

func (u *unit) request(ctx context.Context, logger xlog.Logger, printer printer, dest string) error {
	const op string = "prinery.unit.request"
	req := request{
		ctx:     ctx,
		logger:  logger,
		printer: printer,
		dest:    dest,
		result:  make(chan error),
	}
	u.dispatch(req)
	err := <-req.result
	if err != nil {
		return xerror.New(op, err)
	}
	return nil
}

func (u *unit) dispatch(req request) {
	const op string = "prinery.unit.dispatch"
	select {
	case w := <-u.pool:
		u.logger.DebugfOp(op, "worker '%s' is now in use", w.spec.id())
		w.work <- req
	case <-req.ctx.Done():
		req.result <- xerror.New(op, req.ctx.Err())
	}
}

func (u *unit) completed(w *worker, emergency chan error) {
	const op string = "prinery.unit.completed"
	// TODO check viability, handles errors via emergency chan.
	u.logger.DebugfOp(op, "worker '%s' is ready for a new request", w.spec.id())
	u.pool <- w
}
