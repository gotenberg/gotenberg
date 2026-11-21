package api

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// NewDetachedContext returns an [echo.Context] carrying a snapshot of the
// given keys, of the matched route and of its path values, detached from
// Echo's context pool and from the client's response.
//
// Echo returns an [echo.Context] to its sync.Pool as soon as the middleware
// chain returns, and hands it to the next request, whose c.Reset() swaps the
// request, the response and the store. Code that keeps running past that
// point, such as a webhook goroutine or a handler that outlives its hard
// timeout, must not hold the pooled context: it would read another request's
// values and write into another client's response.
//
// [echo.NewContext] allocates outside the pool, so recycling cannot reach the
// returned context. It shares the [http.Request] of c, so the request
// headers, form values and path parameters still read as they do on c. Keys
// absent from c are omitted; the returned context still returns nil for them,
// matching [echo.Context.Get] behavior.
//
// The response is not the client's: whatever the code writes to the returned
// context lands in a discarding writer. The client only ever gets what the
// owner of c writes.
//
// Call it on the goroutine that owns c, before handing the returned context
// to another one.
func NewDetachedContext(c *echo.Context, keys ...string) *echo.Context {
	detached := echo.NewContext(c.Request(), &discardResponseWriter{header: make(http.Header)}, c.Echo())

	route := c.RouteInfo()
	pathValues := c.PathValues()
	detached.InitializeRoute(&route, &pathValues)

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
