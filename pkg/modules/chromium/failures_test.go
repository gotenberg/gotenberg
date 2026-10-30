package chromium

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestConversionFailures_err(t *testing.T) {
	// Records every failure, so that each scenario only removes the ones
	// that take priority over the expected one.
	all := func() *conversionFailures {
		f := new(conversionFailures)
		f.crash()
		f.denyWebSocket(errWebSocketDenied)
		f.failMainPageStatus(errors.New("500: Internal Server Error"))
		f.failResourceStatus(errors.New("https://cdn/a.css - 404: Not Found"))
		f.failMainPageLoading(errors.New("net::ERR_CONNECTION_REFUSED"))
		f.addConsoleException(errors.New("boom"))
		f.addConsoleMessage("[error] boom")
		f.failResourceLoading(errors.New("resource Stylesheet: net::ERR_NAME_NOT_RESOLVED"))
		return f
	}

	runErr := errors.New("Printing failed (-32000)")

	for _, tc := range []struct {
		scenario    string
		failures    func() *conversionFailures
		runErr      error
		browserGone bool
		failOnRes   bool
		expect      error
	}{
		{scenario: "a crash comes first", failures: all, runErr: runErr, browserGone: true, failOnRes: true, expect: ErrChromiumCrashed},
		{
			scenario: "then a refused WebSocket",
			failures: func() *conversionFailures { f := all(); f.crashed = nil; return f },
			runErr:   runErr, browserGone: true, failOnRes: true,
			expect: errWebSocketDenied,
		},
		{
			scenario: "then the browser going away during a failed run",
			failures: func() *conversionFailures { f := all(); f.crashed, f.webSocketDenied = nil, nil; return f },
			runErr:   runErr, browserGone: true, failOnRes: true,
			expect: ErrChromiumCrashed,
		},
		{
			scenario: "then the main page status",
			failures: func() *conversionFailures { f := all(); f.crashed, f.webSocketDenied = nil, nil; return f },
			runErr:   runErr, failOnRes: true,
			expect: ErrInvalidHttpStatusCode,
		},
		{
			scenario: "then a resource status",
			failures: func() *conversionFailures {
				f := all()
				f.crashed, f.webSocketDenied, f.invalidHttpStatusCode = nil, nil, nil
				return f
			},
			runErr: runErr, failOnRes: true,
			expect: ErrInvalidResourceHttpStatusCode,
		},
		{
			scenario: "then the main page loading",
			failures: func() *conversionFailures {
				f := all()
				f.crashed, f.webSocketDenied, f.invalidHttpStatusCode, f.invalidResourceHttpStatusCode = nil, nil, nil, nil
				return f
			},
			runErr: runErr, failOnRes: true,
			expect: ErrLoadingFailed,
		},
		{
			scenario: "then the run error",
			failures: func() *conversionFailures {
				f := all()
				f.crashed, f.webSocketDenied, f.invalidHttpStatusCode, f.invalidResourceHttpStatusCode, f.loadingFailed = nil, nil, nil, nil, nil
				return f
			},
			runErr: runErr, failOnRes: true,
			expect: ErrPrintingFailed,
		},
		{
			scenario: "then console exceptions",
			failures: func() *conversionFailures {
				f := all()
				f.crashed, f.webSocketDenied, f.invalidHttpStatusCode, f.invalidResourceHttpStatusCode, f.loadingFailed = nil, nil, nil, nil, nil
				return f
			},
			failOnRes: true,
			expect:    ErrConsoleExceptions,
		},
		{
			scenario: "then console messages",
			failures: func() *conversionFailures {
				f := all()
				f.crashed, f.webSocketDenied, f.invalidHttpStatusCode, f.invalidResourceHttpStatusCode, f.loadingFailed, f.consoleExceptions = nil, nil, nil, nil, nil, nil
				return f
			},
			failOnRes: true,
			expect:    ErrConsoleMessages,
		},
		{
			scenario: "then resource loading, when it fails the conversion",
			failures: func() *conversionFailures {
				f := all()
				f.crashed, f.webSocketDenied, f.invalidHttpStatusCode, f.invalidResourceHttpStatusCode, f.loadingFailed, f.consoleExceptions, f.consoleMessages = nil, nil, nil, nil, nil, nil, nil
				return f
			},
			failOnRes: true,
			expect:    ErrResourceLoadingFailed,
		},
		{
			scenario: "resource loading is ignored otherwise",
			failures: func() *conversionFailures {
				f := new(conversionFailures)
				f.failResourceLoading(errors.New("resource Stylesheet: net::ERR_NAME_NOT_RESOLVED"))
				return f
			},
		},
		{
			scenario:    "a browser death after a successful run is ignored",
			failures:    func() *conversionFailures { return new(conversionFailures) },
			browserGone: true,
		},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			err := tc.failures().err(tc.runErr, tc.browserGone, tc.failOnRes)
			if tc.expect == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if !errors.Is(err, tc.expect) {
				t.Fatalf("expected %v, got %v", tc.expect, err)
			}
		})
	}
}

// errWebSocketDenied stands for the refusal a WebSocket listener records.
var errWebSocketDenied = errors.New("ws://internal is denied")

func TestConversionFailures_ConsoleMessagesCap(t *testing.T) {
	f := new(conversionFailures)

	var wg sync.WaitGroup
	for i := range maxConsoleMessages + 3 {
		wg.Go(func() { f.addConsoleMessage(fmt.Sprintf("[log] %d", i)) })
	}
	wg.Wait()

	err := f.err(nil, false, false)
	if !errors.Is(err, ErrConsoleMessages) || !strings.Contains(err.Error(), "(3 more console message(s) omitted)") {
		t.Fatalf("expected the messages past the cap to be counted, got %v", err)
	}
}
