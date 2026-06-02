package chromium

import (
	"net/url"
	"sync"

	"github.com/chromedp/cdproto/network"
)

// maxTrackedOrigins bounds the distinct origins kept per conversion so a
// pathological page cannot grow the set without limit.
const maxTrackedOrigins = 64

// networkAggregate accumulates per-conversion network activity from Chromium
// DevTools events. It is safe for concurrent use by the chromedp event listener
// goroutine and the conversion goroutine that reads the snapshot afterwards.
type networkAggregate struct {
	mu sync.Mutex

	requestCount int64
	bytesTotal   int64
	failedCount  int64

	origins        map[string]struct{}
	requestURLByID map[network.RequestID]string

	heaviestURL   string
	heaviestBytes int64
}

// networkStats is an immutable snapshot of a [networkAggregate].
type networkStats struct {
	requestCount  int64
	bytesTotal    int64
	failedCount   int64
	uniqueOrigins int64
	heaviestURL   string
	heaviestBytes int64
}

func newNetworkAggregate() *networkAggregate {
	return &networkAggregate{
		origins:        make(map[string]struct{}),
		requestURLByID: make(map[network.RequestID]string),
	}
}

// onResponseReceived records the response origin and remembers the URL for the
// request id, so a later loading-finished event can attribute its bytes.
func (a *networkAggregate) onResponseReceived(ev *network.EventResponseReceived) {
	if ev == nil || ev.Response == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if origin := originOf(ev.Response.URL); origin != "" {
		if _, ok := a.origins[origin]; !ok && len(a.origins) < maxTrackedOrigins {
			a.origins[origin] = struct{}{}
		}
	}
	a.requestURLByID[ev.RequestID] = ev.Response.URL
}

// onLoadingFinished records a successfully completed request and its size,
// tracking the single heaviest resource.
func (a *networkAggregate) onLoadingFinished(ev *network.EventLoadingFinished) {
	if ev == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.requestCount++

	size := int64(ev.EncodedDataLength)
	a.bytesTotal += size
	if size > a.heaviestBytes {
		a.heaviestBytes = size
		a.heaviestURL = a.requestURLByID[ev.RequestID]
	}
}

// onLoadingFailed records a request that failed to complete.
func (a *networkAggregate) onLoadingFailed(ev *network.EventLoadingFailed) {
	if ev == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.requestCount++
	a.failedCount++
}

func (a *networkAggregate) snapshot() networkStats {
	a.mu.Lock()
	defer a.mu.Unlock()

	return networkStats{
		requestCount:  a.requestCount,
		bytesTotal:    a.bytesTotal,
		failedCount:   a.failedCount,
		uniqueOrigins: int64(len(a.origins)),
		heaviestURL:   a.heaviestURL,
		heaviestBytes: a.heaviestBytes,
	}
}

// originOf returns the scheme://host of rawURL, or an empty string when it has
// no host (for example data: or file: URLs).
func originOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
