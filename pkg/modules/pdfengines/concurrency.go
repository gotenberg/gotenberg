package pdfengines

import (
	"sync"

	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
)

// defaultMaxConcurrency is the number of PDF files a single stub processes at
// once when --pdfengines-max-concurrency (env PDFENGINES_MAX_CONCURRENCY) is
// not set. Each unit of work forks an external binary (qpdf, pdfcpu, pdftk or
// exiftool), so the ceiling trades wall clock against process count and RSS.
//
// It defaults to one, which processes files exactly as the sequential loops
// this package used to run did. Raising it only ever affects a request that
// carries several files, or one that splits into several outputs: a
// single-file request never reaches the concurrent path at all. Operators who
// send multi-file batches and have the memory headroom opt in.
//
// This never covers LibreOffice. libreoffice-pdfengine implements Convert and
// nothing else, every other [gotenberg.PdfEngine] method on it returns
// [gotenberg.ErrPdfEngineMethodNotSupported], and [ConvertStub] deliberately
// does not use this package's helpers. A soffice instance costs too much
// memory to run several of per container, so LibreOffice throughput is scaled
// by adding Gotenberg containers, not by raising this number.
const defaultMaxConcurrency = 1

// maxFileConcurrency is how many files one request may have in flight at once.
// It is replaced during [PdfEngines.Provision].
var maxFileConcurrency = defaultMaxConcurrency

// engineExtraSlots bounds the concurrency this package ADDS, across the whole
// process rather than per request, and holds one fewer slot than
// [maxFileConcurrency] because every request already owns one unit of its own.
//
// Bounding the added concurrency rather than the total is what keeps the
// ceiling from becoming a throughput regression. The sequential loops this
// helper replaced had no ceiling at all: X concurrent requests ran X engine
// binaries, one apiece. A pool covering the total would cut those X requests
// down to the ceiling, so an operator raising the flag to speed up a single
// multi-file request would slow the server down under real load. Reserving
// each request the unit it always had makes the worst case "what happened
// before, plus at most maxFileConcurrency-1".
//
// A per-request limit would have the opposite failure: X simultaneous requests
// forking X times the limit, trading the timeouts this exists to prevent for
// memory exhaustion.
var engineExtraSlots = make(chan struct{}, defaultMaxConcurrency-1)

// acquireEngineSlot waits for the first unit of capacity to become available,
// either this request's reserved unit or a slot from the shared pool, and
// returns the function that gives it back.
//
// Waiting on both at once is the whole point. Committing to one source and
// blocking on it strands the other: a goroutine parked on an exhausted pool
// cannot pick up its own request's reserved unit when the file before it
// finishes, so the reserved units sit idle while every file queues on the
// pool, which is slower than having no pool at all.
func acquireEngineSlot(ctx *api.Context, reserved chan struct{}) (func(), error) {
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
	case engineExtraSlots <- struct{}{}:
		return func() { <-engineExtraSlots }, nil
	case <-done:
		return nil, ctx.Err()
	}
}

// forEachInputPath runs fn against every input path, up to
// --pdfengines-max-concurrency (env PDFENGINES_MAX_CONCURRENCY) at a time.
//
// The stubs mutate each PDF in place, so distinct input paths never touch the
// same file and may run together. Callers that layer operations on one file,
// like [WatermarkStub] applying several watermarks in order, must keep that
// outer sequence and parallelize only the file dimension.
//
// Every path is attempted even after one fails, and the error returned is the
// first in input order rather than the first to arrive. That keeps the failing
// filename in the error message identical to what the sequential form
// reported, which the integration scenarios assert on.
func forEachInputPath(ctx *api.Context, inputPaths []string, fn func(inputPath string) error) error {
	return forEachInputPathIndexed(ctx, inputPaths, func(_ int, inputPath string) error {
		return fn(inputPath)
	})
}

// forEachInputPathIndexed is [forEachInputPath] with the input path's index,
// for callers collecting a result per file. Writing into a preallocated slice
// at the given index needs no further synchronization; writing into a shared
// map does and must not be done from fn.
func forEachInputPathIndexed(ctx *api.Context, inputPaths []string, fn func(i int, inputPath string) error) error {
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
	if maxFileConcurrency < 2 {
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
	// one-binary-apiece they already got. See [engineExtraSlots].
	reserved := make(chan struct{}, 1)
	reserved <- struct{}{}

	errs := make([]error, len(inputPaths))

	var wg sync.WaitGroup
	for i, inputPath := range inputPaths {
		wg.Go(func() {
			release, err := acquireEngineSlot(ctx, reserved)
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
