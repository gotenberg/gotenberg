package chromium

import (
	"fmt"
	"sync"
	"testing"

	"github.com/chromedp/cdproto/network"
)

func TestOriginOf(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{"https://example.com/path?q=1", "https://example.com"},
		{"http://cdn.example.com:8080/a.js", "http://cdn.example.com:8080"},
		{"data:image/png;base64,AAAA", ""},
		{"file:///tmp/index.html", ""},
		{"not a url", ""},
	} {
		if got := originOf(tc.raw); got != tc.want {
			t.Errorf("originOf(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestNetworkAggregate_Snapshot(t *testing.T) {
	a := newNetworkAggregate()

	a.onResponseReceived(&network.EventResponseReceived{
		RequestID: "1",
		Response:  &network.Response{URL: "https://example.com/a.js"},
	})
	a.onResponseReceived(&network.EventResponseReceived{
		RequestID: "2",
		Response:  &network.Response{URL: "https://cdn.example.com/b.png"},
	})
	// Duplicate origin must not grow the set.
	a.onResponseReceived(&network.EventResponseReceived{
		RequestID: "3",
		Response:  &network.Response{URL: "https://example.com/c.css"},
	})

	a.onLoadingFinished(&network.EventLoadingFinished{RequestID: "1", EncodedDataLength: 100})
	a.onLoadingFinished(&network.EventLoadingFinished{RequestID: "2", EncodedDataLength: 900})
	a.onLoadingFailed(&network.EventLoadingFailed{RequestID: "3"})

	got := a.snapshot()
	if got.requestCount != 3 {
		t.Errorf("requestCount = %d, want 3", got.requestCount)
	}
	if got.bytesTotal != 1000 {
		t.Errorf("bytesTotal = %d, want 1000", got.bytesTotal)
	}
	if got.failedCount != 1 {
		t.Errorf("failedCount = %d, want 1", got.failedCount)
	}
	if got.uniqueOrigins != 2 {
		t.Errorf("uniqueOrigins = %d, want 2", got.uniqueOrigins)
	}
	if got.heaviestBytes != 900 || got.heaviestURL != "https://cdn.example.com/b.png" {
		t.Errorf("heaviest = (%q, %d), want (%q, 900)", got.heaviestURL, got.heaviestBytes, "https://cdn.example.com/b.png")
	}
}

func TestNetworkAggregate_OriginCap(t *testing.T) {
	a := newNetworkAggregate()
	for i := range maxTrackedOrigins + 50 {
		a.onResponseReceived(&network.EventResponseReceived{
			RequestID: network.RequestID(fmt.Sprintf("r%d", i)),
			Response:  &network.Response{URL: fmt.Sprintf("https://host%d.example.com/x", i)},
		})
	}
	if got := a.snapshot().uniqueOrigins; got != maxTrackedOrigins {
		t.Errorf("uniqueOrigins = %d, want %d (capped)", got, maxTrackedOrigins)
	}
}

func TestNetworkAggregate_ConcurrentSafe(t *testing.T) {
	a := newNetworkAggregate()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := network.RequestID(fmt.Sprintf("r%d", i))
			a.onResponseReceived(&network.EventResponseReceived{
				RequestID: id,
				Response:  &network.Response{URL: fmt.Sprintf("https://host%d.example.com/x", i)},
			})
			a.onLoadingFinished(&network.EventLoadingFinished{RequestID: id, EncodedDataLength: 10})
		}(i)
	}
	wg.Wait()

	if got := a.snapshot().requestCount; got != 100 {
		t.Errorf("requestCount = %d, want 100", got)
	}
}

// TestNetworkAggregate_SettledRequestsAreDropped covers the growth where every
// response URL stayed in the map for the whole conversion even though nothing
// reads it again once the request settles.
func TestNetworkAggregate_SettledRequestsAreDropped(t *testing.T) {
	a := newNetworkAggregate()

	for i := range 500 {
		id := network.RequestID(fmt.Sprintf("r%d", i))
		a.onResponseReceived(&network.EventResponseReceived{
			RequestID: id,
			Response:  &network.Response{URL: fmt.Sprintf("https://host.example.com/%d", i)},
		})

		if i%2 == 0 {
			a.onLoadingFinished(&network.EventLoadingFinished{RequestID: id, EncodedDataLength: 10})
			continue
		}

		a.onLoadingFailed(&network.EventLoadingFailed{RequestID: id})
	}

	a.mu.Lock()
	tracked := len(a.requestURLByID)
	a.mu.Unlock()

	if tracked != 0 {
		t.Errorf("tracked requests = %d, want 0: settled requests must not be retained", tracked)
	}

	// The bookkeeping the map feeds must survive the pruning.
	got := a.snapshot()
	if got.requestCount != 500 {
		t.Errorf("requestCount = %d, want 500", got.requestCount)
	}
	if got.failedCount != 250 {
		t.Errorf("failedCount = %d, want 250", got.failedCount)
	}
}

// TestNetworkAggregate_UnsettledRequestCap verifies the ceiling that applies
// when requests never settle, which is the only way the map can still grow.
func TestNetworkAggregate_UnsettledRequestCap(t *testing.T) {
	a := newNetworkAggregate()

	for i := range maxTrackedRequests + 500 {
		a.onResponseReceived(&network.EventResponseReceived{
			RequestID: network.RequestID(fmt.Sprintf("r%d", i)),
			Response:  &network.Response{URL: fmt.Sprintf("https://host.example.com/%d", i)},
		})
	}

	a.mu.Lock()
	tracked := len(a.requestURLByID)
	a.mu.Unlock()

	if tracked != maxTrackedRequests {
		t.Errorf("tracked requests = %d, want %d (capped)", tracked, maxTrackedRequests)
	}
}

// TestNetworkAggregate_HeaviestURLSurvivesPruning guards the attribution the
// map exists for: the URL must still be resolved before the entry is dropped.
func TestNetworkAggregate_HeaviestURLSurvivesPruning(t *testing.T) {
	a := newNetworkAggregate()

	a.onResponseReceived(&network.EventResponseReceived{
		RequestID: "small",
		Response:  &network.Response{URL: "https://example.com/small.css"},
	})
	a.onLoadingFinished(&network.EventLoadingFinished{RequestID: "small", EncodedDataLength: 10})

	a.onResponseReceived(&network.EventResponseReceived{
		RequestID: "big",
		Response:  &network.Response{URL: "https://example.com/big.png"},
	})
	a.onLoadingFinished(&network.EventLoadingFinished{RequestID: "big", EncodedDataLength: 4096})

	got := a.snapshot()
	if got.heaviestURL != "https://example.com/big.png" || got.heaviestBytes != 4096 {
		t.Errorf("heaviest = (%q, %d), want (%q, 4096)", got.heaviestURL, got.heaviestBytes, "https://example.com/big.png")
	}
}
