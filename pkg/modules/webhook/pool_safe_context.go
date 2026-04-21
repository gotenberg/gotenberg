package webhook

import (
	"sync"

	"github.com/labstack/echo/v4"
)

// poolSafeContext wraps an [echo.Context] and keeps a private snapshot of
// the values that downstream middleware and route handlers read from the
// store. Echo returns an [echo.Context] to its sync.Pool as soon as the
// synchronous handler returns, including when the webhook middleware
// returns [api.ErrAsyncProcess]. A concurrent request can then claim the
// recycled context and c.Reset() wipes the shared store out from under
// the webhook goroutine, which causes any
// `c.Get("logger").(*slog.Logger)`-style assertion further down the
// chain to panic on a nil value.
//
// Wrapping c before handing it to the goroutine insulates the async work
// from pool reuse: Get/Set read and write the private store while every
// other [echo.Context] method delegates to the embedded context for
// anything the downstream might still need.
type poolSafeContext struct {
	echo.Context
	mu    sync.RWMutex
	store map[string]any
}

// newPoolSafeContext snapshots the given keys from c into a detached
// store and returns a wrapper whose Get/Set operate on that store
// exclusively. Keys absent from c are omitted; the wrapper still
// returns nil for them, matching [echo.Context.Get] behavior.
func newPoolSafeContext(c echo.Context, keys ...string) *poolSafeContext {
	store := make(map[string]any, len(keys))
	for _, key := range keys {
		if v := c.Get(key); v != nil {
			store[key] = v
		}
	}
	return &poolSafeContext{Context: c, store: store}
}

// Get returns the value stored in the detached store, not the embedded
// context's pooled store.
func (p *poolSafeContext) Get(key string) any {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.store[key]
}

// Set writes to the detached store, not the embedded context's pooled
// store. This prevents downstream middleware writes from leaking into a
// later request that claims the same pooled context.
func (p *poolSafeContext) Set(key string, val any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.store[key] = val
}
