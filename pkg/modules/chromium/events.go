package chromium

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

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
