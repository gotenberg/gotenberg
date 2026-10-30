package pdf

import (
	"sync"

	"github.com/gotenberg/gotenberg/v9/pkg/modules/api"
)

// defaultMaxConcurrency is the number of PDF documents a request processes at
// once when --pdf-max-concurrency (env PDF_MAX_CONCURRENCY) is not set. Each
// unit of work runs a litho pipeline in process, so the ceiling trades wall
// clock against CPU and memory.
//
// It defaults to one, which processes documents one after the other. Raising
// it only ever affects a request that carries several documents, or one that
// splits into several outputs: a single-document request never reaches the
// concurrent path at all. Operators who send multi-document batches and have
// the headroom opt in.
const defaultMaxConcurrency = 1

// fileConcurrency bounds how many files a [Processor] runs at once.
//
// max is how many files one request may have in flight at once. Below two, a
// request processes its files one after the other.
//
// extraSlots bounds the concurrency this package ADDS, across the whole
// process rather than per request, and holds one fewer slot than max because
// every request already owns one unit of its own.
//
// Bounding the added concurrency rather than the total is what keeps the
// ceiling from becoming a throughput regression. The sequential loops this
// helper replaced had no ceiling at all: X concurrent requests processed X
// documents, one apiece. A pool covering the total would cut those X requests
// down to the ceiling, so an operator raising the flag to speed up a single
// multi-file request would slow the server down under real load. Reserving
// each request the unit it always had makes the worst case "what happened
// before, plus at most max-1".
//
// A per-request limit would have the opposite failure: X simultaneous requests
// forking X times the limit, trading the timeouts this exists to prevent for
// memory exhaustion.
type fileConcurrency struct {
	max        int
	extraSlots chan struct{}
}

// newFileConcurrency returns the [fileConcurrency] of a ceiling of maxFiles,
// or of [defaultMaxConcurrency] when maxFiles is not positive.
func newFileConcurrency(maxFiles int) fileConcurrency {
	if maxFiles < 1 {
		maxFiles = defaultMaxConcurrency
	}

	return fileConcurrency{
		max:        maxFiles,
		extraSlots: make(chan struct{}, maxFiles-1),
	}
}

// acquireSlot waits for the first unit of capacity to become available,
// either this request's reserved unit or a slot from the shared pool, and
// returns the function that gives it back.
//
// Waiting on both at once is the whole point. Committing to one source and
// blocking on it strands the other: a goroutine parked on an exhausted pool
// cannot pick up its own request's reserved unit when the file before it
// finishes, so the reserved units sit idle while every file queues on the
// pool, which is slower than having no pool at all.
func (p *Processor) acquireSlot(ctx *api.Context, reserved chan struct{}) (func(), error) {
	// An [api.Context] carries a request context in production, but one built
	// as a literal, which the unit tests do, embeds a nil [context.Context]
	// and would panic on Done. A nil channel never fires, which correctly
	// leaves the two capacity sources as the only things to wait on.
	var done <-chan struct{}
	if ctx != nil && ctx.Context != nil {
		done = ctx.Done()
	}

	// A select whose cancellation and capacity cases are both ready picks
	// between them at random, so an already-dead request would start more
	// files on a coin flip. Check first and stop taking on work.
	if done != nil {
		select {
		case <-done:
			return nil, ctx.Err()
		default:
		}
	}

	select {
	case <-reserved:
		return func() { reserved <- struct{}{} }, nil
	case p.files.extraSlots <- struct{}{}:
		return func() { <-p.files.extraSlots }, nil
	case <-done:
		return nil, ctx.Err()
	}
}

// forEachInputPathIndexed runs fn against every input path, with its index, up
// to --pdf-max-concurrency (env PDF_MAX_CONCURRENCY) at a time.
//
// Each document runs its own pipeline and writes its own outputs, so distinct
// input paths never touch the same file and may run together. Stages stay in
// order within a document: only the document dimension runs in parallel.
//
// Every path is attempted even after one fails, and the error returned is the
// first in input order rather than the first to arrive. That keeps the failing
// filename in the error message identical to what the sequential form
// reported, which the integration scenarios assert on.
//
// fn receives the input path's index, for callers collecting a result per
// file. Writing into a preallocated slice at the given index needs no further
// synchronization; writing into a shared map does and must not be done from fn.
func (p *Processor) forEachInputPathIndexed(ctx *api.Context, inputPaths []string, fn func(i int, inputPath string) error) error {
	if len(inputPaths) == 0 {
		return nil
	}

	// The common case is a single file. Skip the goroutine and the slot: the
	// caller is already inside whatever bound its own route applies.
	if len(inputPaths) == 1 {
		return fn(0, inputPaths[0])
	}

	// At the default ceiling of one, run the plain sequential loop this helper
	// replaced. Racing goroutines for a single slot would serialize the work
	// just the same, but the order files are picked up in would be down to the
	// scheduler, and every file would be attempted even once one has failed.
	// Taking the old path keeps the default a genuine no-op: same order, same
	// early return, no goroutines.
	if p.files.max < 2 {
		for i, inputPath := range inputPaths {
			err := fn(i, inputPath)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// The unit this request would have had all to itself before any of this
	// existed. Whichever file claims it runs without touching the shared pool,
	// so concurrent requests can never throttle each other below the
	// one-document-apiece they already got. See [fileConcurrency].
	reserved := make(chan struct{}, 1)
	reserved <- struct{}{}

	errs := make([]error, len(inputPaths))

	var wg sync.WaitGroup
	for i, inputPath := range inputPaths {
		wg.Go(func() {
			release, err := p.acquireSlot(ctx, reserved)
			if err != nil {
				errs[i] = err
				return
			}
			defer release()

			errs[i] = fn(i, inputPath)
		})
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}
