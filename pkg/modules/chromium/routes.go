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
	"strconv"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"go.uber.org/multierr"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/api"
	"github.com/gotenberg/gotenberg/v8/pkg/modules/pdfengines"
)

// FormDataChromiumOptions creates [Options] from the form data. Fallback to
// the default value if the considered key is not present.
func FormDataChromiumOptions(ctx *api.Context) (*api.FormData, Options) {
	defaultOptions := DefaultOptions()

	var (
		skipNetworkIdleEvent          bool
		failOnHttpStatusCodes         []int64
		failOnResourceHttpStatusCodes []int64
		failOnResourceLoadingFailed   bool
		failOnConsoleExceptions       bool
		waitDelay                     time.Duration
		waitWindowStatus              string
		waitForExpression             string
		cookies                       []Cookie
		userAgent                     string
		extraHttpHeaders              []ExtraHttpHeader
		emulatedMediaType             string
		omitBackground                bool
	)

	form := ctx.FormData().
		Bool("skipNetworkIdleEvent", &skipNetworkIdleEvent, defaultOptions.SkipNetworkIdleEvent).
		Custom("failOnHttpStatusCodes", func(value string) error {
			if value == "" {
				failOnHttpStatusCodes = defaultOptions.FailOnHttpStatusCodes
				return nil
			}

			err := json.Unmarshal([]byte(value), &failOnHttpStatusCodes)
			if err != nil {
				return fmt.Errorf("unmarshal failOnHttpStatusCodes: %w", err)
			}

			return nil
		}).
		Custom("failOnResourceHttpStatusCodes", func(value string) error {
			if value == "" {
				failOnResourceHttpStatusCodes = defaultOptions.FailOnResourceHttpStatusCodes
				return nil
			}

			err := json.Unmarshal([]byte(value), &failOnResourceHttpStatusCodes)
			if err != nil {
				return fmt.Errorf("unmarshal failOnResourceHttpStatusCodes: %w", err)
			}

			return nil
		}).
		Bool("failOnResourceLoadingFailed", &failOnResourceLoadingFailed, defaultOptions.FailOnResourceLoadingFailed).
		Bool("failOnConsoleExceptions", &failOnConsoleExceptions, defaultOptions.FailOnConsoleExceptions).
		Duration("waitDelay", &waitDelay, defaultOptions.WaitDelay).
		String("waitWindowStatus", &waitWindowStatus, defaultOptions.WaitWindowStatus).
		String("waitForExpression", &waitForExpression, defaultOptions.WaitForExpression).
		Custom("cookies", func(value string) error {
			if value == "" {
				cookies = defaultOptions.Cookies
				return nil
			}

			err := json.Unmarshal([]byte(value), &cookies)
			if err != nil {
				return fmt.Errorf("unmarshal cookies: %w", err)
			}

			for i, cookie := range cookies {
				if strings.TrimSpace(cookie.Name) == "" || strings.TrimSpace(cookie.Value) == "" || strings.TrimSpace(cookie.Domain) == "" {
					err = multierr.Append(err, fmt.Errorf("cookie %d must have its name, value and domain set", i))
				}
			}

			return err
		}).
		String("userAgent", &userAgent, defaultOptions.UserAgent).
		Custom("extraHttpHeaders", func(value string) error {
			if value == "" {
				extraHttpHeaders = defaultOptions.ExtraHttpHeaders
				return nil
			}

			var headers map[string]string
			err := json.Unmarshal([]byte(value), &headers)
			if err != nil {
				return fmt.Errorf("unmarshal extraHttpHeaders: %w", err)
			}

			for k, v := range headers {
				var scope string
				var valueTokens []string
				var invalidScopeToken bool

				tokens := strings.Split(v, ";")
				for _, token := range tokens {
					if strings.HasPrefix(strings.ToLower(strings.TrimSpace(token)), "scope") {
						tokenNoSpaces := strings.Join(strings.Fields(token), "")
						parts := strings.SplitN(tokenNoSpaces, "=", 2)

						if len(parts) == 2 && strings.ToLower(parts[0]) == "scope" && parts[1] != "" {
							scope = parts[1]
						} else {
							err = multierr.Append(err, fmt.Errorf("invalid scope '%s' for header '%s'", scope, k))
							invalidScopeToken = true
							break
						}
					} else {
						if token != "" {
							valueTokens = append(valueTokens, token)
						}
					}
				}

				if invalidScopeToken {
					continue
				}

				var scopeRegexp *regexp2.Regexp
				if len(scope) > 0 {
					p, errCompile := regexp2.Compile(scope, 0)
					if errCompile != nil {
						err = multierr.Append(err, fmt.Errorf("invalid scope regex pattern for header '%s': %w", k, errCompile))
						continue
					}
					scopeRegexp = p
				}

				extraHttpHeaders = append(extraHttpHeaders, ExtraHttpHeader{
					Name:  k,
					Value: strings.Join(valueTokens, "; "),
					Scope: scopeRegexp,
				})
			}

			return err
		}).
		Custom("emulatedMediaType", func(value string) error {
			if value == "" {
				emulatedMediaType = defaultOptions.EmulatedMediaType
				return nil
			}

			if value != "screen" && value != "print" {
				return errors.New("wrong value, expected either 'screen', 'print' or empty")
			}

			emulatedMediaType = value

			return nil
		}).
		Bool("omitBackground", &omitBackground, defaultOptions.OmitBackground)

	options := Options{
		SkipNetworkIdleEvent:          skipNetworkIdleEvent,
		FailOnHttpStatusCodes:         failOnHttpStatusCodes,
		FailOnResourceHttpStatusCodes: failOnResourceHttpStatusCodes,
		FailOnResourceLoadingFailed:   failOnResourceLoadingFailed,
		FailOnConsoleExceptions:       failOnConsoleExceptions,
		WaitDelay:                     waitDelay,
		WaitWindowStatus:              waitWindowStatus,
		WaitForExpression:             waitForExpression,
		Cookies:                       cookies,
		UserAgent:                     userAgent,
		ExtraHttpHeaders:              extraHttpHeaders,
		EmulatedMediaType:             emulatedMediaType,
		OmitBackground:                omitBackground,
	}

	return form, options
}

// FormDataChromiumPdfOptions creates [PdfOptions] from the form data. Fallback to
// the default value if the considered key is not present.
func FormDataChromiumPdfOptions(ctx *api.Context) (*api.FormData, PdfOptions) {
	form, options := FormDataChromiumOptions(ctx)
	defaultPdfOptions := DefaultPdfOptions()

	var (
		landscape, printBackground, singlePage           bool
		scale, paperWidth, paperHeight                   float64
		marginTop, marginBottom, marginLeft, marginRight float64
		pageRanges                                       string
		headerTemplate, footerTemplate                   string
		preferCssPageSize                                bool
		generateDocumentOutline                          bool
	)

	form.
		Bool("landscape", &landscape, defaultPdfOptions.Landscape).
		Bool("printBackground", &printBackground, defaultPdfOptions.PrintBackground).
		Float64("scale", &scale, defaultPdfOptions.Scale).
		Bool("singlePage", &singlePage, defaultPdfOptions.SinglePage).
		Inches("paperWidth", &paperWidth, defaultPdfOptions.PaperWidth).
		Inches("paperHeight", &paperHeight, defaultPdfOptions.PaperHeight).
		Inches("marginTop", &marginTop, defaultPdfOptions.MarginTop).
		Inches("marginBottom", &marginBottom, defaultPdfOptions.MarginBottom).
		Inches("marginLeft", &marginLeft, defaultPdfOptions.MarginLeft).
		Inches("marginRight", &marginRight, defaultPdfOptions.MarginRight).
		String("nativePageRanges", &pageRanges, defaultPdfOptions.PageRanges).
		Content("header.html", &headerTemplate, defaultPdfOptions.HeaderTemplate).
		Content("footer.html", &footerTemplate, defaultPdfOptions.FooterTemplate).
		Bool("preferCssPageSize", &preferCssPageSize, defaultPdfOptions.PreferCssPageSize).
		Bool("generateDocumentOutline", &generateDocumentOutline, defaultPdfOptions.GenerateDocumentOutline)

	pdfOptions := PdfOptions{
		Options:                 options,
		Landscape:               landscape,
		PrintBackground:         printBackground,
		Scale:                   scale,
		SinglePage:              singlePage,
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
		GenerateDocumentOutline: generateDocumentOutline,
	}

	return form, pdfOptions
}

// FormDataChromiumScreenshotOptions creates [ScreenshotOptions] from the form
// data. Fallback to the default value if the considered key is not present.
func FormDataChromiumScreenshotOptions(ctx *api.Context) (*api.FormData, ScreenshotOptions) {
	form, options := FormDataChromiumOptions(ctx)
	defaultScreenshotOptions := DefaultScreenshotOptions()

	var (
		width, height    int
		clip             bool
		format           string
		quality          int
		optimizeForSpeed bool
	)

	form.
		Int("width", &width, defaultScreenshotOptions.Width).
		Int("height", &height, defaultScreenshotOptions.Height).
		Bool("clip", &clip, defaultScreenshotOptions.Clip).
		Custom("format", func(value string) error {
			if value == "" {
				format = defaultScreenshotOptions.Format
				return nil
			}

			if value != "png" && value != "jpeg" && value != "webp" {
				return fmt.Errorf("wrong value, expected either 'png', 'jpeg' or 'webp'")
			}

			format = value

			return nil
		}).
		Custom("quality", func(value string) error {
			if value == "" {
				quality = defaultScreenshotOptions.Quality
				return nil
			}

			intValue, err := strconv.Atoi(value)
			if err != nil {
				return err
			}

			if intValue < 0 {
				return errors.New("value is negative")
			}

			if intValue > 100 {
				return errors.New("value is superior to 100")
			}

			quality = intValue
			return nil
		}).
		Bool("optimizeForSpeed", &optimizeForSpeed, defaultScreenshotOptions.OptimizeForSpeed)

	screenshotOptions := ScreenshotOptions{
		Options:          options,
		Width:            width,
		Height:           height,
		Clip:             clip,
		Format:           format,
		Quality:          quality,
		OptimizeForSpeed: optimizeForSpeed,
	}

	return form, screenshotOptions
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
			mode := pdfengines.FormDataPdfSplitMode(form, false)
			pdfFormats := pdfengines.FormDataPdfFormats(form)
			metadata := pdfengines.FormDataPdfMetadata(form, false)

			var url string
			err := form.
				MandatoryString("url", &url).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = convertUrl(ctx, chromium, engine, url, options, mode, pdfFormats, metadata)
			if err != nil {
				return fmt.Errorf("convert URL to PDF: %w", err)
			}

			return nil
		},
	}
}

// screenshotUrlRoute returns an [api.Route] which can take a screenshot from a
// URL.
func screenshotUrlRoute(chromium Api) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/chromium/screenshot/url",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form, options := FormDataChromiumScreenshotOptions(ctx)

			var url string
			err := form.
				MandatoryString("url", &url).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			err = screenshotUrl(ctx, chromium, url, options)
			if err != nil {
				return fmt.Errorf("URL screenshot: %w", err)
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
			mode := pdfengines.FormDataPdfSplitMode(form, false)
			pdfFormats := pdfengines.FormDataPdfFormats(form)
			metadata := pdfengines.FormDataPdfMetadata(form, false)

			var inputPath string
			err := form.
				MandatoryPath("index.html", &inputPath).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			url := fmt.Sprintf("file://%s", inputPath)
			err = convertUrl(ctx, chromium, engine, url, options, mode, pdfFormats, metadata)
			if err != nil {
				return fmt.Errorf("convert HTML to PDF: %w", err)
			}

			return nil
		},
	}
}

// screenshotHtmlRoute returns an [api.Route] which can take a screenshot from
// an HTML file.
func screenshotHtmlRoute(chromium Api) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/chromium/screenshot/html",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form, options := FormDataChromiumScreenshotOptions(ctx)

			var inputPath string
			err := form.
				MandatoryPath("index.html", &inputPath).
				Validate()
			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			url := fmt.Sprintf("file://%s", inputPath)
			err = screenshotUrl(ctx, chromium, url, options)
			if err != nil {
				return fmt.Errorf("HTML screenshot: %w", err)
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
			mode := pdfengines.FormDataPdfSplitMode(form, false)
			pdfFormats := pdfengines.FormDataPdfFormats(form)
			metadata := pdfengines.FormDataPdfMetadata(form, false)

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

			url, err := markdownToHtml(ctx, inputPath, markdownPaths)
			if err != nil {
				return fmt.Errorf("transform markdown file(s) to HTML: %w", err)
			}

			err = convertUrl(ctx, chromium, engine, url, options, mode, pdfFormats, metadata)
			if err != nil {
				return fmt.Errorf("convert markdown to PDF: %w", err)
			}

			return nil
		},
	}
}

// screenshotMarkdownRoute returns an [api.Route] which can take a screenshot
// from Markdown files.
func screenshotMarkdownRoute(chromium Api) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/chromium/screenshot/markdown",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)
			form, options := FormDataChromiumScreenshotOptions(ctx)

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

			url, err := markdownToHtml(ctx, inputPath, markdownPaths)
			if err != nil {
				return fmt.Errorf("transform markdown file(s) to HTML: %w", err)
			}

			err = screenshotUrl(ctx, chromium, url, options)
			if err != nil {
				return fmt.Errorf("markdown screenshot: %w", err)
			}

			return nil
		},
	}
}

func markdownToHtml(ctx *api.Context, inputPath string, markdownPaths []string) (string, error) {
	// We have to convert each Markdown file referenced in the HTML
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
		return "", fmt.Errorf("parse template file: %w", err)
	}

	var buffer bytes.Buffer

	err = tmpl.Execute(&buffer, &struct{}{})
	if err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	if markdownFilesNotFoundErr != nil {
		return "", api.WrapError(
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
		return "", fmt.Errorf("write template result: %w", err)
	}

	return fmt.Sprintf("file://%s", inputPath), nil
}

func convertUrl(ctx *api.Context, chromium Api, engine gotenberg.PdfEngine, url string, options PdfOptions, mode gotenberg.SplitMode, pdfFormats gotenberg.PdfFormats, metadata map[string]interface{}) error {
	outputPath := ctx.GeneratePath(".pdf")

	err := chromium.Pdf(ctx, ctx.Log(), url, outputPath, options)
	err = handleChromiumError(err, options.Options)
	if err != nil {
		if errors.Is(err, ErrOmitBackgroundWithoutPrintBackground) {
			return api.WrapError(
				fmt.Errorf("convert to PDF: %w", err),
				api.NewSentinelHttpError(
					http.StatusBadRequest,
					"omitBackground requires printBackground set to true",
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

		return fmt.Errorf("convert to PDF: %w", err)
	}

	outputPaths, err := pdfengines.SplitPdfStub(ctx, engine, mode, []string{outputPath})
	if err != nil {
		return fmt.Errorf("split PDF: %w", err)
	}

	convertOutputPaths, err := pdfengines.ConvertStub(ctx, engine, pdfFormats, outputPaths)
	if err != nil {
		return fmt.Errorf("convert PDF(s): %w", err)
	}

	err = pdfengines.WriteMetadataStub(ctx, engine, metadata, convertOutputPaths)
	if err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	zeroValuedSplitMode := gotenberg.SplitMode{}
	zeroValuedPdfFormats := gotenberg.PdfFormats{}
	if mode != zeroValuedSplitMode && pdfFormats != zeroValuedPdfFormats {
		// The PDF has been split and split parts have been converted to a
		// specific format. We want to keep the split naming.
		for i, convertOutputPath := range convertOutputPaths {
			err = ctx.Rename(convertOutputPath, outputPaths[i])
			if err != nil {
				return fmt.Errorf("rename output path: %w", err)
			}
		}
	} else {
		outputPaths = convertOutputPaths
	}

	err = ctx.AddOutputPaths(outputPaths...)
	if err != nil {
		return fmt.Errorf("add output paths: %w", err)
	}

	return nil
}

func screenshotUrl(ctx *api.Context, chromium Api, url string, options ScreenshotOptions) error {
	ext := fmt.Sprintf(".%s", options.Format)
	outputPath := ctx.GeneratePath(ext)

	err := chromium.Screenshot(ctx, ctx.Log(), url, outputPath, options)
	err = handleChromiumError(err, options.Options)
	if err != nil {
		return fmt.Errorf("screenshot: %w", err)
	}

	err = ctx.AddOutputPaths(outputPath)
	if err != nil {
		return fmt.Errorf("add output path: %w", err)
	}

	return nil
}

func handleChromiumError(err error, options Options) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, ErrInvalidEvaluationExpression) {
		if options.WaitForExpression == "" {
			// We do not expect the 'waitWindowStatus' form field to return
			// an ErrInvalidEvaluationExpression error. In such a scenario,
			// we return a 500.
			return err
		}

		return api.WrapError(
			err,
			api.NewSentinelHttpError(
				http.StatusBadRequest,
				fmt.Sprintf("The expression '%s' (waitForExpression) returned an exception or undefined", options.WaitForExpression),
			),
		)
	}

	if errors.Is(err, ErrInvalidHttpStatusCode) {
		return api.WrapError(
			err,
			api.NewSentinelHttpError(
				http.StatusConflict,
				fmt.Sprintf("Invalid HTTP status code from the main page: %s", strings.ReplaceAll(err.Error(), fmt.Sprintf(": %s", ErrInvalidHttpStatusCode.Error()), "")),
			),
		)
	}

	if errors.Is(err, ErrInvalidResourceHttpStatusCode) {
		return api.WrapError(
			err,
			api.NewSentinelHttpError(
				http.StatusConflict,
				fmt.Sprintf("Invalid HTTP status code from resources:\n%s", strings.ReplaceAll(err.Error(), fmt.Sprintf(": %s", ErrInvalidResourceHttpStatusCode.Error()), "")),
			),
		)
	}

	if errors.Is(err, ErrConsoleExceptions) {
		return api.WrapError(
			err,
			api.NewSentinelHttpError(
				http.StatusConflict,
				fmt.Sprintf("Chromium console exceptions:\n%s", strings.ReplaceAll(err.Error(), ErrConsoleExceptions.Error(), "")),
			),
		)
	}

	if errors.Is(err, ErrLoadingFailed) {
		return api.WrapError(
			err,
			api.NewSentinelHttpError(
				http.StatusBadRequest,
				fmt.Sprintf("Chromium returned %v", err),
			),
		)
	}

	if errors.Is(err, ErrResourceLoadingFailed) {
		return api.WrapError(
			err,
			api.NewSentinelHttpError(
				http.StatusConflict,
				fmt.Sprintf("Chromium failed to load resources: %v", strings.ReplaceAll(err.Error(), fmt.Sprintf(": %s", ErrResourceLoadingFailed.Error()), "")),
			),
		)
	}

	return err
}
