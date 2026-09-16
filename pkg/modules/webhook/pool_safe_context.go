package webhook

import (
	"github.com/labstack/echo/v5"
)

// newDetachedContext returns an [echo.Context] carrying a snapshot of the given
// keys, detached from Echo's context pool.
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
// Only the asynchronous path uses this. Nothing downstream of the webhook
// middleware writes to the response: contextMiddleware sits upstream and has
// already answered 204 by the time the goroutine runs.
func newDetachedContext(c *echo.Context, keys ...string) *echo.Context {
	detached := echo.NewContext(c.Request(), c.Response(), c.Echo())

	for _, key := range keys {
		if v := c.Get(key); v != nil {
			detached.Set(key, v)
		}
	}

	return detached
}
