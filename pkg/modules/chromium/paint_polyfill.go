package chromium

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// paintCallbacksPolyfill is a JavaScript shim installed before any user
// script runs. It replaces [requestAnimationFrame], [cancelAnimationFrame],
// [ResizeObserver], and [IntersectionObserver] with timer-backed
// implementations. Headless Chromium's print-emulation pipeline does not
// tick the compositor refresh driver reliably between
// "load" and [Page.printToPDF], which leaves the native rAF /
// ResizeObserver / IntersectionObserver queues permanently stalled and
// breaks charting libraries (visx, ApexCharts, and similar) that rely on
// rAF-gated measurement. The polyfill exposes the same APIs with timer
// semantics, so user scripts that schedule work on those callbacks
// receive their measurements and Gotenberg's rendered output reflects
// the page the author intended. See
// https://github.com/gotenberg/gotenberg/issues/1535.
const paintCallbacksPolyfill = `
(function () {
	var nextHandle = 0;
	var pending = new Map();

	window.requestAnimationFrame = function (callback) {
		var handle = ++nextHandle;
		pending.set(handle, setTimeout(function () {
			pending.delete(handle);
			callback(performance.now());
		}, 16));
		return handle;
	};
	window.cancelAnimationFrame = function (handle) {
		clearTimeout(pending.get(handle));
		pending.delete(handle);
	};

	function PollingResizeObserver(callback) {
		this._callback = callback;
		this._observed = [];
		this._timer = null;
	}
	PollingResizeObserver.prototype.observe = function (el) {
		var entry = { target: el, lastW: -1, lastH: -1 };
		this._observed.push(entry);
		if (!this._timer) {
			var self = this;
			this._timer = setInterval(function () { self._tick(); }, 100);
		}
		this._tick();
	};
	PollingResizeObserver.prototype.unobserve = function (el) {
		this._observed = this._observed.filter(function (e) { return e.target !== el; });
		if (this._observed.length === 0 && this._timer) {
			clearInterval(this._timer);
			this._timer = null;
		}
	};
	PollingResizeObserver.prototype.disconnect = function () {
		this._observed = [];
		if (this._timer) {
			clearInterval(this._timer);
			this._timer = null;
		}
	};
	PollingResizeObserver.prototype._tick = function () {
		var changed = [];
		for (var i = 0; i < this._observed.length; i++) {
			var e = this._observed[i];
			var w = e.target.offsetWidth;
			var h = e.target.offsetHeight;
			if (w !== e.lastW || h !== e.lastH) {
				e.lastW = w;
				e.lastH = h;
				changed.push({
					target: e.target,
					contentRect: { width: w, height: h, top: 0, left: 0, right: w, bottom: h, x: 0, y: 0 },
					borderBoxSize: [{ inlineSize: w, blockSize: h }],
					contentBoxSize: [{ inlineSize: w, blockSize: h }],
					devicePixelContentBoxSize: [{ inlineSize: w, blockSize: h }],
				});
			}
		}
		if (changed.length > 0) {
			try { this._callback(changed, this); } catch (err) { console.error(err); }
		}
	};
	window.ResizeObserver = PollingResizeObserver;

	function ImmediateIntersectionObserver(callback) {
		this._callback = callback;
		this._observed = [];
	}
	ImmediateIntersectionObserver.prototype.observe = function (el) {
		this._observed.push(el);
		var self = this;
		setTimeout(function () {
			var rect = el.getBoundingClientRect();
			var entry = {
				target: el,
				isIntersecting: true,
				intersectionRatio: 1,
				boundingClientRect: rect,
				intersectionRect: rect,
				rootBounds: null,
				time: performance.now(),
			};
			try { self._callback([entry], self); } catch (err) { console.error(err); }
		}, 0);
	};
	ImmediateIntersectionObserver.prototype.unobserve = function (el) {
		this._observed = this._observed.filter(function (x) { return x !== el; });
	};
	ImmediateIntersectionObserver.prototype.disconnect = function () {
		this._observed = [];
	};
	ImmediateIntersectionObserver.prototype.takeRecords = function () { return []; };
	window.IntersectionObserver = ImmediateIntersectionObserver;
})();
`

// injectPaintCallbacksPolyfillActionFunc installs the
// [paintCallbacksPolyfill] via [page.AddScriptToEvaluateOnNewDocument]
// so that it runs before any user script on the navigated page. Callers
// set install=false to skip the shim when no readiness signal gates the
// conversion, which preserves the native implementations for pages that
// do not require it.
func injectPaintCallbacksPolyfillActionFunc(logger *slog.Logger, install bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if !install {
			logger.DebugContext(ctx, "paint-callbacks polyfill not requested")
			return nil
		}
		logger.DebugContext(ctx, "inject paint-callbacks polyfill")
		_, err := page.AddScriptToEvaluateOnNewDocument(paintCallbacksPolyfill).Do(ctx)
		if err != nil {
			return fmt.Errorf("inject paint-callbacks polyfill: %w", err)
		}
		return nil
	}
}
