package chromium

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// conversionFailures gathers what the event listeners of one conversion
// record, behind one mutex, and decides which failure the conversion reports.
// Listeners write from chromedp's event goroutines while the conversion runs.
type conversionFailures struct {
	mu sync.Mutex

	crashed                       error
	webSocketDenied               error
	invalidHttpStatusCode         error
	invalidResourceHttpStatusCode error
	loadingFailed                 error
	resourceLoadingFailed         error
	consoleExceptions             error
	consoleMessages               []string
	droppedConsoleMessages        int
}

// crash records a renderer crash.
func (f *conversionFailures) crash() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.crashed = ErrChromiumCrashed
}

// denyWebSocket records the first WebSocket handshake the policy refuses.
func (f *conversionFailures) denyWebSocket(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.webSocketDenied == nil {
		f.webSocketDenied = fmt.Errorf("WebSocket handshake: %w", err)
	}
}

// failMainPageStatus records the main page answering a failing status code.
func (f *conversionFailures) failMainPageStatus(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invalidHttpStatusCode = err
}

// failResourceStatus adds a resource answering a failing status code.
func (f *conversionFailures) failResourceStatus(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invalidResourceHttpStatusCode = errors.Join(f.invalidResourceHttpStatusCode, err)
}

// failMainPageLoading records the main page failing to load.
func (f *conversionFailures) failMainPageLoading(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loadingFailed = err
}

// failResourceLoading adds a resource failing to load.
func (f *conversionFailures) failResourceLoading(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resourceLoadingFailed = errors.Join(f.resourceLoadingFailed, err)
}

// addConsoleException adds an exception thrown in the page.
func (f *conversionFailures) addConsoleException(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consoleExceptions = errors.Join(f.consoleExceptions, err)
}

// addConsoleMessage adds a console message, up to [maxConsoleMessages], and
// counts the ones past it.
func (f *conversionFailures) addConsoleMessage(message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.consoleMessages) >= maxConsoleMessages {
		f.droppedConsoleMessages++
		return
	}
	f.consoleMessages = append(f.consoleMessages, message)
}

// err returns the failure the conversion reports, given the error of the
// chromedp run, whether the browser went away, and whether a resource that
// fails to load fails the conversion.
//
// The order matters. A crashed renderer is the root cause of every other
// failure, and a refused WebSocket cancels the conversion, so both come
// before the cancellation they cause. Event-driven main page failures come
// before the run error, since canceling the run on them turns it into a less
// informative context error. See
// https://github.com/gotenberg/gotenberg/issues/1492 and
// https://github.com/gotenberg/gotenberg/issues/1640.
func (f *conversionFailures) err(runErr error, browserGone, failOnResourceLoadingFailed bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.crashed != nil {
		return fmt.Errorf("handle tasks: %w", f.crashed)
	}

	if f.webSocketDenied != nil {
		return fmt.Errorf("handle tasks: %w", f.webSocketDenied)
	}

	// The browser context is only ever canceled when the browser process dies
	// or is stopped, never on a request timeout. Gated on runErr, so that a
	// successful conversion is never discarded by a browser death right after
	// it.
	if runErr != nil && browserGone {
		return fmt.Errorf("handle tasks: %w", ErrChromiumCrashed)
	}

	// See https://github.com/gotenberg/gotenberg/issues/613.
	if f.invalidHttpStatusCode != nil {
		return fmt.Errorf("%v: %w", f.invalidHttpStatusCode, ErrInvalidHttpStatusCode)
	}

	// See https://github.com/gotenberg/gotenberg/issues/1021.
	if f.invalidResourceHttpStatusCode != nil {
		return fmt.Errorf("%v: %w", f.invalidResourceHttpStatusCode, ErrInvalidResourceHttpStatusCode)
	}

	// See https://github.com/gotenberg/gotenberg/issues/913 and
	// https://github.com/gotenberg/gotenberg/issues/959.
	if f.loadingFailed != nil {
		return fmt.Errorf("%v: %w", f.loadingFailed, ErrLoadingFailed)
	}

	if runErr != nil {
		return runError(runErr)
	}

	// See https://github.com/gotenberg/gotenberg/issues/262.
	if f.consoleExceptions != nil {
		return fmt.Errorf("%v: %w", f.consoleExceptions, ErrConsoleExceptions)
	}

	// See https://github.com/gotenberg/gotenberg/issues/1667.
	if len(f.consoleMessages) != 0 {
		messages := f.consoleMessages
		if f.droppedConsoleMessages != 0 {
			messages = append(slices.Clone(messages), fmt.Sprintf("(%d more console message(s) omitted)", f.droppedConsoleMessages))
		}

		return fmt.Errorf("%s: %w", strings.Join(messages, "\n"), ErrConsoleMessages)
	}

	// See https://github.com/gotenberg/gotenberg/issues/1021.
	if failOnResourceLoadingFailed && f.resourceLoadingFailed != nil {
		return fmt.Errorf("%v: %w", f.resourceLoadingFailed, ErrResourceLoadingFailed)
	}

	return nil
}

// runError maps the error of a chromedp run to the sentinel of the Chromium
// failure it reports, when there is one.
func runError(runErr error) error {
	errMessage := runErr.Error()

	switch {
	case strings.Contains(errMessage, "Printing failed (-32000)"):
		return ErrPrintingFailed
	case isInvalidPrinterSettingsError(errMessage):
		return ErrInvalidPrinterSettings
	case strings.Contains(errMessage, "Page range syntax error"):
		return ErrPageRangesSyntaxError
	case strings.Contains(errMessage, "Page range exceeds page count (-32000)"):
		return ErrPageRangesExceedsPageCount
	case strings.Contains(errMessage, "rpcc: message too large"):
		return ErrRpccMessageTooLarge
	default:
		return fmt.Errorf("handle tasks: %w", runErr)
	}
}
