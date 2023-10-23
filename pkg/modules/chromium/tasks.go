package chromium

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
)

func printToPdfActionFunc(logger *zap.Logger, outputPath string, options Options) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		printToPdf := page.PrintToPDF().
			WithTransferMode(page.PrintToPDFTransferModeReturnAsStream).
			WithLandscape(options.Landscape).
			WithPrintBackground(options.PrintBackground).
			WithScale(options.Scale).
			WithPaperWidth(options.PaperWidth).
			WithPaperHeight(options.PaperHeight).
			WithMarginTop(options.MarginTop).
			WithMarginBottom(options.MarginBottom).
			WithMarginLeft(options.MarginLeft).
			WithMarginRight(options.MarginRight).
			WithPageRanges(options.PageRanges).
			WithPreferCSSPageSize(options.PreferCssPageSize)

		hasCustomHeaderFooter := options.HeaderTemplate != DefaultOptions().HeaderTemplate ||
			options.FooterTemplate != DefaultOptions().FooterTemplate

		if !hasCustomHeaderFooter {
			logger.Debug("no custom header nor footer")

			printToPdf = printToPdf.WithDisplayHeaderFooter(false)
		} else {
			logger.Debug("with custom header and/or footer")

			printToPdf = printToPdf.
				WithDisplayHeaderFooter(true).
				WithHeaderTemplate(options.HeaderTemplate).
				WithFooterTemplate(options.FooterTemplate)
		}

		logger.Debug(fmt.Sprintf("print to Pdf with: %+v", printToPdf))

		_, stream, err := printToPdf.Do(ctx)
		if err != nil {
			return fmt.Errorf("print to Pdf: %w", err)
		}

		reader := &streamReader{
			ctx:    ctx,
			handle: stream,
			r:      nil,
			pos:    0,
			eof:    false,
		}

		defer func() {
			err := reader.Close()
			if err != nil {
				logger.Error(fmt.Sprintf("close reader: %s", err))
			}
		}()

		file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("open output path: %w", err)
		}

		defer func() {
			err := file.Close()
			if err != nil {
				logger.Error(fmt.Sprintf("close output path: %s", err))
			}
		}()

		buffer := bufio.NewReader(reader)

		_, err = buffer.WriteTo(file)
		if err != nil {
			return fmt.Errorf("write result to output path: %w", err)
		}

		return nil
	}
}

func disableJavaScriptActionFunc(logger *zap.Logger, disable bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// See https://github.com/gotenberg/gotenberg/issues/175.
		if !disable {
			logger.Debug("JavaScript not disabled")

			return nil
		}

		logger.Debug("disable JavaScript")

		err := emulation.SetScriptExecutionDisabled(true).Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("disable JavaScript: %w", err)
	}
}

func extraHttpHeadersActionFunc(logger *zap.Logger, extraHttpHeaders map[string]string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if len(extraHttpHeaders) == 0 {
			logger.Debug("no extra HTTP headers")

			return nil
		}

		logger.Debug(fmt.Sprintf("extra HTTP headers: %+v", extraHttpHeaders))

		headers := make(network.Headers, len(extraHttpHeaders))
		for key, value := range extraHttpHeaders {
			headers[key] = value
		}

		err := network.SetExtraHTTPHeaders(headers).Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("set extra HTTP headers: %w", err)
	}
}

func navigateActionFunc(logger *zap.Logger, url string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		logger.Debug(fmt.Sprintf("navigate to '%s'", url))

		_, _, _, err := page.Navigate(url).Do(ctx)
		if err != nil {
			return fmt.Errorf("navigate to '%s': %w", url, err)
		}

		err = runBatch(
			ctx,
			waitForEventDomContentEventFired(ctx, logger),
			waitForEventLoadEventFired(ctx, logger),
			waitForEventNetworkIdle(ctx, logger),
			waitForEventLoadingFinished(ctx, logger),
		)

		if err == nil {
			return nil
		}

		return fmt.Errorf("wait for events: %w", err)
	}
}

func hideDefaultWhiteBackgroundActionFunc(logger *zap.Logger, omitBackground, printBackground bool) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// See https://github.com/gotenberg/gotenberg/issues/226.
		if !omitBackground {
			logger.Debug("default white background not hidden")

			return nil
		}

		if !printBackground {
			// See https://github.com/chromedp/chromedp/issues/1179#issuecomment-1284794416.
			return fmt.Errorf("validate omit background: %w", ErrOmitBackgroundWithoutPrintBackground)
		}

		logger.Debug("hide default white background")

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

func forceExactColorsActionFunc() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		// See:
		// https://github.com/gotenberg/gotenberg/issues/354
		// https://github.com/puppeteer/puppeteer/issues/2685
		// https://github.com/chromedp/chromedp/issues/520
		script := `
(() => {
	const css = 'html { -webkit-print-color-adjust: exact !important; }';

	const style = document.createElement('style');
	style.type = 'text/css';
	style.appendChild(document.createTextNode(css));
	document.head.appendChild(style);
})();
`

		evaluate := chromedp.Evaluate(script, nil)
		err := evaluate.Do(ctx)

		if err == nil {
			return nil
		}

		return fmt.Errorf("add CSS for exact colors: %w", err)
	}
}

func emulateMediaTypeActionFunc(logger *zap.Logger, mediaType string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if mediaType == "" {
			logger.Debug("no emulated media type")

			return nil
		}

		if mediaType != "screen" && mediaType != "print" {
			return fmt.Errorf("validate emulated media type '%s': %w", mediaType, ErrInvalidEmulatedMediaType)
		}

		logger.Debug(fmt.Sprintf("emulate media type '%s'", mediaType))

		emulatedMedia := emulation.SetEmulatedMedia()
		err := emulatedMedia.WithMedia(mediaType).Do(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("emulate media type '%s': %w", mediaType, err)
	}
}

func waitDelayBeforePrintActionFunc(logger *zap.Logger, disableJavaScript bool, delay time.Duration) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if disableJavaScript {
			logger.Debug("JavaScript disabled, skipping wait delay")

			return nil
		}

		if delay <= 0 {
			logger.Debug("no wait delay")

			return nil
		}

		// We wait for a given amount of time so that JavaScript
		// scripts have a chance to finish before printing the page.
		logger.Debug(fmt.Sprintf("wait '%s' before print", delay))

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait delay: %w", ctx.Err())
		case <-time.After(delay):
			return nil
		}
	}
}

func waitForExpressionBeforePrintActionFunc(logger *zap.Logger, disableJavaScript bool, expression string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if disableJavaScript {
			logger.Debug("JavaScript disabled, skipping wait expression")

			return nil
		}

		if expression == "" {
			logger.Debug("no wait expression")

			return nil
		}

		// We wait until the evaluation of the expression is true or
		// until the context is done.
		logger.Debug(fmt.Sprintf("wait until '%s' is true before print", expression))
		ticker := time.NewTicker(time.Duration(100) * time.Millisecond)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()

				return fmt.Errorf("context done while evaluating '%s': %w", expression, ctx.Err())
			case <-ticker.C:
				var ok bool

				evaluate := chromedp.Evaluate(expression, &ok)
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
