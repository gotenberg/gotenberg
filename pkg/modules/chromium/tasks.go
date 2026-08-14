package chromium

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// resolvePdfOptions applies the cross-option constraints Chromium imposes
// before printing.
//
// Chromium derives the PDF document outline from the tagged-PDF structure
// tree, so [PdfOptions.GenerateDocumentOutline] produces no outline unless
// tagged PDF is also generated. Requesting an outline therefore implies
// tagged PDF. See https://github.com/gotenberg/gotenberg/issues/1579.
func resolvePdfOptions(options PdfOptions) PdfOptions {
	if options.GenerateDocumentOutline {
		options.GenerateTaggedPdf = true
	}

	return options
}

func printToPdfActionFunc(reqCtx context.Context, logger *slog.Logger, outputPath string, options PdfOptions) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if options.GenerateDocumentOutline && !options.GenerateTaggedPdf {
			logger.DebugContext(ctx, "document outline requested, enabling tagged PDF because Chromium derives the outline from the structure tree")
		}

		options = resolvePdfOptions(options)

		// ctx is the chromedp task context, derived from context.Background(),
		// so the span is started under reqCtx to keep print_to_pdf in the
		// conversion trace instead of orphaning it into a new one.
		_, span := gotenberg.Tracer().Start(reqCtx, "chromium.print_to_pdf",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(printToPdfAttrs(options)...),
		)
		defer span.End()

		err := func() error {
			paperWidth := options.PaperWidth
			paperHeight := options.PaperHeight
			pageRanges := options.PageRanges

			if options.SinglePage {
				logger.DebugContext(ctx, "single page PDF")

				_, _, _, _, _, cssContentSize, err := page.GetLayoutMetrics().Do(ctx)
				if err != nil {
					return fmt.Errorf("get layout metrics: %w", err)
				}

				// There are 96 CSS pixels per inch.
				// See https://issues.chromium.org/issues/40267771#comment14.
				if options.Landscape {
					// Landscape swaps the paper dimensions, so the page is
					// WithPaperHeight wide by WithPaperWidth tall. Size both to
					// the content so the width expands to fit a wide document
					// (e.g. a table) instead of the height-only expansion
					// landing on the width axis and truncating it.
					// See https://github.com/gotenberg/gotenberg/issues/1390.
					paperWidth = (cssContentSize.Height / 96) + options.MarginTop + options.MarginBottom
					paperHeight = (cssContentSize.Width / 96) + options.MarginLeft + options.MarginRight
				} else {
					// We add top and bottom margins so that the content area
					// is large enough to fit the entire content.
					paperHeight = (cssContentSize.Height / 96) + options.MarginTop + options.MarginBottom
				}
				pageRanges = "1" // little dirty hack to avoid leftovers.
			}

			printToPdf := page.PrintToPDF().
				WithTransferMode(page.PrintToPDFTransferModeReturnAsStream).
				WithLandscape(options.Landscape).
				WithPrintBackground(options.PrintBackground).
				WithScale(options.Scale).
				WithPaperWidth(paperWidth).
				WithPaperHeight(paperHeight).
				WithMarginTop(options.MarginTop).
				WithMarginBottom(options.MarginBottom).
				WithMarginLeft(options.MarginLeft).
				WithMarginRight(options.MarginRight).
				WithPageRanges(pageRanges).
				WithPreferCSSPageSize(options.PreferCssPageSize).
				WithGenerateDocumentOutline(options.GenerateDocumentOutline).
				// See https://github.com/gotenberg/gotenberg/issues/1210.
				WithGenerateTaggedPDF(options.GenerateTaggedPdf)

			hasCustomHeaderFooter := options.HeaderTemplate != DefaultPdfOptions().HeaderTemplate ||
				options.FooterTemplate != DefaultPdfOptions().FooterTemplate

			if !hasCustomHeaderFooter {
				logger.DebugContext(ctx, "no custom header nor footer")

				printToPdf = printToPdf.WithDisplayHeaderFooter(false)
			} else {
				logger.DebugContext(ctx, "with custom header and/or footer")

				printToPdf = printToPdf.
					WithDisplayHeaderFooter(true).
					WithHeaderTemplate(options.HeaderTemplate).
					WithFooterTemplate(options.FooterTemplate)
			}

			logger.DebugContext(ctx, fmt.Sprintf("print to PDF with: %+v", printToPdf))

			_, stream, err := printToPdf.Do(ctx)
			if err != nil {
				return fmt.Errorf("print to PDF: %w", err)
			}

			reader := &streamReader{
				ctx:    ctx,
				handle: stream,
				r:      nil,
				pos:    0,
				eof:    false,
			}

			defer func() {
				err = reader.Close()
				if err != nil {
					logger.ErrorContext(ctx, fmt.Sprintf("close reader: %s", err))
				}
			}()

			file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				return fmt.Errorf("open output path: %w", err)
			}

			defer func() {
				err = file.Close()
				if err != nil {
					logger.ErrorContext(ctx, fmt.Sprintf("close output path: %s", err))
				}
			}()

			buffer := bufio.NewReader(reader)

			_, err = buffer.WriteTo(file)
			if err != nil {
				return fmt.Errorf("write result to output path: %w", err)
			}

			return nil
		}()

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// printToPdfAttrs derives bounded, low-cardinality attributes from the print
// options. Raw header/footer templates and page ranges are reduced to booleans
// to avoid leaking document content and exploding cardinality.
func printToPdfAttrs(options PdfOptions) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Bool("gotenberg.chromium.print.landscape", options.Landscape),
		attribute.Bool("gotenberg.chromium.print.print_background", options.PrintBackground),
		attribute.Float64("gotenberg.chromium.print.scale", options.Scale),
		attribute.Float64("gotenberg.chromium.print.paper_width", options.PaperWidth),
		attribute.Float64("gotenberg.chromium.print.paper_height", options.PaperHeight),
		attribute.Bool("gotenberg.chromium.print.single_page", options.SinglePage),
		attribute.Bool("gotenberg.chromium.print.prefer_css_page_size", options.PreferCssPageSize),
		attribute.Bool("gotenberg.chromium.print.generate_tagged_pdf", options.GenerateTaggedPdf),
		attribute.Bool("gotenberg.chromium.print.has_page_ranges", options.PageRanges != ""),
		attribute.Bool("gotenberg.chromium.print.has_header", options.HeaderTemplate != DefaultPdfOptions().HeaderTemplate),
		attribute.Bool("gotenberg.chromium.print.has_footer", options.FooterTemplate != DefaultPdfOptions().FooterTemplate),
	}
}

func captureScreenshotActionFunc(logger *slog.Logger, outputPath string, options ScreenshotOptions) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		captureScreenshot := page.CaptureScreenshot().
			WithCaptureBeyondViewport(true).
			WithFromSurface(true).
			WithOptimizeForSpeed(options.OptimizeForSpeed).
			WithFormat(page.CaptureScreenshotFormat(options.Format))

		switch {
		case options.Selector != "":
			clip, err := elementClip(ctx, options.Selector)
			if err != nil {
				return err
			}

			logger.DebugContext(ctx, fmt.Sprintf("clip screenshot to selector '%s'", options.Selector))
			captureScreenshot = captureScreenshot.WithClip(clip)
		case options.Clip:
			captureScreenshot = captureScreenshot.WithClip(&page.Viewport{
				Width:  float64(options.Width),
				Height: float64(options.Height),
				Scale:  1,
			})
		}

		if options.Format == "jpeg" {
			captureScreenshot = captureScreenshot.
				WithQuality(int64(options.Quality))
		}

		logger.DebugContext(ctx, fmt.Sprintf("capture screenshot with: %+v", captureScreenshot))

		buffer, err := captureScreenshot.Do(ctx)
		if err != nil {
			return fmt.Errorf("capture screenshot: %w", err)
		}

		file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("open output path: %w", err)
		}

		defer func() {
			err = file.Close()
			if err != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("close output path: %s", err))
			}
		}()

		_, err = file.Write(buffer)
		if err != nil {
			return fmt.Errorf("write result to output path: %w", err)
		}

		return nil
	}
}

// elementClip resolves the first element matching selector to a page-space clip
// rectangle for Page.captureScreenshot.
//
// getBoundingClientRect reports viewport-relative CSS pixels; adding the scroll
// offset puts the rectangle in the document coordinate space that
// WithCaptureBeyondViewport expects. It fails with
// [ErrScreenshotSelectorNotFound] when nothing matches or the match has no
// rendered box (display:none or a zero area), so the caller can answer 400.
func elementClip(ctx context.Context, selector string) (*page.Viewport, error) {
	var rect struct {
		Found  bool    `json:"found"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}

	expr := fmt.Sprintf(`(() => {
	const el = document.querySelector(%s);
	if (!el) {
		return { found: false };
	}
	const r = el.getBoundingClientRect();
	return { found: true, x: r.left + window.scrollX, y: r.top + window.scrollY, width: r.width, height: r.height };
})()`, strconv.Quote(selector))

	err := chromedp.Evaluate(expr, &rect).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("evaluate selector box: %v: %w", err, ErrScreenshotSelectorNotFound)
	}
	if !rect.Found || rect.Width <= 0 || rect.Height <= 0 {
		return nil, fmt.Errorf("selector %q matched no element with a visible box: %w", selector, ErrScreenshotSelectorNotFound)
	}

	return &page.Viewport{
		X:      rect.X,
		Y:      rect.Y,
		Width:  rect.Width,
		Height: rect.Height,
		Scale:  1,
	}, nil
}

func setDeviceMetricsOverride(logger *slog.Logger, width, height int, deviceScaleFactor float64) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		logger.DebugContext(ctx, "set device metrics override")

		err := emulation.SetDeviceMetricsOverride(int64(width), int64(height), deviceScaleFactor, false).Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("set device metrics override: %w", err)
	}
}

func clearCacheActionFunc(logger *slog.Logger, clear bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// See https://github.com/gotenberg/gotenberg/issues/753.
		if !clear {
			logger.DebugContext(ctx, "cache not cleared")
			return nil
		}

		logger.DebugContext(ctx, "clear cache")

		err := network.ClearBrowserCache().Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("clear cache: %w", err)
	}
}

func clearCookiesActionFunc(logger *slog.Logger, clear bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// See https://github.com/gotenberg/gotenberg/issues/753.
		if !clear {
			logger.DebugContext(ctx, "cookies not cleared")
			return nil
		}

		logger.DebugContext(ctx, "clear cookies")

		err := network.ClearBrowserCookies().Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("clear cookies: %w", err)
	}
}

// clearStorageActionFunc clears the converted origin's local storage before the
// page loads, so state written by a previous conversion of the same origin does
// not leak into this one. See https://github.com/gotenberg/gotenberg/issues/919.
//
// Session storage is not touched: each conversion runs in its own browsing
// context (a fresh tab), so it is already isolated and cannot leak. Local
// storage is per-origin and shared across tabs of the long-lived browser, so it
// is the only web storage that carries over.
func clearStorageActionFunc(logger *slog.Logger, clear bool, rawURL string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if !clear {
			logger.DebugContext(ctx, "local storage not cleared")
			return nil
		}

		origin, ok := httpOrigin(rawURL)
		if !ok {
			// A file:// upload gets an opaque, per-request origin that is not
			// shared between conversions, so there is nothing to clear.
			logger.DebugContext(ctx, "local storage not cleared: non-http(s) origin is already isolated")
			return nil
		}

		logger.DebugContext(ctx, fmt.Sprintf("clear local storage for %s", origin))

		err := storage.ClearDataForOrigin(origin, string(storage.TypeLocalStorage)).Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("clear local storage: %w", err)
	}
}

// httpOrigin returns the http(s) security origin (scheme://host[:port]) of
// rawURL, and false when rawURL is not http(s). A non-http(s) URL such as a
// file:// upload has an opaque origin that no other conversion shares.
func httpOrigin(rawURL string) (string, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host), true
}

func disableJavaScriptActionFunc(logger *slog.Logger, disable bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// See https://github.com/gotenberg/gotenberg/issues/175.
		if !disable {
			logger.DebugContext(ctx, "JavaScript not disabled")
			return nil
		}

		logger.DebugContext(ctx, "disable JavaScript")

		err := emulation.SetScriptExecutionDisabled(true).Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("disable JavaScript: %w", err)
	}
}

func setCookiesActionFunc(logger *slog.Logger, cookies []Cookie) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if len(cookies) == 0 {
			logger.DebugContext(ctx, "no cookies to set")
			return nil
		}

		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("context has no deadline, cannot set cookies")
		}
		epochTime := cdp.TimeSinceEpoch(deadline)

		cookiePretty := func(c *network.SetCookieParams) string {
			return fmt.Sprintf(
				"Name: '%s', Value: '%s', Domain: '%s', Path: '%s', Secure: %t, HTTPOnly: %t, SameSite: '%s', Expires: %s",
				c.Name,
				c.Value,
				c.Domain,
				c.Path,
				c.Secure,
				c.HTTPOnly,
				c.SameSite.String(),
				c.Expires.Time().String(),
			)
		}

		for _, cookie := range cookies {
			cookieParams := network.
				SetCookie(cookie.Name, cookie.Value).
				WithDomain(cookie.Domain).
				WithPath(cookie.Path).
				WithSecure(cookie.Secure).
				WithHTTPOnly(cookie.HttpOnly).
				WithSameSite(cookie.SameSite).
				WithExpires(&epochTime)

			err := cookieParams.Do(ctx)
			if err != nil {
				return fmt.Errorf("set cookie %s: %w", cookiePretty(cookieParams), err)
			}

			logger.DebugContext(ctx, fmt.Sprintf("set cookie %s", cookiePretty(cookieParams)))
		}

		return nil
	}
}

func userAgentOverride(logger *slog.Logger, userAgent string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if len(userAgent) == 0 {
			logger.DebugContext(ctx, "no user agent override")
			return nil
		}

		logger.DebugContext(ctx, fmt.Sprintf("user agent override: %s", userAgent))
		err := emulation.SetUserAgentOverride(userAgent).Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("set user agent override: %w", err)
	}
}

// This code has been replaced with the listenForEventRequestPaused function.
// Indeed, the user may want to scope the headers per domain, but using
// network.SetExtraHTTPHeaders set the headers for ALL requests from the page.
// See https://github.com/gotenberg/gotenberg/issues/1011.
//
//  func extraHttpHeadersActionFunc(logger *slog.Logger, extraHttpHeaders map[string]string) chromedp.ActionFunc {
// 	return func(ctx context.Context) error {
// 		if len(extraHttpHeaders) == 0 {
// 			logger.DebugContext(ctx,"no extra HTTP headers")
// 			return nil
// 		}
//
// 		logger.DebugContext(ctx,fmt.Sprintf("extra HTTP headers: %+v", extraHttpHeaders))
//
// 		headers := make(network.Headers, len(extraHttpHeaders))
// 		for key, value := range extraHttpHeaders {
// 			headers[key] = value
// 		}
//
// 		err := network.SetExtraHTTPHeaders(headers).Do(ctx)
// 		if err == nil {
// 			return nil
// 		}
//
// 		return fmt.Errorf("set extra HTTP headers: %w", err)
// 	}
// }

func navigateActionFunc(logger *slog.Logger, url string, skipNetworkIdleEvent, skipNetworkAlmostIdleEvent bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		logger.DebugContext(ctx, fmt.Sprintf("navigate to '%s'", url))

		// Register lifecycle listeners before issuing Page.navigate. For
		// fast loads (typically file:// pages with no external
		// sub-resources), DomContentEventFired / LoadEventFired /
		// LoadingFinished can fire between Navigate.Do returning and
		// runBatch spawning the waiter goroutines. Registering ahead of
		// the navigate command closes that race.
		// See https://github.com/gotenberg/gotenberg/issues/1561.
		waitFunc := []func() error{
			waitForEventDomContentEventFired(ctx, logger),
			waitForEventLoadEventFired(ctx, logger),
			waitForEventLoadingFinished(ctx, logger),
		}

		if !skipNetworkIdleEvent {
			waitFunc = append(waitFunc, waitForEventNetworkIdle(ctx, logger))
		} else {
			logger.DebugContext(ctx, "skipping network idle event")
		}

		if !skipNetworkAlmostIdleEvent {
			waitFunc = append(waitFunc, waitForEventNetworkAlmostIdle(ctx, logger))
		} else {
			logger.DebugContext(ctx, "skipping network almost idle event")
		}

		_, _, _, _, err := page.Navigate(url).Do(ctx)
		if err != nil {
			return fmt.Errorf("navigate to '%s': %w", url, err)
		}

		err = runBatch(
			ctx,
			waitFunc...,
		)

		if err == nil {
			return nil
		}

		return fmt.Errorf("wait for events: %w", err)
	}
}

func hideDefaultWhiteBackgroundActionFunc(logger *slog.Logger, omitBackground, printBackground bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// See https://github.com/gotenberg/gotenberg/issues/226.
		if !omitBackground {
			logger.DebugContext(ctx, "default white background not hidden")
			return nil
		}

		if !printBackground {
			// See https://github.com/chromedp/chromedp/issues/1179#issuecomment-1284794416.
			return fmt.Errorf("validate omit background: %w", ErrOmitBackgroundWithoutPrintBackground)
		}

		logger.DebugContext(ctx, "hide default white background")

		err := emulation.SetDefaultBackgroundColorOverride().WithColor(
			&cdp.RGBA{
				R: 0,
				G: 0,
				B: 0,
				A: 0,
			}).Do(ctx)

		if err == nil {
			return nil
		}

		return fmt.Errorf("hide default white background: %w", err)
	}
}

func forceExactColorsActionFunc(logger *slog.Logger, printBackground bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		css := "html { -webkit-print-color-adjust: exact !important; }"
		if !printBackground {
			// The -webkit-print-color-adjust: exact CSS property forces the
			// print of the background, whatever the printToPDF args.
			// See https://github.com/gotenberg/gotenberg/issues/1154.
			additionalCss := "html, body { background: none !important; }"
			logger.DebugContext(ctx, fmt.Sprintf("inject %s as printBackground is %t", additionalCss, printBackground))
			css += additionalCss
		}

		script := fmt.Sprintf(`
(() => {
	const css = '%s';
	const style = document.createElement('style');
	style.type = 'text/css';
	style.appendChild(document.createTextNode(css));
	document.head.appendChild(style);
})();
`, css)

		evaluate := chromedp.Evaluate(script, nil)
		err := evaluate.Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("add CSS for exact colors: %w", err)
	}
}

func emulateMediaTypeActionFunc(logger *slog.Logger, mediaType string, mediaFeatures []EmulatedMediaFeature) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if mediaType == "" && len(mediaFeatures) == 0 {
			logger.DebugContext(ctx, "no emulated media type or features")
			return nil
		}

		if mediaType != "" && mediaType != "screen" && mediaType != "print" {
			return fmt.Errorf("validate emulated media type '%s': %w", mediaType, ErrInvalidEmulatedMediaType)
		}

		emulatedMedia := emulation.SetEmulatedMedia()

		if mediaType != "" {
			logger.DebugContext(ctx, fmt.Sprintf("emulate media type '%s'", mediaType))
			emulatedMedia = emulatedMedia.WithMedia(mediaType)
		}

		if len(mediaFeatures) > 0 {
			logger.DebugContext(ctx, fmt.Sprintf("emulate media features %+v", mediaFeatures))

			features := make([]*emulation.MediaFeature, len(mediaFeatures))
			for i, f := range mediaFeatures {
				features[i] = &emulation.MediaFeature{
					Name:  f.Name,
					Value: f.Value,
				}
			}

			emulatedMedia = emulatedMedia.WithFeatures(features)
		}

		err := emulatedMedia.Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("emulate media: %w", err)
	}
}

func waitDelayBeforePrintActionFunc(logger *slog.Logger, disableJavaScript bool, delay time.Duration) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if disableJavaScript {
			logger.DebugContext(ctx, "JavaScript disabled, skipping wait delay")
			return nil
		}

		if delay <= 0 {
			logger.DebugContext(ctx, "no wait delay")
			return nil
		}

		// We wait for a given amount of time so that JavaScript
		// scripts have a chance to finish before printing the page.
		logger.DebugContext(ctx, fmt.Sprintf("wait '%s' before print", delay))

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait delay: %w", ctx.Err())
		case <-time.After(delay):
			return nil
		}
	}
}

func waitForExpressionBeforePrintActionFunc(logger *slog.Logger, disableJavaScript bool, expression string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if disableJavaScript {
			logger.DebugContext(ctx, "JavaScript disabled, skipping wait expression")
			return nil
		}

		if expression == "" {
			logger.DebugContext(ctx, "no wait expression")
			return nil
		}

		// We wait until the evaluation of the expression is true or
		// until the context is done.
		logger.DebugContext(ctx, fmt.Sprintf("wait until '%s' is true before print", expression))
		ticker := time.NewTicker(time.Duration(100) * time.Millisecond)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return fmt.Errorf("context done while evaluating '%s': %w", expression, ctx.Err())
			case <-ticker.C:
				var ok bool
				// If the provided expression is a async function (thenable) it will be awaited
				//   (async () => {
				//		document.querySelector('#accept').click();
				//		await new Promise(r => setTimeout(r, 2000));
				//		return true;
				//	})()
				evaluate := chromedp.Evaluate(expression, &ok, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
					return p.WithAwaitPromise(true)
				})

				err := evaluate.Do(ctx)
				if err != nil {
					return fmt.Errorf("evaluate: %v: %w", err, ErrInvalidEvaluationExpression)
				}

				if ok {
					ticker.Stop()
					return nil
				}

				continue
			}
		}
	}
}

func waitForSelectorVisibleBeforePrintActionFunc(logger *slog.Logger, selector string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if selector == "" {
			logger.DebugContext(ctx, "no wait selector")
			return nil
		}

		logger.DebugContext(ctx, fmt.Sprintf("wait until '%s' is visible before print", selector))
		err := chromedp.WaitVisible(selector, chromedp.ByQuery, chromedp.RetryInterval(time.Duration(100)*time.Millisecond)).Do(ctx)
		if err != nil {
			return fmt.Errorf("wait visible: %v: %w", err, ErrInvalidSelectorQuery)
		}
		return nil
	}
}
