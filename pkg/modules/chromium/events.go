package chromium

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/dlclark/regexp2"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// listenForEventRequestPaused listens for requests to check if they are
// allowed or not.
func listenForEventRequestPaused(ctx context.Context, logger *zap.Logger, allowList *regexp2.Regexp, denyList *regexp2.Regexp) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				logger.Debug(fmt.Sprintf("event EventRequestPaused fired for '%s'", e.Request.URL))
				allow := true

				deadline, ok := ctx.Deadline()
				if !ok {
					logger.Error("context has no deadline, cannot filter URL")
					return
				}

				err := gotenberg.FilterDeadline(allowList, denyList, e.Request.URL, deadline)
				if err != nil {
					logger.Warn(err.Error())
					allow = false
				}

				cctx := chromedp.FromContext(ctx)
				executorCtx := cdp.WithExecutor(ctx, cctx.Target)

				if allow {
					req := fetch.ContinueRequest(e.RequestID)
					err = req.Do(executorCtx)
					if err != nil {
						logger.Error(fmt.Sprintf("continue request: %s", err))
					}
					return
				}

				req := fetch.FailRequest(e.RequestID, network.ErrorReasonAccessDenied)
				err = req.Do(executorCtx)
				if err != nil {
					logger.Error(fmt.Sprintf("fail request: %s", err))
				}
			}()
		}
	})
}

// listenForEventResponseReceived listens for an invalid HTTP status code is
// returned by the main page.
// See https://github.com/gotenberg/gotenberg/issues/613.
func listenForEventResponseReceived(ctx context.Context, logger *zap.Logger, url string, failOnHttpStatusCodes []int64, invalidHttpStatusCode *error, invalidHttpStatusCodeMu *sync.RWMutex) {
	for _, code := range []int64{199, 299, 399, 499, 599} {
		if slices.Contains(failOnHttpStatusCodes, code) {
			for i := code - 99; i <= code; i++ {
				failOnHttpStatusCodes = append(failOnHttpStatusCodes, i)
			}
		}
	}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			if ev.Response.URL != url {
				return
			}

			logger.Debug(fmt.Sprintf("event EventResponseReceived fired for main page: %+v", ev.Response))

			if slices.Contains(failOnHttpStatusCodes, ev.Response.Status) {
				invalidHttpStatusCodeMu.Lock()
				defer invalidHttpStatusCodeMu.Unlock()

				*invalidHttpStatusCode = fmt.Errorf("%d: %s", ev.Response.Status, ev.Response.StatusText)
			}
		}
	})
}

// listenForEventExceptionThrown listens for exceptions in the console and
// appends those exceptions to the given error pointer.
// See https://github.com/gotenberg/gotenberg/issues/262.
func listenForEventExceptionThrown(ctx context.Context, logger *zap.Logger, consoleExceptions *error, consoleExceptionsMu *sync.RWMutex) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventExceptionThrown:
			logger.Debug(fmt.Sprintf("event EventExceptionThrown fired: %+v", ev.ExceptionDetails))

			consoleExceptionsMu.Lock()
			defer consoleExceptionsMu.Unlock()

			*consoleExceptions = multierr.Append(*consoleExceptions, fmt.Errorf("\n%+v", ev.ExceptionDetails))
		}
	})
}

// waitForEventDomContentEventFired waits until the event DomContentEventFired
// is fired or the context timeout.
func waitForEventDomContentEventFired(ctx context.Context, logger *zap.Logger) func() error {
	return func() error {
		ch := make(chan struct{})
		cctx, cancel := context.WithCancel(ctx)
		chromedp.ListenTarget(cctx, func(ev interface{}) {
			switch ev.(type) {
			case *page.EventDomContentEventFired:
				cancel()
				close(ch)
			}
		})

		select {
		case <-ch:
			logger.Debug("event DomContentEventFired fired")
			return nil
		case <-ctx.Done():
			return fmt.Errorf("wait for event DomContentEventFired: %w", ctx.Err())
		}
	}
}

// waitForEventLoadEventFired waits until the event LoadEventFired is fired or
// the context timeout.
func waitForEventLoadEventFired(ctx context.Context, logger *zap.Logger) func() error {
	return func() error {
		ch := make(chan struct{})
		cctx, cancel := context.WithCancel(ctx)
		chromedp.ListenTarget(cctx, func(ev interface{}) {
			switch ev.(type) {
			case *page.EventLoadEventFired:
				cancel()
				close(ch)
			}
		})

		select {
		case <-ch:
			logger.Debug("event LoadEventFired fired")
			return nil
		case <-ctx.Done():
			return fmt.Errorf("wait for event LoadEventFired: %w", ctx.Err())
		}
	}
}

// waitForEventNetworkIdle waits until the event networkIdle is fired or the
// context timeout.
func waitForEventNetworkIdle(ctx context.Context, logger *zap.Logger) func() error {
	return func() error {
		ch := make(chan struct{})
		cctx, cancel := context.WithCancel(ctx)
		chromedp.ListenTarget(cctx, func(ev interface{}) {
			switch e := ev.(type) {
			case *page.EventLifecycleEvent:
				if e.Name == "networkIdle" {
					cancel()
					close(ch)
				}
			}
		})

		select {
		case <-ch:
			logger.Debug("event networkIdle fired")
			return nil
		case <-ctx.Done():
			return fmt.Errorf("wait for event networkIdle: %w", ctx.Err())
		}
	}
}

// waitForEventLoadingFinished waits until the event LoadingFinished is fired
// or the context timeout.
func waitForEventLoadingFinished(ctx context.Context, logger *zap.Logger) func() error {
	return func() error {
		ch := make(chan struct{})
		cctx, cancel := context.WithCancel(ctx)
		chromedp.ListenTarget(cctx, func(ev interface{}) {
			switch ev.(type) {
			case *network.EventLoadingFinished:
				cancel()
				close(ch)
			}
		})

		select {
		case <-ch:
			logger.Debug("event LoadingFinished fired")
			return nil
		case <-ctx.Done():
			return fmt.Errorf("wait for event LoadingFinished: %w", ctx.Err())
		}
	}
}

// runBatch runs all functions simultaneously and waits until all of them are
// completed or an error is encountered.
func runBatch(ctx context.Context, fn ...func() error) error {
	eg, _ := errgroup.WithContext(ctx)
	for _, f := range fn {
		eg.Go(f)
	}

	return eg.Wait()
}
