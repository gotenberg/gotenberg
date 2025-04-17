package chromium

import (
	"context"
	"fmt"
	"net/http"
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

type eventRequestPausedOptions struct {
	allowList, denyList *regexp2.Regexp
	extraHttpHeaders    []ExtraHttpHeader
}

// listenForEventRequestPaused listens for requests to check if they are
// allowed or not.  It also set the extra HTTP headers, if any.
// See https://github.com/gotenberg/gotenberg/issues/1011.
// TODO: https://chromedevtools.github.io/devtools-protocol/tot/Network/#method-setBlockedURLs (experimental for now).
func listenForEventRequestPaused(ctx context.Context, logger *zap.Logger, options eventRequestPausedOptions) {
	if len(options.extraHttpHeaders) == 0 {
		logger.Debug("no extra HTTP headers")
	} else {
		logger.Debug(fmt.Sprintf("extra HTTP headers: %+v", options.extraHttpHeaders))
	}

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

				err := gotenberg.FilterDeadline(options.allowList, options.denyList, e.Request.URL, deadline)
				if err != nil {
					logger.Warn(err.Error())
					allow = false
				}

				cctx := chromedp.FromContext(ctx)
				executorCtx := cdp.WithExecutor(ctx, cctx.Target)

				if !allow {
					req := fetch.FailRequest(e.RequestID, network.ErrorReasonAccessDenied)
					err = req.Do(executorCtx)
					if err != nil {
						logger.Error(fmt.Sprintf("fail request: %s", err))
					}
					return
				}

				req := fetch.ContinueRequest(e.RequestID)

				var extraHttpHeadersToSet []ExtraHttpHeader
				if len(options.extraHttpHeaders) > 0 {
					// The user wants to set extra HTTP headers.

					// First, we have to check if at least one header has to be
					// set for the current request.
					for _, header := range options.extraHttpHeaders {
						if header.Scope == nil {
							// Non-scoped header.
							logger.Debug(fmt.Sprintf("extra HTTP header '%s' will be set for request URL '%s'", header.Name, e.Request.URL))
							extraHttpHeadersToSet = append(extraHttpHeadersToSet, header)
							continue
						}

						ok, err := header.Scope.MatchString(e.Request.URL)
						if err != nil {
							logger.Error(fmt.Sprintf("fail to match extra HTTP header '%s' scope with URL '%s': %s", header.Name, e.Request.URL, err))
						} else if ok {
							logger.Debug(fmt.Sprintf("extra HTTP header '%s' (scoped) will be set for request URL '%s'", header.Name, e.Request.URL))
							extraHttpHeadersToSet = append(extraHttpHeadersToSet, header)
						} else {
							logger.Debug(fmt.Sprintf("scoped extra HTTP header '%s' (scoped) will not be set for request URL '%s'", header.Name, e.Request.URL))
						}
					}
				}

				if len(extraHttpHeadersToSet) > 0 {
					logger.Debug(fmt.Sprintf("setting extra HTTP headers for request URL '%s': %+v", e.Request.URL, extraHttpHeadersToSet))

					originalHeaders := e.Request.Headers
					headers := make(map[string]string)

					for key, value := range originalHeaders {
						strValue, ok := value.(string)
						if ok {
							headers[key] = strValue
						} else {
							logger.Error(fmt.Sprintf("ignoring header '%s' for URL '%s' since it cannot be cast to a string", key, e.Request.URL))
						}
					}

					var headersEntries []*fetch.HeaderEntry
					for key, value := range headers {
						headersEntries = append(headersEntries, &fetch.HeaderEntry{
							Name:  key,
							Value: value,
						})
					}
					for _, header := range extraHttpHeadersToSet {
						headersEntries = append(headersEntries, &fetch.HeaderEntry{
							Name:  header.Name,
							Value: header.Value,
						})
					}

					req.Headers = headersEntries
				}

				err = req.Do(executorCtx)
				if err != nil {
					logger.Error(fmt.Sprintf("continue request: %s", err))
				}
			}()
		}
	})
}

type eventResponseReceivedOptions struct {
	mainPageUrl                     string
	failOnHttpStatusCodes           []int64
	invalidHttpStatusCode           *error
	invalidHttpStatusCodeMu         *sync.RWMutex
	failOnResourceOnHttpStatusCode  []int64
	invalidResourceHttpStatusCode   *error
	invalidResourceHttpStatusCodeMu *sync.RWMutex
}

// listenForEventResponseReceived listens for an invalid HTTP status code
// returned by the main page or by one or more resources.
// See:
// https://github.com/gotenberg/gotenberg/issues/613.
// https://github.com/gotenberg/gotenberg/issues/1021.
func listenForEventResponseReceived(
	ctx context.Context,
	logger *zap.Logger,
	options eventResponseReceivedOptions,
) {
	for _, code := range []int64{199, 299, 399, 499, 599} {
		if slices.Contains(options.failOnHttpStatusCodes, code) {
			for i := code - 99; i <= code; i++ {
				options.failOnHttpStatusCodes = append(options.failOnHttpStatusCodes, i)
			}
		}

		if slices.Contains(options.failOnResourceOnHttpStatusCode, code) {
			for i := code - 99; i <= code; i++ {
				options.failOnResourceOnHttpStatusCode = append(options.failOnResourceOnHttpStatusCode, i)
			}
		}
	}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			if ev.Response.URL == options.mainPageUrl {
				logger.Debug(fmt.Sprintf("event EventResponseReceived fired for main page: %+v", ev.Response))

				if slices.Contains(options.failOnHttpStatusCodes, ev.Response.Status) {
					options.invalidHttpStatusCodeMu.Lock()
					defer options.invalidHttpStatusCodeMu.Unlock()

					*options.invalidHttpStatusCode = fmt.Errorf("%d: %s", ev.Response.Status, ev.Response.StatusText)
				}

				return
			}

			logger.Debug(fmt.Sprintf("event EventResponseReceived fired for a resource: %+v", ev.Response))

			if slices.Contains(options.failOnResourceOnHttpStatusCode, ev.Response.Status) {
				options.invalidResourceHttpStatusCodeMu.Lock()
				defer options.invalidResourceHttpStatusCodeMu.Unlock()

				*options.invalidResourceHttpStatusCode = multierr.Append(
					*options.invalidResourceHttpStatusCode,
					fmt.Errorf("%s - %d: %s", ev.Response.URL, ev.Response.Status, http.StatusText(int(ev.Response.Status))),
				)
			}
		}
	})
}

type eventLoadingFailedOptions struct {
	loadingFailed           *error
	loadingFailedMu         *sync.RWMutex
	resourceLoadingFailed   *error
	resourceLoadingFailedMu *sync.RWMutex
}

// listenForEventLoadingFailed listens for an event indicating that the main
// page or one or more resources failed to load.
// See:
// https://github.com/gotenberg/gotenberg/issues/913.
// https://github.com/gotenberg/gotenberg/issues/959.
// https://github.com/gotenberg/gotenberg/issues/1021.
func listenForEventLoadingFailed(ctx context.Context, logger *zap.Logger, options eventLoadingFailedOptions) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventLoadingFailed:
			logger.Debug(fmt.Sprintf("event EventLoadingFailed fired: %+v", ev.ErrorText))

			// We are looking for common errors.
			// TODO: sufficient?
			errors := []string{
				"net::ERR_CONNECTION_CLOSED",
				"net::ERR_CONNECTION_RESET",
				"net::ERR_CONNECTION_REFUSED",
				"net::ERR_CONNECTION_ABORTED",
				"net::ERR_CONNECTION_FAILED",
				"net::ERR_NAME_NOT_RESOLVED",
				"net::ERR_INTERNET_DISCONNECTED",
				"net::ERR_ADDRESS_UNREACHABLE",
				"net::ERR_BLOCKED_BY_CLIENT",
				"net::ERR_BLOCKED_BY_RESPONSE",
				"net::ERR_FILE_NOT_FOUND",
			}
			if !slices.Contains(errors, ev.ErrorText) {
				logger.Debug(fmt.Sprintf("skip EventLoadingFailed: '%s' is not part of %+v", ev.ErrorText, errors))
				return
			}

			if ev.Type == network.ResourceTypeDocument {
				// Supposition: except iframe, an event loading failed with a
				// resource type Document is about the main page.
				logger.Debug("event EventLoadingFailed fired for main page")

				options.loadingFailedMu.Lock()
				defer options.loadingFailedMu.Unlock()

				*options.loadingFailed = fmt.Errorf("%s", ev.ErrorText)

				return
			}

			logger.Debug("event EventLoadingFailed fired for a resource")

			options.resourceLoadingFailedMu.Lock()
			defer options.resourceLoadingFailedMu.Unlock()

			*options.resourceLoadingFailed = multierr.Append(
				*options.resourceLoadingFailed,
				fmt.Errorf("resource %s: %s", ev.Type, ev.ErrorText),
			)
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
