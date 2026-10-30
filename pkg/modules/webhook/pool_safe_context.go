package webhook

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// newDetachedContext returns an [echo.Context] carrying a snapshot of the given
// keys, detached from Echo's context pool and from the client's response.
//
// Echo returns an [echo.Context] to its sync.Pool as soon as the synchronous
// handler returns, including when the webhook middleware returns
// [api.ErrAsyncProcess]. A concurrent request can then claim the recycled
// context and c.Reset() wipes the shared store out from under the webhook
// goroutine, which causes any `c.Get("logger").(*slog.Logger)`-style assertion
// further down the chain to panic on a nil value.
//
// [echo.NewContext] allocates outside the pool, so recycling cannot reach the
// returned context. Keys absent from c are omitted; the returned context still
// returns nil for them, matching [echo.Context.Get] behavior.
//
// The response is pooled too, and the client must only ever get the webhook
// middleware's 204. Whatever the downstream chain writes, such as a route
// answering with JSON, lands in a discarding writer instead.
func newDetachedContext(c *echo.Context, keys ...string) *echo.Context {
	detached := echo.NewContext(c.Request(), &discardResponseWriter{header: make(http.Header)}, c.Echo())

	for _, key := range keys {
		if v := c.Get(key); v != nil {
			detached.Set(key, v)
		}
	}

	return detached
}

// discardResponseWriter is an [http.ResponseWriter] that drops everything.
type discardResponseWriter struct {
	header http.Header
}

func (w *discardResponseWriter) Header() http.Header { return w.header }

func (w *discardResponseWriter) Write(p []byte) (int, error) { return len(p), nil }

func (w *discardResponseWriter) WriteHeader(int) {}
