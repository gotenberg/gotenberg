package chromium

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"go.uber.org/multierr"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
)

// FormDataChromiumPdfOptions creates [Options] from the form data. Fallback to
// default value if the considered key is not present.
func FormDataChromiumPdfOptions(ctx *api.Context) (*api.FormData, Options) {
	defaultOptions := DefaultOptions()

	var (
		failOnConsoleExceptions                          bool
		waitDelay                                        time.Duration
		waitWindowStatus                                 string
		waitForExpression                                string
		userAgent                                        string
		extraHttpHeaders                                 map[string]string
		emulatedMediaType                                string
		landscape, printBackground, omitBackground       bool
		scale, paperWidth, paperHeight                   float64
		marginTop, marginBottom, marginLeft, marginRight float64
		pageRanges                                       string
		headerTemplate, footerTemplate                   string
		preferCssPageSize                                bool
	)

	form := ctx.FormData().
		Bool("failOnConsoleExceptions", &failOnConsoleExceptions, defaultOptions.FailOnConsoleExceptions).
		Duration("waitDelay", &waitDelay, defaultOptions.WaitDelay).
		String("waitWindowStatus", &waitWindowStatus, defaultOptions.WaitWindowStatus).
		String("waitForExpression", &waitForExpression, defaultOptions.WaitForExpression).
		String("userAgent", &userAgent, ""). // FIXME: deprecated.
		Custom("extraHttpHeaders", func(value string) error {
			if value == "" {
				extraHttpHeaders = defaultOptions.ExtraHttpHeaders

				return nil
			}

			err := json.Unmarshal([]byte(value), &extraHttpHeaders)
			if err != nil {
				return fmt.Errorf("unmarshal extra HTTP headers: %w", err)
			}

			return nil
		}).
		Custom("emulatedMediaType", func(value string) error {
			if value == "" {
				emulatedMediaType = defaultOptions.EmulatedMediaType

				return nil
			}

			if value != "screen" && value != "print" {
				return fmt.Errorf("wrong value, expected either 'screen', 'print' or empty")
			}

			emulatedMediaType = value

			return nil
		}).
		Bool("landscape", &landscape, defaultOptions.Landscape).
		Bool("printBackground", &printBackground, defaultOptions.PrintBackground).
		Bool("omitBackground", &omitBackground, defaultOptions.OmitBackground).
		Float64("scale", &scale, defaultOptions.Scale).
		Float64("paperWidth", &paperWidth, defaultOptions.PaperWidth).
		Float64("paperHeight", &paperHeight, defaultOptions.PaperHeight).
		Float64("marginTop", &marginTop, defaultOptions.MarginTop).
		Float64("marginBottom", &marginBottom, defaultOptions.MarginBottom).
		Float64("marginLeft", &marginLeft, defaultOptions.MarginLeft).
		Float64("marginRight", &marginRight, defaultOptions.MarginRight).
		String("nativePageRanges", &pageRanges, defaultOptions.PageRanges).
		Content("header.html", &headerTemplate, defaultOptions.HeaderTemplate).
		Content("footer.html", &footerTemplate, defaultOptions.FooterTemplate).
		Bool("preferCssPageSize", &preferCssPageSize, defaultOptions.PreferCssPageSize)

	// FIXME: deprecated.
	if userAgent != "" {
		ctx.Log().Warn("'userAgent' is deprecated; prefer the 'extraHttpHeaders' form field instead")

		if extraHttpHeaders == nil {
			extraHttpHeaders = make(map[string]string)
		}

		extraHttpHeaders["User-Agent"] = userAgent
	}

	options := Options{
		FailOnConsoleExceptions: failOnConsoleExceptions,
		WaitDelay:               waitDelay,
		WaitWindowStatus:        waitWindowStatus,
		WaitForExpression:       waitForExpression,
		ExtraHttpHeaders:        extraHttpHeaders,
		EmulatedMediaType:       emulatedMediaType,
		Landscape:               landscape,
		PrintBackground:         printBackground,
		OmitBackground:          omitBackground,
		Scale:                   scale,
		PaperWidth:              paperWidth,
		PaperHeight:             paperHeight,
		MarginTop:               marginTop,
		MarginBottom:            marginBottom,
		MarginLeft:              marginLeft,
		MarginRight:             marginRight,
		PageRanges:              pageRanges,
		HeaderTemplate:          headerTemplate,
		FooterTemplate:          footerTemplate,
		PreferCssPageSize:       preferCssPageSize,
	}

	return form, options
}

// FormDataChromiumPdfFormats creates [gotenberg.PdfFormats] from the form
// data. Fallback to default value if the considered key is not present.
func FormDataChromiumPdfFormats(ctx *api.Context) gotenberg.PdfFormats {
	var (
		pdfFormat string
		pdfa      string
		pdfua     bool
	)

	ctx.FormData().
		String("pdfFormat", &pdfFormat, "").
		String("pdfa", &pdfa, "").
		Bool("pdfua", &pdfua, false)

	// FIXME: deprecated.
	// pdfa > pdfFormat.
	var actualPdfArchive string

	if pdfFormat != "" {
		ctx.Log().Warn("'pdfFormat' is deprecated; prefer the 'pdfa' form field instead")
		actualPdfArchive = pdfFormat
	}

	if pdfa != "" {
		actualPdfArchive = pdfa
	}

	return gotenberg.PdfFormats{
		PdfA:  actualPdfArchive,
		PdfUa: pdfua,
	}
}

// convertUrlRoute returns an [api.Route] which can convert a URL to PDF.
func convertUrlRoute(chromium Api, engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/chromium/convert/url",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form, options := FormDataChromiumPdfOptions(ctx)
			pdfFormats := FormDataChromiumPdfFormats(ctx)

			var url string
			err := form.
				MandatoryString("url", &url).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = convertUrl(ctx, chromium, engine, url, pdfFormats, options)
			if err != nil {
				return fmt.Errorf("convert URL to PDF: %w", err)
			}

			return nil
		},
	}
}

// convertHtmlRoute returns an [api.Route] which can convert an HTML file to
// PDF.
func convertHtmlRoute(chromium Api, engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/chromium/convert/html",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form, options := FormDataChromiumPdfOptions(ctx)
			pdfFormats := FormDataChromiumPdfFormats(ctx)

			var inputPath string
			err := form.
				MandatoryPath("index.html", &inputPath).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			url := fmt.Sprintf("file://%s", inputPath)
			err = convertUrl(ctx, chromium, engine, url, pdfFormats, options)
			if err != nil {
				return fmt.Errorf("convert HTML to PDF: %w", err)
			}

			return nil
		},
	}
}

// convertMarkdownRoute returns an [api.Route] which can convert markdown files
// to PDF.
func convertMarkdownRoute(chromium Api, engine gotenberg.PdfEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/chromium/convert/markdown",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form, options := FormDataChromiumPdfOptions(ctx)
			pdfFormats := FormDataChromiumPdfFormats(ctx)

			var (
				inputPath     string
				markdownPaths []string
			)

			err := form.
				MandatoryPath("index.html", &inputPath).
				MandatoryPaths([]string{".md"}, &markdownPaths).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			// We have to convert each markdown file referenced in the HTML
			// file to... HTML. Thanks to the "html/template" package, we are
			// able to provide the "toHTML" function which the user may call
			// directly inside the HTML file.

			var markdownFilesNotFoundErr error

			tmpl, err := template.
				New(filepath.Base(inputPath)).
				Funcs(template.FuncMap{
					"toHTML": func(filename string) (template.HTML, error) {
						var path string

						for _, markdownPath := range markdownPaths {
							markdownFilename := filepath.Base(markdownPath)

							if filename == markdownFilename {
								path = markdownPath
								break
							}
						}

						if path == "" {
							markdownFilesNotFoundErr = multierr.Append(
								markdownFilesNotFoundErr,
								fmt.Errorf("'%s'", filename),
							)

							return "", nil
						}

						b, err := os.ReadFile(path)
						if err != nil {
							return "", fmt.Errorf("read markdown file '%s': %w", filename, err)
						}

						unsafe := blackfriday.Run(b)
						sanitized := bluemonday.UGCPolicy().SanitizeBytes(unsafe)

						// #nosec
						return template.HTML(sanitized), nil
					},
				}).ParseFiles(inputPath)
			if err != nil {
				return fmt.Errorf("parse template file: %w", err)
			}

			var buffer bytes.Buffer

			err = tmpl.Execute(&buffer, &struct{}{})
			if err != nil {
				return fmt.Errorf("execute template: %w", err)
			}

			if markdownFilesNotFoundErr != nil {
				return api.WrapError(
					fmt.Errorf("markdown files not found: %w", markdownFilesNotFoundErr),
					api.NewSentinelHttpError(
						http.StatusBadRequest,
						fmt.Sprintf("Markdown file(s) not found: %s", markdownFilesNotFoundErr),
					),
				)
			}

			inputPath = ctx.GeneratePath(".html")

			err = os.WriteFile(inputPath, buffer.Bytes(), 0o600)
			if err != nil {
				return fmt.Errorf("write template result: %w", err)
			}

			url := fmt.Sprintf("file://%s", inputPath)

			err = convertUrl(ctx, chromium, engine, url, pdfFormats, options)
			if err != nil {
				return fmt.Errorf("convert markdown to PDF: %w", err)
			}

			return nil
		},
	}
}

// convertUrl is a stub which is called by the other methods of this file.
func convertUrl(ctx *api.Context, chromium Api, engine gotenberg.PdfEngine, url string, pdfFormats gotenberg.PdfFormats, options Options) error {
	outputPath := ctx.GeneratePath(".pdf")

	err := chromium.Pdf(ctx, ctx.Log(), url, outputPath, options)
	if err != nil {

		if errors.Is(err, ErrUrlNotAuthorized) {
			return api.WrapError(
				fmt.Errorf("convert to PDF: %w", err),
				api.NewSentinelHttpError(
					http.StatusForbidden,
					fmt.Sprintf("'%s' does not match the authorized URLs", url),
				),
			)
		}

		if errors.Is(err, ErrOmitBackgroundWithoutPrintBackground) {
			return api.WrapError(
				fmt.Errorf("convert to PDF: %w", err),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					"omitBackground requires printBackground set to true",
				),
			)
		}

		if errors.Is(err, ErrInvalidEvaluationExpression) {
			if options.WaitForExpression == "" {
				// We do not expect the 'waitWindowStatus' form field to return
				// an ErrInvalidEvaluationExpression error. In such a scenario,
				// we return a 500.
				return fmt.Errorf("convert to PDF: %w", err)
			}

			return api.WrapError(
				fmt.Errorf("convert to PDF: %w", err),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					fmt.Sprintf("The expression '%s' (waitForExpression) returned an exception or undefined", options.WaitForExpression),
				),
			)
		}

		if errors.Is(err, ErrInvalidPrinterSettings) {
			return api.WrapError(
				fmt.Errorf("convert to PDF: %w", err),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					"Chromium does not handle the provided settings; please check for aberrant form values",
				),
			)
		}

		if errors.Is(err, ErrPageRangesSyntaxError) {
			return api.WrapError(
				fmt.Errorf("convert to PDF: %w", err),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					fmt.Sprintf("Chromium does not handle the page ranges '%s' (nativePageRanges)", options.PageRanges),
				),
			)
		}

		if errors.Is(err, ErrConsoleExceptions) {
			return api.WrapError(
				fmt.Errorf("convert to PDF: %w", err),
				api.NewSentinelHttpError(
					http.StatusConflict,
					fmt.Sprintf("Chromium console exceptions:\n %s", strings.ReplaceAll(err.Error(), ErrConsoleExceptions.Error(), "")),
				),
			)
		}

		return fmt.Errorf("convert to PDF: %w", err)
	}

	// So far so good, the URL has been converted to PDF.
	// Now, let's check if the client want to convert the resulting PDF
	// to specific formats.
	zeroValued := gotenberg.PdfFormats{}
	if pdfFormats != zeroValued {
		convertInputPath := outputPath
		convertOutputPath := ctx.GeneratePath(".pdf")

		err = engine.Convert(ctx, ctx.Log(), pdfFormats, convertInputPath, convertOutputPath)

		if err != nil {
			if errors.Is(err, gotenberg.ErrPdfFormatNotSupported) {
				return api.WrapError(
					fmt.Errorf("convert PDF: %w", err),
					api.NewSentinelHttpError(
						http.StatusBadRequest,
						fmt.Sprintf("At least one PDF engine does not handle one of the PDF format in '%+v', while other have failed to convert for other reasons", pdfFormats),
					),
				)
			}

			return fmt.Errorf("convert PDF: %w", err)
		}

		// Important: the output path is now the converted file.
		outputPath = convertOutputPath
	}

	err = ctx.AddOutputPaths(outputPath)
	if err != nil {
		return fmt.Errorf("add output path: %w", err)
	}

	return nil
}
