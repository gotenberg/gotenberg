package libreoffice

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gotenberg/gotenberg/v7/pkg/gotenberg"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/api"
	"github.com/gotenberg/gotenberg/v7/pkg/modules/libreoffice/uno"
	"github.com/labstack/echo/v4"
)

// convertRoute returns an api.Route which can convert LibreOffice documents
// to PDF, or in some cases, html.
func convertRoute(unoAPI uno.API, engine gotenberg.PDFEngine) api.Route {
	return api.Route{
		Method:      http.MethodPost,
		Path:        "/forms/libreoffice/convert",
		IsMultipart: true,
		Handler: func(c echo.Context) error {
			ctx := c.Get("context").(*api.Context)

			// Let's get the data from the form and validate them.
			var (
				inputPaths         []string
				landscape          bool
				nativePageRanges   string
				nativePDFA1aFormat bool
				nativePDFformat    string
				PDFformat          string
				htmlFormat         bool
				merge              bool
			)

			err := ctx.FormData().
				MandatoryPaths(unoAPI.Extensions(), &inputPaths).
				Bool("landscape", &landscape, false).
				String("nativePageRanges", &nativePageRanges, "").
				Bool("nativePdfA1aFormat", &nativePDFA1aFormat, false).
				String("nativePdfFormat", &nativePDFformat, "").
				String("pdfFormat", &PDFformat, "").
				Bool("htmlFormat", &htmlFormat, false).
				Bool("merge", &merge, false).
				Validate()

			if err != nil {
				return fmt.Errorf("validate form data: %w", err)
			}

			if nativePDFA1aFormat {
				ctx.Log().Warn("'nativePdfA1aFormat' is deprecated; prefer 'nativePdfFormat' or 'pdfFormat' form fields instead")
			}

			if nativePDFA1aFormat && nativePDFformat != "" {
				return api.WrapError(
					errors.New("got both 'nativePdfFormat' and 'nativePdfA1aFormat' form fields"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Both 'nativePdfFormat' and 'nativePdfA1aFormat' form fields are provided"),
				)
			}

			if nativePDFA1aFormat && PDFformat != "" {
				return api.WrapError(
					errors.New("got both 'pdfFormat' and 'nativePdfA1aFormat' form fields"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Both 'pdfFormat' and 'nativePdfA1aFormat' form fields are provided"),
				)
			}

			if nativePDFformat != "" && PDFformat != "" {
				return api.WrapError(
					errors.New("got both 'pdfFormat' and 'nativePdfFormat' form fields"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Both 'pdfFormat' and 'nativePdfFormat' form fields are provided"),
				)
			}

			if nativePDFA1aFormat {
				nativePDFformat = gotenberg.FormatPDFA1a
			}

			// Check for conflicts with HTML output flag.
			if htmlFormat && merge && len(inputPaths) > 1 {
				return api.WrapError(
					errors.New("unable to merge multiple files with htmlFormat"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Unable to merge multiple files using htmlFormat"),
				)
			}

			if htmlFormat && nativePDFA1aFormat {
				return api.WrapError(
					errors.New("got both 'htmlFormat' and 'nativePdfA1aFormat' form fields"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Both 'htmlFormat' and 'nativePdfA1aFormat' form fields are provided"),
				)
			}
			if htmlFormat && PDFformat != "" {
				return api.WrapError(
					errors.New("got both 'htmlFormat' and 'PDFformat' form fields"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Both 'htmlFormat' and 'PDFformat' form fields are provided"),
				)
			}
			if htmlFormat && nativePageRanges != "" {
				return api.WrapError(
					errors.New("got both 'htmlFormat' and 'nativePageRanges' form fields"),
					api.NewSentinelHTTPError(http.StatusBadRequest, "Both 'htmlFormat' and 'nativePageRanges' form fields are provided"),
				)
			}

			// Alright, let's convert each document.
			outputPaths := make([]string, len(inputPaths))
			extension := ".pdf"
			if htmlFormat {
				extension = ".html"
			}
			for i, inputPath := range inputPaths {
				outputPaths[i] = ctx.GeneratePath(extension)

				options := uno.Options{
					Landscape:  landscape,
					PageRanges: nativePageRanges,
					PDFformat:  nativePDFformat,
					HTMLformat: htmlFormat,
				}

				err = unoAPI.Convert(ctx, ctx.Log(), inputPath, outputPaths[i], options)

				if err != nil {
					if errors.Is(err, uno.ErrMalformedPageRanges) {
						return api.WrapError(
							fmt.Errorf("convert to %s: %w", extension, err),
							api.NewSentinelHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed page ranges '%s' (nativePageRanges)", options.PageRanges)),
						)
					}

					return fmt.Errorf("convert to %s: %w", extension, err)
				}
			}

			// So far so good, let's check if we have to merge the PDFs. Quick
			// win: if there is only one PDF, skip this step.

			if len(outputPaths) > 1 && merge {
				outputPath := ctx.GeneratePath(".pdf")

				err = engine.Merge(ctx, ctx.Log(), outputPaths, outputPath)
				if err != nil {
					return fmt.Errorf("merge PDFs: %w", err)
				}

				// Now, let's check if the client want to convert this result
				// PDF to a specific PDF format.

				// Note: nativePdfA1aFormat/nativePdfFormat have not been
				// specified if PDFformat is not empty.

				if PDFformat != "" {
					convertInputPath := outputPath
					convertOutputPath := ctx.GeneratePath(".pdf")

					err = engine.Convert(ctx, ctx.Log(), PDFformat, convertInputPath, convertOutputPath)

					if err != nil {
						if errors.Is(err, gotenberg.ErrPDFFormatNotAvailable) {
							return api.WrapError(
								fmt.Errorf("convert PDF: %w", err),
								api.NewSentinelHTTPError(
									http.StatusBadRequest,
									fmt.Sprintf("At least one PDF engine does not handle the PDF format '%s' (pdfFormat), while other have failed to convert for other reasons", PDFformat),
								),
							)
						}

						return fmt.Errorf("convert PDF: %w", err)
					}

					// Important: the output path is now the converted file.
					outputPath = convertOutputPath
				}

				// Last but not least, add the output path to the context so that
				// the API is able to send it as a response to the client.

				err = ctx.AddOutputPaths(outputPath)
				if err != nil {
					return fmt.Errorf("add output path: %w", err)
				}

				return nil
			}

			// Ok, we don't have to merge the PDFs. Let's check if the client
			// want to convert each PDF to a specific PDF format.

			// Note: nativePdfA1aFormat/nativePdfFormat have not been
			// specified if PDFformat is not empty.

			if PDFformat != "" {
				convertOutputPaths := make([]string, len(outputPaths))

				for i, outputPath := range outputPaths {
					convertInputPath := outputPath
					convertOutputPaths[i] = ctx.GeneratePath(".pdf")

					err = engine.Convert(ctx, ctx.Log(), PDFformat, convertInputPath, convertOutputPaths[i])

					if err != nil {
						if errors.Is(err, gotenberg.ErrPDFFormatNotAvailable) {
							return api.WrapError(
								fmt.Errorf("convert PDF: %w", err),
								api.NewSentinelHTTPError(
									http.StatusBadRequest,
									fmt.Sprintf("At least one PDF engine does not handle the PDF format '%s' (pdfFormat), while other have failed to convert for other reasons", PDFformat),
								),
							)
						}

						return fmt.Errorf("convert PDF: %w", err)
					}

				}

				// Important: the output paths are now the converted files.
				outputPaths = convertOutputPaths
			}

			// Last but not least, add the output paths to the context so that
			// the API is able to send them as a response to the client.

			err = ctx.AddOutputPaths(outputPaths...)
			if err != nil {
				return fmt.Errorf("add output paths: %w", err)
			}

			return nil
		},
	}
}
